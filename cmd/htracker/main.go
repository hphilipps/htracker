package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/exporter"
	"gitlab.com/henri.philipps/htracker/scraper"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"gitlab.com/henri.philipps/htracker/watcher"
	"golang.org/x/exp/slog"
)

var urlFlag = flag.String("url", "", "url to be scraped")
var filterFlag = flag.String("f", "", "filter to be applied to scraped content")
var contentTypeFlag = flag.String("t", "text", "content type of the scraped url")
var chromeWSFlag = flag.String("ws", "ws://localhost:3000", "websocket url to connect to chrome instance for site rendering")
var renderWithChromeFlag = flag.Bool("r", false, "render site content using chrome if true (needed for java script)")

func main() {

	flag.Parse()

	logger := slog.Default()
	storage := memory.NewSiteStorage(logger)
	archive := service.NewSiteArchive(storage)
	exp := exporter.NewExporter(context.Background(), archive)
	site := &htracker.Site{
		URL:         *urlFlag,
		Filter:      *filterFlag,
		ContentType: *contentTypeFlag,
	}

	opts := []scraper.ScraperOpt{scraper.WithExporters([]exporter.Interface{exp})}
	if *renderWithChromeFlag {
		opts = append(opts, scraper.WithBrowserEndpoint(*chromeWSFlag))
	}

	h1 := scraper.NewScraper(
		[]*htracker.Site{site},
		opts...,
	)

	h2 := scraper.NewScraper(
		[]*htracker.Site{site},
		opts...,
	)

	h1.Start()

	sc, err := archive.Get(site)
	if err != nil {
		logger.Error("db.Get() failed", err)
		os.Exit(1)
	}

	fmt.Printf("Site on 1st update: lastUdpdated: %v, lastChecked: %v, checksum: %s, diff: %s\ncontent: %s\n", sc.LastUpdated, sc.LastChecked, sc.Checksum, sc.Diff, sc.Content)
	content1 := sc.Content

	time.Sleep(time.Second)

	h2.Start()

	sc, err = archive.Get(site)
	if err != nil {
		logger.Error("db.Get() failed", err)
		os.Exit(1)
	}

	fmt.Printf("Site on 2nd update: lastUdpdated: %v, lastChecked: %v, checksum: %s, diff: --%s--\ncontent: %s\n", sc.LastUpdated, sc.LastChecked, sc.Checksum, sc.Diff, sc.Content)

	fmt.Println("len c1:", len(content1), "len c2:", len(sc.Content))

	for i, c := range content1 {
		if c != sc.Content[i] {
			fmt.Println(i, ": ", c, "!=", sc.Content[i])
			fmt.Println(string(content1)[i-5 : i+10])
			fmt.Println(string(sc.Content)[i-5 : i+10])

			break
		}
	}

	subscriptionSvc := service.NewSubscriptionSvc(storage)

	subscriptionSvc.Subscribe("email1", &htracker.Site{URL: "http://httpbin.org/anything/1"})
	subscriptionSvc.Subscribe("email1", &htracker.Site{URL: "http://httpbin.org/anything/2"})
	subscriptionSvc.Subscribe("email2", &htracker.Site{URL: "http://httpbin.org/anything/2"})

	dbgLogger := slog.New(slog.HandlerOptions{Level: slog.DebugLevel}.NewTextHandler(os.Stdout))

	w := watcher.NewWatcher(archive, subscriptionSvc, watcher.WithInterval(5*time.Second), watcher.WithLogger(dbgLogger))

	ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
	if err := w.Start(ctx); err != nil {
		logger.Error("Watcher", err)
	}

	sa, err := archive.Get(&htracker.Site{URL: "http://httpbin.org/anything/2"})
	if err != nil {
		logger.Error("ArchiveService", err)
	}

	fmt.Printf("LU: %v: %v\n", sa.LastUpdated, sa.Diff)
}
