package scraper

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/exporter"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"golang.org/x/exp/slog"
)

const intTestVarName = "INTEGRATION_TESTS"

func runIntegrationTests() bool {
	intTestVar := os.Getenv(intTestVarName)

	if run, err := strconv.ParseBool(intTestVar); err != nil || !run {
		return false
	}

	return true
}

func TestScraper(t *testing.T) {

	sites := []*htracker.Site{
		{URL: "https://httpbin.org/anything"},
		{URL: "http://quotes.toscrape.com/"},
		{URL: "https://httpbin.org/anything"},
		{URL: "http://quotes.toscrape.com/"},
	}

	storage := memory.NewArchiveStorage(*slog.New(slog.NewTextHandler(os.Stdout)))
	archive := service.NewSiteArchive(storage)
	exp := exporter.NewExporter(context.Background(), archive)
	scraper := NewScraper(sites, WithExporters([]exporter.Interface{exp}))

	date := time.Now()
	scraper.Start()

	for _, s := range sites {
		sa, err := archive.Get(s)

		if err != nil {
			t.Errorf("MemoryDB.GetSite() failed: %v", err)
		}

		if sa.LastUpdated.Before(date) {
			t.Errorf("Expected lastUpdated to be after %v, got %v", date, sa.LastUpdated)
		}

		if !sa.LastUpdated.Equal(sa.LastChecked) {
			t.Errorf("Expected lastUpdated (%v) to be equal to lastChecked (%v)", sa.LastUpdated, sa.LastChecked)
		}

		if len(sa.Content) < 10 {
			t.Errorf("Expected content length to be greater then 10, got %d", len(sa.Content))
		}

		if sa.Checksum == "" {
			t.Errorf("Expected checksum not to be empty")
		}

		if sa.Diff != "" {
			t.Errorf("Expected diff to be empty")
		}
	}
}

func TestGetRendered(t *testing.T) {

	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	s := &Scraper{}
	s.Geziyor = geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
			g.GetRendered("http://quotes.toscrape.com/", g.Opt.ParseFunc)
			g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
			g.GetRendered("http://quotes.toscrape.com/", g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			if l := len(r.Header["Content-Length"]); l != 1 {
				t.Errorf("Expected to find 1 Content-Length header, got %d", l)
			}
		},
		BrowserEndpoint: "ws://localhost:3000",
	})

	s.Start()
}

func TestGetRenderedWithFilter(t *testing.T) {

	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	s := &Scraper{}
	s.Geziyor = geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.GetRendered("http://quotes.toscrape.com/", func(g *geziyor.Geziyor, r *client.Response) {
				count := 0
				r.HTMLDoc.Find("div.quote").Each(func(i int, s *goquery.Selection) {
					count++
				})
				if count != 10 {
					t.Errorf("Expected 10 quotes, got %d", count)
				}
			})
		},
		BrowserEndpoint: "ws://localhost:3000",
	})

	s.Start()
}
