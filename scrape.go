package htracker

import "github.com/geziyor/geziyor"

// Scraper is used to scrape web sites.
type Scraper struct {
	*geziyor.Geziyor
}

// NewScraper is returning a new Scraper to scrape given web sites.
func NewScraper(opts *geziyor.Options) *Scraper {
	return &Scraper{geziyor.NewGeziyor(opts)}
}
