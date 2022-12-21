package storage

import "gitlab.com/henri.philipps/htracker"

// Subscriber is holding the list of subscriptions of a Subscriber.
type Subscriber struct {
	Email         string
	Subscriptions []*htracker.Subscription
}

// SubscriptionStorage is an interface describing a storage backend for a SubscriptionSvc.
type SubscriptionStorage interface {
	FindBySubscriber(email string) ([]*htracker.Subscription, error)
	FindBySubscription(*htracker.Subscription) ([]*Subscriber, error)
	GetAllSubscribers() ([]*Subscriber, error)
	AddSubscription(email string, subscription *htracker.Subscription) error
	RemoveSubscription(email string, subscription *htracker.Subscription) error
	RemoveSubscriber(email string) error
}
