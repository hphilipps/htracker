package htracker

import "time"

// Site contains the meta data necessary to describe a web site to be scraped.
type Site struct {
	URL         string
	Filter      string
	ContentType string
	UseChrome   bool
	Interval    time.Duration
}

// Equal is a method for comparing identifying metadata for this site with the given site.
// The combination of URL, Filter and ContentType must be equal for sites to be equal.
// It is not meant to compare the _content_ of web sites.
func (s *Site) Equals(site *Site) bool {
	return s.URL == site.URL && s.Filter == site.Filter && s.ContentType == site.ContentType && s.UseChrome == site.UseChrome
}

// SiteContent is holding metadata, checksum, content and diff to previous version of a scraped web site.
type SiteContent struct {
	Site        *Site
	LastUpdated time.Time
	LastChecked time.Time
	Content     []byte
	Checksum    string
	Diff        string
}