package htracker

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
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

	sites := []*Site{
		{URL: "https://httpbin.org/anything"},
		{URL: "http://quotes.toscrape.com/"},
		{URL: "https://httpbin.org/anything"},
		{URL: "http://quotes.toscrape.com/"},
	}

	db := NewMemoryDB()
	exporter := NewExporter(context.Background(), db)
	scraper := NewScraper(sites, WithExporters([]Exporter{exporter}))

	date := time.Now()
	scraper.Start()

	for _, s := range sites {
		lastUpdated, lastChecked, content, checksum, diff, err := db.GetSite(s.URL, "", "")

		if err != nil {
			t.Errorf("MemoryDB.GetSite() failed: %v", err)
		}

		if lastUpdated.Before(date) {
			t.Errorf("Expected lastUpdated to be after %v, got %v", date, lastUpdated)
		}

		if !lastUpdated.Equal(lastChecked) {
			t.Errorf("Expected lastUpdated (%v) to be equal to lastChecked (%v)", lastUpdated, lastChecked)
		}

		if len(content) < 10 {
			t.Errorf("Expected content length to be greater then 10, got %d", len(content))
		}

		if checksum == "" {
			t.Errorf("Expected checksum not to be empty")
		}

		if diff != "" {
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
