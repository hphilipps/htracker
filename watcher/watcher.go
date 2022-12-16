package watcher

import (
	"fmt"
	"sync"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/exporter"
	"gitlab.com/henri.philipps/htracker/scraper"
	"gitlab.com/henri.philipps/htracker/service"
	"golang.org/x/exp/slog"
	"golang.org/x/net/context"
)

// Watcher is scraping subscribed sites in regular intervals.
type Watcher struct {
	archive         service.SiteArchive
	subscriptions   service.Subscription
	logger          *slog.Logger
	interval        time.Duration
	batchSize       int
	threads         int
	scraperTimeout  time.Duration
	browserEndpoint string
}

// NewWatcher is returning a new Watcher instance.
func NewWatcher(archive service.SiteArchive, subscriptions service.Subscription, opts ...Opt) *Watcher {
	watcher := &Watcher{
		archive:       archive,
		subscriptions: subscriptions,
		logger:        slog.Default(),
		interval:      time.Hour,
	}

	for _, opt := range opts {
		opt(watcher)
	}

	return watcher
}

// Opt is a functional option for a watcher.
type Opt func(*Watcher)

// WithBrowserEndpoint configures a Watcher to connect to the given
// chrome instance for rendering websites.
func WithBrowserEndpoint(endpoint string) Opt {
	return func(w *Watcher) {
		w.browserEndpoint = endpoint
	}
}

// GenerateScrapeList is returning a list of Sites to be scraped by going through
// all subscriptions and deduplicating them.
func (w *Watcher) GenerateScrapeList() (sites []*htracker.Site, err error) {

	// set of unique sites for deduplication of scrape list
	siteSet := map[string]bool{}

	subscribers, err := w.subscriptions.GetSubscribers()
	if err != nil {
		return sites, fmt.Errorf("service.Subscription.GetSubscribers() - %w", err)
	}

	for _, sub := range subscribers {
		for _, site := range sub.Sites {
			// deduplicate sites
			if !siteSet[site.URL+site.Filter+site.ContentType] {
				sites = append(sites, site)
				siteSet[site.URL+site.Filter+site.ContentType] = true
			}
		}
	}

	return sites, nil
}

// RunScrapers is starting up worker threads to scrape the given sites and waits for them to finish.
func (w *Watcher) RunScrapers(sites []*htracker.Site) error {
	ctx, _ := context.WithTimeout(context.Background(), w.interval)
	//done := make(chan struct{}, w.threads)
	wg := &sync.WaitGroup{}

	exporters := []exporter.Interface{exporter.NewExporter(ctx, w.archive)}
	batches := make(chan []*htracker.Site, w.threads)

	// spin up workers
	for i := 0; i < w.threads; i++ {

		// capture loop var for use in closure
		n := i

		wg.Add(1)
		w.logger.Debug("watcher: starting worker", "worker", i)
		go func() {
			defer wg.Done()
			for {
				w.logger.Debug("watcher: waiting for next batch of sites to process", "worker", n)
				select {
				case batch, ok := <-batches:
					if !ok {
						w.logger.Debug("watcher: no more sites to process - worker shutting down", "worker", n)
						return
					}

					w.logger.Debug("watcher: scraper starting", "worker", n)
					scraper.NewScraper(batch,
						scraper.WithExporters(exporters),
						scraper.WithBrowserEndpoint(w.browserEndpoint),
						scraper.WithLogger(w.logger),
						scraper.WithTimeout(w.scraperTimeout),
					).Start()
					w.logger.Debug("watcher: scraper finished", "worker", n)

				case <-ctx.Done():
					w.logger.Debug("watcher: worker canceled - shutting down", "worker", n, "error", ctx.Err())
					return
				}
			}
		}()
	}

	batch := []*htracker.Site{}
	count := 0
	last := len(sites) - 1

	// send batches of sites to workers for scraping
	for i, site := range sites {
		count++
		batch = append(batch, site)
		if count == w.batchSize || i == last {
			select {
			case batches <- batch:
			case <-ctx.Done():
				w.logger.Debug("watcher: RunScrapers() canceled", "error", ctx.Err())
				return ctx.Err()
			}
			count = 0
			batch = []*htracker.Site{}
		}
	}

	close(batches)

	w.logger.Debug("watcher: waiting for workers to finish")
	wg.Wait()
	w.logger.Debug("watcher: all workers finished")

	return nil
}
