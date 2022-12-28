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
	archive     service.SiteArchive
	subSvc      service.SubscriptionSvc
	logger      *slog.Logger
	interval    time.Duration
	batchSize   int
	threads     int
	scraperOpts []scraper.Opt
}

// NewWatcher is returning a new Watcher instance.
func NewWatcher(archive service.SiteArchive, subSvc service.SubscriptionSvc, opts ...Opt) *Watcher {
	watcher := &Watcher{
		archive:   archive,
		subSvc:    subSvc,
		logger:    slog.Default(),
		interval:  time.Hour,
		batchSize: 4,
		threads:   2,
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

// WithBatchSize sets the size of the batch of subscriptions given to a Scraper instance for processing.
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

// GenerateScrapeList is returning a list of Subscriptions to be scraped by going through
// all subscriptions and deduplicating them.
func (w *Watcher) GenerateScrapeList() (subscriptions []*htracker.Subscription, err error) {

	// set of unique sites for deduplication of scrape list
	siteSet := map[string]bool{}

	subscribers, err := w.subSvc.GetSubscribers(context.Background())
	if err != nil {
		return subscriptions, fmt.Errorf("SubscriptionSvc.GetSubscribers(): %w", err)
	}

	for _, subscriber := range subscribers {
		for _, subscription := range subscriber.Subscriptions {
			// deduplicate subscriptions
			if !siteSet[subscription.URL+subscription.Filter+subscription.ContentType] {
				subscriptions = append(subscriptions, subscription)
				siteSet[subscription.URL+subscription.Filter+subscription.ContentType] = true
			}
		}
	}

	return subscriptions, nil
}

// RunScrapers is starting up worker threads to scrape the given subscriptions and waits for them to finish.
// When all scrapers finished there still might be exporters processing the results asynchronously.
func (w *Watcher) RunScrapers(ctx context.Context, subscriptions []*htracker.Subscription) error {
	tctx, _ := context.WithTimeout(ctx, w.interval)
	wg := &sync.WaitGroup{}
	batches := make(chan []*htracker.Subscription, w.threads)

	// spin up workers
	w.startWorkers(tctx, batches, wg)

	batch := []*htracker.Subscription{}
	count := 0
	last := len(subscriptions) - 1

	// send batches of subscriptions to workers for scraping
	for i, sub := range subscriptions {
		count++
		batch = append(batch, sub)
		if count == w.batchSize || i == last {
			select {
			case batches <- batch:
			case <-tctx.Done():
				w.logger.Debug("watcher: RunScrapers() interrupted", "error", tctx.Err())
				return tctx.Err()
			}
			count = 0
			batch = []*htracker.Subscription{}
		}
	}

	close(batches)

	w.logger.Debug("watcher: waiting for workers to finish")
	wg.Wait()
	w.logger.Debug("watcher: all workers finished")

	return nil
}

// startWorkers is spinning up scraper threads for concurrent processing of batches of subscriptions.
func (w *Watcher) startWorkers(ctx context.Context, batches chan []*htracker.Subscription, wg *sync.WaitGroup) {

	exporters := []exporter.Interface{exporter.NewExporter(ctx, w.archive)}

	for i := 0; i < w.threads; i++ {
		workerNr := i // capture loop var for use in closure
		wg.Add(1)
		w.logger.Debug("watcher: starting worker", "worker", i)

		go func() {
			defer wg.Done()
			for {
				w.logger.Debug("watcher: waiting for next batch of subscriptions to process", slog.Int("worker", workerNr))
				select {
				case batch, ok := <-batches:
					if !ok {
						w.logger.Debug("watcher: no more subscriptions to process - worker shutting down", slog.Int("worker", workerNr))
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
