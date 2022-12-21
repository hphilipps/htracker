package htracker

import "time"

// Subscription contains the meta data necessary to describe a web site to be watched for updates.
type Subscription struct {
	URL         string
	Filter      string
	ContentType string
	UseChrome   bool
	Interval    time.Duration
}

// Equal is a method for comparing subscriptions, mainly to deduplicate same subscriptions by different subscribers.
// The combination of URL, Filter and ContentType must be equal for subscriptions to be equal.
func (s1 *Subscription) Equals(s2 *Subscription) bool {
	return s1.URL == s2.URL && s1.Filter == s2.Filter && s1.ContentType == s2.ContentType && s1.UseChrome == s2.UseChrome
}
