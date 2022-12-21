package service

import (
	"fmt"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

// SubscriptionSvc is an interface for a service managing subscriptions to updates of web sites
// to be scraped.
type SubscriptionSvc interface {
	Subscribe(email string, subscription *htracker.Subscription) error
	GetSubscriptionsBySubscriber(email string) ([]*htracker.Subscription, error)
	GetSubscribersBySubscription(*htracker.Subscription) ([]*Subscriber, error)
	GetSubscribers() ([]*Subscriber, error)
	Unsubscribe(email string, subscription *htracker.Subscription) error
	DeleteSubscriber(email string) error
}

// Subscriber is describing a user holding subscriptions to sites.
type Subscriber struct {
	Email         string
	Subscriptions []*htracker.Subscription
}

// subscriptionSvc is implementing the SubscriptionSvc interface.
type subscriptionSvc struct {
	storage storage.SubscriptionStorage
	logger  slog.Logger
}

// compile time check of interface implementation.
var _ SubscriptionSvc = &subscriptionSvc{}

// NewSubscriptionSvc is returning a new SubscriptionService using the given storage backend.
func NewSubscriptionSvc(storage storage.SubscriptionStorage) *subscriptionSvc {
	return &subscriptionSvc{storage: storage}
}

// SubScriptionSvcOpt is representing functional options for the SubscriptionSvc.
type SubscriptionSvcOpt func(*subscriptionSvc)

// WithLogger is setting the logger of the SubscriptionSvc.
func WithLogger(logger *slog.Logger) SubscriptionSvcOpt {
	return func(svc *subscriptionSvc) {
		svc.logger = *logger
	}
}

// Subscribe is adding a subscription for the given email and will return
// an error if the subscription already exists.
func (svc *subscriptionSvc) Subscribe(email string, subscription *htracker.Subscription) error {
	err := svc.storage.AddSubscription(email, subscription)
	if err != nil {
		return fmt.Errorf("storage.AddSubscription(): %w", err)
	}

	return nil
}

// GetSubscriptionsBySubscribers returns a list of subscriptions for the given subscriber.
func (svc *subscriptionSvc) GetSubscriptionsBySubscriber(email string) ([]*htracker.Subscription, error) {
	sites, err := svc.storage.FindBySubscriber(email)
	if err != nil {
		return sites, fmt.Errorf("storage.FindBySubscriber(): %w", err)
	}

	return sites, nil
}

// GetSubscribersBySubscription returns a list of subscribed emails for a given site.
func (svc *subscriptionSvc) GetSubscribersBySubscription(subscription *htracker.Subscription) (subscribers []*Subscriber, err error) {
	storSubscribers, err := svc.storage.FindBySubscription(subscription)
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
func (svc *subscriptionSvc) GetSubscribers() (subscribers []*Subscriber, err error) {
	storSubscribers, err := svc.storage.GetAllSubscribers()
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
func (svc *subscriptionSvc) Unsubscribe(email string, subscription *htracker.Subscription) error {
	if err := svc.storage.RemoveSubscription(email, subscription); err != nil {
		return fmt.Errorf("storage.RemoveSubscription(): %w", err)
	}

	return nil
}

// DeleteSubscriber is removing a subscriber with all it's subscriptions.
func (svc *subscriptionSvc) DeleteSubscriber(email string) error {
	if err := svc.storage.RemoveSubscriber(email); err != nil {
		return fmt.Errorf("storage.RemoveSubscriber: %w", err)
	}

	return nil
}
