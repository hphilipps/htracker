package main

import (
	"context"
	"flag"
	"os"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/exporter"
	"gitlab.com/henri.philipps/htracker/scraper"
	"gitlab.com/henri.philipps/htracker/service/memory"
	"golang.org/x/exp/slog"
)

var urlFlag = flag.String("url", "", "url to be scraped")
var filterFlag = flag.String("f", "", "filter to be applied to scraped content")
var contentTypeFlag = flag.String("t", "text", "content type of the scraped url")

func main() {

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout))
	db := memory.NewMemoryDB()
	exp := exporter.NewExporter(context.Background(), db)
	site := &htracker.Site{
		URL:         *urlFlag,
		Filter:      *filterFlag,
		ContentType: *contentTypeFlag,
	}

	h1 := scraper.NewScraper(
		[]*htracker.Site{site},
		scraper.WithExporters([]exporter.Interface{exp}),
		scraper.WithBrowserEndpoint("ws://localhost:3000"),
	)

	h2 := scraper.NewScraper(
		[]*htracker.Site{site},
		scraper.WithExporters([]exporter.Interface{exp}),
		scraper.WithBrowserEndpoint("ws://localhost:3000"),
	)

	h1.Start()

	sa, err := db.Get(site)
	if err != nil {
		logger.Error("db.Get failed", err)
		os.Exit(1)
	}

	logger.Info("Site on 1st update", "lastUdpdated", sa.LastUpdated, "lastChecked", sa.LastChecked, "checksum", sa.Checksum, "diff", sa.Diff, "content", string(sa.Content)[0:20])

	time.Sleep(time.Second)

	h2.Start()

	sa, err = db.Get(site)
	if err != nil {
		logger.Error("db.Get failed", err)
		os.Exit(1)
	}

	logger.Info("Site on 2nd update", "lastUdpdated", sa.LastUpdated, "lastChecked", sa.LastChecked, "checksum", sa.Checksum, "diff", sa.Diff, "content", string(sa.Content)[0:20])
}
