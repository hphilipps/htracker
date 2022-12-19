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
	archive       service.SiteArchive
	subscriptions service.Subscription
	logger        *slog.Logger
	interval      time.Duration
	batchSize     int
	threads       int
	scraperOpts   []scraper.Opt
}

// NewWatcher is returning a new Watcher instance.
func NewWatcher(archive service.SiteArchive, subscriptions service.Subscription, opts ...Opt) *Watcher {
	watcher := &Watcher{
		archive:       archive,
		subscriptions: subscriptions,
		logger:        slog.Default(),
		interval:      time.Hour,
		batchSize:     4,
		threads:       2,
	}

	for _, opt := range opts {
		opt(watcher)
	}

	return watcher
}

// Opt is a functional option for a watcher.
type Opt func(*Watcher)

// WithInterval sets the interval between scrape runs.
func WithInterval(interval time.Duration) Opt {
	return func(w *Watcher) {
		w.interval = interval
	}
}

// WithScraperOpts sets options for the scrapers that are launched with RunScrapers().
func WithScraperOpts(opts ...scraper.Opt) Opt {
	return func(w *Watcher) {
		w.scraperOpts = opts
	}
}

// WithBatchSize sets the size of the batch of sites given to a Scraper instance for processing.
func WithBatchSize(bs int) Opt {
	return func(w *Watcher) {
		w.batchSize = bs
	}
}

// WithThreads sets the number of scrapers working in parallel.
func WithThreads(threads int) Opt {
	return func(w *Watcher) {
		w.threads = threads
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Opt {
	return func(w *Watcher) {
		w.logger = logger
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
// When all scrapers finished there still might be exporters processing the results asynchronously.
func (w *Watcher) RunScrapers(ctx context.Context, sites []*htracker.Site) error {
	tctx, _ := context.WithTimeout(ctx, w.interval)
	wg := &sync.WaitGroup{}
	batches := make(chan []*htracker.Site, w.threads)

	// spin up workers
	w.startWorkers(tctx, batches, wg)

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
			case <-tctx.Done():
				w.logger.Debug("watcher: RunScrapers() interrupted", "error", tctx.Err())
				return tctx.Err()
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

// startWorkers is spinning up scraper threads for concurrent processing of batches of sites.
func (w *Watcher) startWorkers(ctx context.Context, batches chan []*htracker.Site, wg *sync.WaitGroup) {

	exporters := []exporter.Interface{exporter.NewExporter(ctx, w.archive)}

	for i := 0; i < w.threads; i++ {
		workerNr := i // capture loop var for use in closure
		wg.Add(1)
		w.logger.Debug("watcher: starting worker", "worker", i)

		go func() {
			defer wg.Done()
			for {
				w.logger.Debug("watcher: waiting for next batch of sites to process", slog.Int("worker", workerNr))
				select {
				case batch, ok := <-batches:
					if !ok {
						w.logger.Debug("watcher: no more sites to process - worker shutting down", slog.Int("worker", workerNr))
						return
					}

					scraper := scraper.NewScraper(batch,
						scraper.WithExporters(exporters),
						scraper.WithLogger(w.logger),
					)
					for _, opt := range w.scraperOpts {
						opt(scraper)
					}

					w.logger.Debug("watcher: scraper starting", slog.Int("worker", workerNr))
					scraper.Start()
					w.logger.Debug("watcher: scraper finished", "worker", workerNr)

				case <-ctx.Done():
					w.logger.Debug("watcher: worker canceled - shutting down", slog.Int("worker", workerNr), "error", ctx.Err())
					return
				}
			}
		}()
	}
}

// Start is making the watcher scrape all subscribed websites in regular intervals.
// It can be stopped by canceling the given context.
func (w *Watcher) Start(ctx context.Context) error {

	w.logger.Info("Watcher started", "interval", w.interval, "threads", w.threads, "batchSize", w.batchSize)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		sites, err := w.GenerateScrapeList()
		if err != nil {
			return fmt.Errorf("watcher.GenerateScrapeList(): %w", err)
		}

		if err := w.RunScrapers(ctx, sites); err != nil {
			w.logger.Error("Watcher: RunScrapers() failed", err)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
