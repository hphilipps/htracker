package storage

import (
	"context"

	"gitlab.com/henri.philipps/htracker"
)

// Subscriber is holding the list of subscriptions of a Subscriber.
type Subscriber struct {
	Email             string
	Subscriptions     []*htracker.Subscription
	SubscriptionLimit int
}

// SubscriptionStorage is an interface describing a storage backend for a SubscriptionSvc.
type SubscriptionStorage interface {
	FindBySubscriber(ctx context.Context, email string) ([]*htracker.Subscription, error)
	FindBySubscription(context.Context, *htracker.Subscription) ([]*Subscriber, error)
	SubscriberCount(context.Context) (int, error)
	AddSubscriber(context.Context, *Subscriber) error
	GetAllSubscribers(context.Context) ([]*Subscriber, error)
	GetSubscriber(ctx context.Context, email string) (*Subscriber, error)
	AddSubscription(ctx context.Context, email string, subscription *htracker.Subscription) error
	RemoveSubscription(ctx context.Context, email string, subscription *htracker.Subscription) error
	RemoveSubscriber(ctx context.Context, email string) error
}
