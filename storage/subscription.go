package storage

import "gitlab.com/henri.philipps/htracker"

// Subscriber is holding the list of subscribed sites of a Subscriber.
type Subscriber struct {
	Email string
	Sites []*htracker.Site
}

// SubscriptionStorage is an interface describing a storage backend for a Subscription service.
type SubscriptionStorage interface {
	FindBySubscriber(email string) (sites []*htracker.Site, err error)
	FindBySite(*htracker.Site) (subscribers []*Subscriber, err error)
	GetAllSubscribers() (subscribers []*Subscriber, err error)
	AddSubscription(email string, site *htracker.Site) error
	RemoveSubscription(email string, site *htracker.Site) error
	RemoveSubscriber(email string) error
}
