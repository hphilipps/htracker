package htracker

import "time"

// Site is holding content and metadata of a subscribed site.
type Site struct {
	Subscription *Subscription
	LastUpdated  time.Time
	LastChecked  time.Time
	Content      []byte
	Checksum     string
	Diff         string
}
