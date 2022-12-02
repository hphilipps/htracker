package htracker

import (
	"fmt"
	"os"
	"strconv"
	"testing"

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

func TestGetRendered(t *testing.T) {

	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	s := NewScraper(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
			g.GetRendered("http://quotes.toscrape.com/", g.Opt.ParseFunc)
			g.GetRendered("https://httpbin.org/anything2", g.Opt.ParseFunc)
			g.GetRendered("http://quotes.toscrape.com/2", g.Opt.ParseFunc)
		},
		//StartURLs: []string{"http://quotes.toscrape.com/", "https://httpbin.org/anything"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			//t.Log(string(r.Body))
			fmt.Println(r.Request.URL.String(), r.Header)
		},
		BrowserEndpoint: "ws://localhost:3000",
	})

	s.Start()
}

func TestGetFiltered(t *testing.T) {

	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	s := NewScraper(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.GetRendered("https://httpbin.org/anything", func(g *geziyor.Geziyor, r *client.Response) {
				r.HTMLDoc.Find("url").Each(func(i int, s *goquery.Selection) {
					t.Log(s.Text())
				})
			})
		},
		BrowserEndpoint: "ws://localhost:3000",
	})

	s.Start()
}
