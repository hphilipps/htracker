package scraper

import (
	"net/http"
	"regexp"
	"time"

	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/export"
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/exporter"
	"gitlab.com/henri.philipps/htracker/service"
	"golang.org/x/exp/slog"
)

// Scraper is used to scrape web sites.
type Scraper struct {
	*geziyor.Geziyor

	Subscriptions []*htracker.Subscription
	Logger        *slog.Logger

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

// newParseFunc is returning a new parser func, setup to parse the site content for the given subscription.
// and send the results as siteArchive to the Exports channel.
func newParseFunc(subscription *htracker.Subscription, logger *slog.Logger) func(*geziyor.Geziyor, *client.Response) {
	return func(g *geziyor.Geziyor, r *client.Response) {
		var content []byte

		if r.Response.StatusCode >= http.StatusBadRequest {
			logger.Warn("got error status code", "code", r.Response.StatusCode, "url", subscription.URL)
			return
		}

		if subscription.Filter == "" {
			content = r.Body
		} else if r.HTMLDoc != nil {
			content = []byte(r.HTMLDoc.Find(subscription.Filter).Text())
		} else {
			exp, err := regexp.Compile(subscription.Filter)
			if err != nil {
				logger.Error("ParseFunc failed to compile regexp", err, slog.String("regexp", subscription.Filter), slog.String("site", subscription.URL))
				return
			}
			content = exp.Find(r.Body)
		}

		sa := &htracker.Site{
			Subscription: subscription,
			LastChecked:  time.Now(),
			Content:      content,
			Checksum:     service.Checksum(content),
		}

		g.Exports <- sa
	}
}

// NewScraper is returning a new Scraper to scrape the sites of the given subscriptions.
func NewScraper(subscriptions []*htracker.Subscription, opts ...Opt) *Scraper {

	scraper := &Scraper{
		Subscriptions: subscriptions,
		Logger:        slog.Default(),
		UserAgent:     "HTracker/Geziyor 1.0",
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
		for _, subscription := range scraper.Subscriptions {
			if subscription.UseChrome {
				// using external chrome browser for rendering java script
				g.GetRendered(subscription.URL, newParseFunc(subscription, scraper.Logger))
			} else {
				// directly scrape the plain web site content without rendering JS
				g.Get(subscription.URL, newParseFunc(subscription, scraper.Logger))
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
