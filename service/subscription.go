package service

import (
	"context"
	"fmt"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

// SubscriptionSvc is an interface for a service managing subscriptions to updates of web sites
// to be scraped.
type SubscriptionSvc interface {
	AddSubscriber(context.Context, *Subscriber) error
	Subscribe(ctx context.Context, email string, subscription *htracker.Subscription) error
	GetSubscriptionsBySubscriber(ctx context.Context, email string) ([]*htracker.Subscription, error)
	GetSubscribersBySubscription(context.Context, *htracker.Subscription) ([]*Subscriber, error)
	GetSubscribers(context.Context) ([]*Subscriber, error)
	Unsubscribe(ctx context.Context, email string, subscription *htracker.Subscription) error
	DeleteSubscriber(ctx context.Context, email string) error
}

// Subscriber is describing a user holding subscriptions to sites.
type Subscriber struct {
	Email             string
	Subscriptions     []*htracker.Subscription
	SubscriptionLimit int
}

// subscriptionSvc is implementing the SubscriptionSvc interface.
type subscriptionSvc struct {
	storage           storage.SubscriptionStorage
	logger            slog.Logger
	subscriptionLimit int
	subscriberLimit   int
}

// compile time check of interface implementation.
var _ SubscriptionSvc = &subscriptionSvc{}

// NewSubscriptionSvc is returning a new SubscriptionService using the given storage backend.
func NewSubscriptionSvc(storage storage.SubscriptionStorage, opts ...SubscriptionSvcOpt) *subscriptionSvc {
	svc := &subscriptionSvc{storage: storage, subscriptionLimit: 100, subscriberLimit: 100}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// SubScriptionSvcOpt is representing functional options for the SubscriptionSvc.
type SubscriptionSvcOpt func(*subscriptionSvc)

// WithLogger is setting the logger of the SubscriptionSvc.
func WithLogger(logger *slog.Logger) SubscriptionSvcOpt {
	return func(svc *subscriptionSvc) {
		svc.logger = *logger
	}
}

// WithSubscriptionLimit is setting the maximum number of subscriptions per subscriber.
func WithSubscriptionLimit(limit int) SubscriptionSvcOpt {
	return func(svc *subscriptionSvc) {
		svc.subscriptionLimit = limit
	}
}

// WithSubscriberLimit is setting the maximum number of subscribers.
func WithSubscriberLimit(limit int) SubscriptionSvcOpt {
	return func(svc *subscriptionSvc) {
		svc.subscriberLimit = limit
	}
}

// AddSubscriber is adding a new subscriber.
// A SubscriptionLimit of -1 means unlimited subscriptions.
func (svc *subscriptionSvc) AddSubscriber(ctx context.Context, subscriber *Subscriber) error {
	count, err := svc.storage.SubscriberCount(ctx)
	if err != nil {
		return fmt.Errorf("storage.SubscriberCount(): %w", err)
	}
	if count == svc.subscriberLimit {
		return fmt.Errorf("can't add new subscriber - reached %d subscribers: %w", count, htracker.ErrLimit)
	}

	limit := subscriber.SubscriptionLimit
	if limit == 0 {
		limit = svc.subscriptionLimit
	}
	sub := &storage.Subscriber{Email: subscriber.Email, SubscriptionLimit: limit}

	err = svc.storage.AddSubscriber(ctx, sub)
	if err != nil {
		return fmt.Errorf("storage.AddSubscriber(): %w", err)
	}

	return nil
}

// Subscribe is adding a subscription for the given email and will return
// an error if the subscription already exists or we hit the subscription limit.
func (svc *subscriptionSvc) Subscribe(ctx context.Context, email string, subscription *htracker.Subscription) error {
	subscriber, err := svc.storage.GetSubscriber(ctx, email)
	if err != nil {
		return fmt.Errorf("storage.GetSubscriber(): %w", err)
	}

	if subscriber.SubscriptionLimit > 0 && len(subscriber.Subscriptions) >= subscriber.SubscriptionLimit {
		return fmt.Errorf("can't add new subscription - reached %d subscriptions: %w", subscriber.SubscriptionLimit, htracker.ErrLimit)
	}

	err = svc.storage.AddSubscription(ctx, email, subscription)
	if err != nil {
		return fmt.Errorf("storage.AddSubscription(): %w", err)
	}

	return nil
}

// GetSubscriptionsBySubscribers returns a list of subscriptions for the given subscriber.
func (svc *subscriptionSvc) GetSubscriptionsBySubscriber(ctx context.Context, email string) ([]*htracker.Subscription, error) {
	subscriptions, err := svc.storage.FindBySubscriber(ctx, email)
	if err != nil {
		return subscriptions, fmt.Errorf("storage.FindBySubscriber(): %w", err)
	}

	return subscriptions, nil
}

// GetSubscribersBySubscription returns a list of subscribed emails for a given subscription.
func (svc *subscriptionSvc) GetSubscribersBySubscription(ctx context.Context, subscription *htracker.Subscription) (subscribers []*Subscriber, err error) {
	storSubscribers, err := svc.storage.FindBySubscription(ctx, subscription)
	if err != nil {
		return []*Subscriber{}, fmt.Errorf("storage.FindBySubscription(): %w", err)
	}

	// TODO: should we avoid this transformation? factor out Subscriber type?
	for _, s := range storSubscribers {
		subscribers = append(subscribers, &Subscriber{Email: s.Email, Subscriptions: s.Subscriptions})
	}

	return subscribers, nil
}

// GetSubscribers returns all existing subscribers.
func (svc *subscriptionSvc) GetSubscribers(ctx context.Context) (subscribers []*Subscriber, err error) {
	storSubscribers, err := svc.storage.GetAllSubscribers(ctx)
	if err != nil {
		return []*Subscriber{}, fmt.Errorf("storage.GetAllSubscribers(): %w", err)
	}

	// TODO: should we avoid this transformation? factor out Subscriber type?
	for _, s := range storSubscribers {
		subscribers = append(subscribers, &Subscriber{Email: s.Email, Subscriptions: s.Subscriptions})
	}

	return subscribers, nil
}

// Unsubscribe is unsubscribing a subscriber from watching a site.
func (svc *subscriptionSvc) Unsubscribe(ctx context.Context, email string, subscription *htracker.Subscription) error {
	if err := svc.storage.RemoveSubscription(ctx, email, subscription); err != nil {
		return fmt.Errorf("storage.RemoveSubscription(): %w", err)
	}

	return nil
}

// DeleteSubscriber is removing a subscriber with all it's subscriptions.
func (svc *subscriptionSvc) DeleteSubscriber(ctx context.Context, email string) error {
	if err := svc.storage.RemoveSubscriber(ctx, email); err != nil {
		return fmt.Errorf("storage.RemoveSubscriber: %w", err)
	}

	return nil
}
