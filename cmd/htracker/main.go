package main

import (
	"context"
	"flag"
	"os"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"golang.org/x/exp/slog"
)

var urlFlag = flag.String("url", "", "url to be scraped")
var filterFlag = flag.String("f", "", "filter to be applied to scraped content")
var contentTypeFlag = flag.String("t", "text", "content type of the scraped url")

func main() {

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout))
	db := htracker.NewMemoryDB()
	exporter := htracker.NewExporter(context.Background(), db)
	site := &htracker.Site{
		URL:         *urlFlag,
		Filter:      *filterFlag,
		ContentType: *contentTypeFlag,
	}

	h1 := htracker.NewScraper(
		[]*htracker.Site{site},
		htracker.WithExporters([]htracker.Exporter{exporter}),
		htracker.WithBrowserEndpoint("ws://localhost:3000"),
	)

	h2 := htracker.NewScraper(
		[]*htracker.Site{site},
		htracker.WithExporters([]htracker.Exporter{exporter}),
		htracker.WithBrowserEndpoint("ws://localhost:3000"),
	)

	h1.Start()

	sa, err := db.GetSiteArchive(site)
	if err != nil {
		logger.Error("db.GetSiteArchive failed", err)
		os.Exit(1)
	}

	logger.Info("Site on 1st update", "lastUdpdated", sa.LastUpdated, "lastChecked", sa.LastChecked, "checksum", sa.Checksum, "diff", sa.Diff, "content", string(sa.Content)[0:20])

	time.Sleep(time.Second)

	h2.Start()

	sa, err = db.GetSiteArchive(site)
	if err != nil {
		logger.Error("db.GetSiteArchive failed", err)
		os.Exit(1)
	}

	logger.Info("Site on 2nd update", "lastUdpdated", sa.LastUpdated, "lastChecked", sa.LastChecked, "checksum", sa.Checksum, "diff", sa.Diff, "content", string(sa.Content)[0:20])
}
