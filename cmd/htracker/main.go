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
	"golang.org/x/exp/slog"
)

var urlFlag = flag.String("url", "", "url to be scraped")
var filterFlag = flag.String("f", "", "filter to be applied to scraped content")
var contentTypeFlag = flag.String("t", "text", "content type of the scraped url")
var chromeWSFlag = flag.String("ws", "ws://localhost:3000", "websocket url to connect to chrome instance for site rendering")
var renderWithChromeFlag = flag.Bool("r", false, "render site content using chrome if true (needed for java script)")

func main() {

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout))
	storage := memory.NewSiteStorage(*slog.New(slog.NewTextHandler(os.Stdout)))
	archive := service.NewSiteArchive(storage)
	exp := exporter.NewExporter(context.Background(), archive)
	site := &htracker.Site{
		URL:         *urlFlag,
		Filter:      *filterFlag,
		ContentType: *contentTypeFlag,
	}

	opts := []scraper.ScraperOpt{scraper.WithExporters([]exporter.Interface{exp})}
	if *renderWithChromeFlag {
		opts = append(opts, scraper.WithBrowserEndpoint("ws://localhost:3000"))
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
		logger.Error("db.Get failed", err)
		os.Exit(1)
	}

	fmt.Printf("Site on 1st update: lastUdpdated: %v, lastChecked: %v, checksum: %s, diff: %s\ncontent: %s\n", sc.LastUpdated, sc.LastChecked, sc.Checksum, sc.Diff, sc.Content)
	content1 := sc.Content

	time.Sleep(time.Second)

	h2.Start()

	sc, err = archive.Get(site)
	if err != nil {
		logger.Error("db.Get failed", err)
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
}
