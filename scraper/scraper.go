package scraper

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/export"
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/exporter"
	"golang.org/x/exp/slog"
)

// Scraper is used to scrape web sites.
type Scraper struct {
	*geziyor.Geziyor

	Sites  []*htracker.Site
	Logger *slog.Logger

	/*** Geziyor Opts ***/

	// AllowedDomains is domains that are allowed to make requests
	// If empty, any domain is allowed
	AllowedDomains []string

	// Chrome headless browser WS endpoint.
	// If you want to run your own Chrome browser runner, provide its endpoint in here
	// For example: ws://localhost:3000
	BrowserEndpoint string

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
func newParseFunc(site *htracker.Site, logger *slog.Logger) func(*geziyor.Geziyor, *client.Response) {
	return func(g *geziyor.Geziyor, r *client.Response) {

		var content []byte

		if r.Response.StatusCode >= http.StatusBadRequest {
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

		sa := &htracker.SiteContent{
			Site:        site,
			LastChecked: time.Now(),
			Content:     content,
			Checksum:    fmt.Sprintf("%x", sha256.Sum256([]byte(content))),
		}

		g.Exports <- sa
	}
}

// NewScraper is returning a new Scraper to scrape given web sites.
func NewScraper(sites []*htracker.Site, opts ...Opt) *Scraper {

	scraper := &Scraper{
		Sites:     sites,
		Logger:    slog.Default(),
		UserAgent: "HTracker/Geziyor 1.0",
	}

	for _, o := range opts {
		o(scraper)
	}

	gcfg := geziyor.Options{
		AllowedDomains:    scraper.AllowedDomains,
		BrowserEndpoint:   scraper.BrowserEndpoint,
		Exporters:         scraper.Exporters,
		MaxBodySize:       scraper.MaxBodySize,
		RequestsPerSecond: scraper.RequestsPerSecond,
		Timeout:           scraper.Timeout,
		UserAgent:         scraper.UserAgent,

		// we do our own deduplication in the watcher
		URLRevisitEnabled: true,
	}

	gcfg.StartRequestsFunc = func(g *geziyor.Geziyor) {
		for _, site := range scraper.Sites {
			if site.UseChrome {
				// using external chrome browser for rendering java script
				g.GetRendered(site.URL, newParseFunc(site, scraper.Logger))
			} else {
				// directly scrape the plain web site content without rendering JS
				g.Get(site.URL, newParseFunc(site, scraper.Logger))
			}
		}
	}

	scraper.Geziyor = geziyor.NewGeziyor(&gcfg)

	return scraper
}

// Opt is a type representing functional Scraper options.
type Opt func(*Scraper)

// WithAlloweDomains is white-listing only the given domains for scraping.
func WithAllowedDomains(domains []string) Opt {
	return func(s *Scraper) {
		s.AllowedDomains = domains
	}
}

// WithBrowserEndpoint is configuring the endpoint for connecting to a chrome
// browser instance for rendering the web site.
func WithBrowserEndpoint(endpoint string) Opt {
	return func(s *Scraper) {
		s.BrowserEndpoint = endpoint
	}
}

// WithTimeout is setting the client timeout of the scraper.
func WithTimeout(timeout time.Duration) Opt {
	return func(s *Scraper) {
		s.Timeout = timeout
	}
}

// WithExporters is adding exporters to export the scraped content (e.g. into a DB).
func WithExporters(exporters []exporter.Interface) Opt {
	return func(s *Scraper) {
		s.Exporters = exporters
	}
}

// WithLogger configures the Logger.
func WithLogger(logger *slog.Logger) Opt {
	return func(s *Scraper) {
		s.Logger = logger
	}
}
