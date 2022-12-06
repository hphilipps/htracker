package service

import "gitlab.com/henri.philipps/htracker"

// Subscription is an interface for a service managing subscribers to updates of web sites
// to be scraped.
type Subscription interface {
	Subscribe(email string, site *htracker.Site) error
	GetSitesBySubscriber(email string) (sites []*htracker.Site, err error)
	GetSubscribersBySite(site *htracker.Site) (emails []string, err error)
	GetSubscribers() (emails []string, err error)
	Unsubscribe(email string, site *htracker.Site) error
	DeleteSubscriber(email string) error
}

// Subscriber is holding the list of subscribed sites of a Subscriber.
type Subscriber struct {
	Email string
	Sites []*htracker.Site
}
