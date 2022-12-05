package htracker

import (
	"crypto/md5"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/export"
	"golang.org/x/exp/slog"
)

// Site contains the meta data necessary to describe a web site to be scraped.
type Site struct {
	URL         string
	Filter      string
	ContentType string
	Interval    time.Duration
}

// Equal is a method for comparing identifying metadata for this site with the given site.
// The combination of URL, Filter and ContentType must be equal for sites to be equal.
// It is not meant to compare the _content_ of web sites.
func (s *Site) Equals(site *Site) bool {
	return s.URL == site.URL && s.Filter == site.Filter && s.ContentType == site.ContentType
}

// SiteArchive is holding metadata, checksum and content of a scraped web site.
type SiteArchive struct {
	Site        *Site
	LastUpdated time.Time
	LastChecked time.Time
	Content     []byte
	Checksum    string
	Diff        string
}

// Scraper is used to scrape web sites.
type Scraper struct {
	*geziyor.Geziyor

	Sites  []*Site
	Logger *slog.Logger

	/*** Geziyor Opts ***/

	// AllowedDomains is domains that are allowed to make requests
	// If empty, any domain is allowed
	AllowedDomains []string

	// Chrome headless browser WS endpoint.
	// If you want to run your own Chrome browser runner, provide its endpoint in here
	// For example: ws://localhost:3000
	BrowserEndpoint string

	// Concurrent requests limit
	ConcurrentRequests int

	// Concurrent requests per domain limit. Uses request.URL.Host
	// Subdomains are different than top domain
	ConcurrentRequestsPerDomain int

	// For extracting data
	Exporters []export.Exporter

	// Max body reading size in bytes. Default: 1GB
	MaxBodySize int64

	// RequestsPerSecond limits requests that is made per seconds. Default: No limit
	RequestsPerSecond float64

	// Timeout is global request timeout
	Timeout time.Duration

	// User Agent.
	// Default: "HTracker/Geziyor 1.0"
	UserAgent string
}

// newParseFunc is returning a new parser func, setup to parse the given site
// and send the results as siteArchive to the Exports channel.
func newParseFunc(site *Site, logger *slog.Logger) func(*geziyor.Geziyor, *client.Response) {
	return func(g *geziyor.Geziyor, r *client.Response) {

		var content []byte

		if r.Response.StatusCode >= 400 {
			logger.Warn("got error status code", "code", r.Response.StatusCode, "url", site.URL)
			return
		}

		if site.Filter == "" {
			content = r.Body
		} else {
			if r.HTMLDoc != nil {
				content = []byte(r.HTMLDoc.Find(site.Filter).Text())
			} else {
				exp, err := regexp.Compile(site.Filter)
				if err != nil {
					logger.Error("ParseFunc failed to compile regexp", err, "regexp", site.Filter, "site", site.URL)
					return
				}
				content = exp.Find(r.Body)
			}
		}

		sa := &SiteArchive{
			Site:        site,
			LastChecked: time.Now(),
			Content:     content,
			Checksum:    fmt.Sprintf("%x", md5.Sum([]byte(content))),
		}

		g.Exports <- sa
	}
}

// NewScraper is returning a new Scraper to scrape given web sites.
func NewScraper(sites []*Site, opts ...ScraperOpt) *Scraper {

	scraper := &Scraper{
		Sites:     sites,
		Logger:    slog.New(slog.NewTextHandler(os.Stdout).WithGroup("scraper")),
		UserAgent: "HTracker/Geziyor 1.0",
	}

	for _, o := range opts {
		o(scraper)
	}

	gcfg := geziyor.Options{
		AllowedDomains:              scraper.AllowedDomains,
		BrowserEndpoint:             scraper.BrowserEndpoint,
		ConcurrentRequests:          scraper.ConcurrentRequests,
		ConcurrentRequestsPerDomain: scraper.ConcurrentRequestsPerDomain,
		Exporters:                   scraper.Exporters,
		MaxBodySize:                 scraper.MaxBodySize,
		RequestsPerSecond:           scraper.RequestsPerSecond,
		Timeout:                     scraper.Timeout,
		UserAgent:                   scraper.UserAgent,
	}

	if scraper.BrowserEndpoint != "" {
		gcfg.StartRequestsFunc = func(g *geziyor.Geziyor) {
			for _, s := range scraper.Sites {
				// using external chrome browser for rendering java script
				g.GetRendered(s.URL, newParseFunc(s, scraper.Logger))
			}
		}
	} else {
		gcfg.StartRequestsFunc = func(g *geziyor.Geziyor) {
			for _, s := range scraper.Sites {
				// directly scrape the plain web site content without rendering JS
				g.Get(s.URL, newParseFunc(s, scraper.Logger))
			}
		}
	}

	scraper.Geziyor = geziyor.NewGeziyor(&gcfg)

	return scraper
}

type ScraperOpt func(*Scraper)

func WithAllowedDomains(domains []string) ScraperOpt {
	return func(s *Scraper) {
		s.AllowedDomains = domains
	}
}

func WithBrowserEndpoint(endpoint string) ScraperOpt {
	return func(s *Scraper) {
		s.BrowserEndpoint = endpoint
	}
}

func WithExporters(exporters []Exporter) ScraperOpt {
	return func(s *Scraper) {
		s.Exporters = exporters
	}
}

func WithLogger(logger *slog.Logger) ScraperOpt {
	return func(s *Scraper) {
		s.Logger = logger
	}
}
