package service

import (
	"fmt"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

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

func NewSubscriptionSvc(storage storage.SubscriptionStorage) *subscriptionSvc {
	return &subscriptionSvc{storage: storage}
}

type subscriptionSvc struct {
	storage storage.SubscriptionStorage
	logger  slog.Logger
}

// Subscribe is adding a subscription for the given email and will return
// an error if the subscription already exists.
func (svc *subscriptionSvc) Subscribe(email string, site *htracker.Site) error {

	err := svc.storage.AddSubscription(email, site)
	if err != nil {
		return fmt.Errorf("storage.AddSubscription() - %w", err)
	}

	return nil
}

// GetSitesBySubscribers returns a list of subscribed sites for the given subscriber.
func (svc *subscriptionSvc) GetSitesBySubscriber(email string) (sites []*htracker.Site, err error) {
	sites, err = svc.storage.FindBySubscriber(email)
	if err != nil {
		return sites, fmt.Errorf("storage.FindBySubscriber() - %w", err)
	}

	return sites, nil
}

// GetSubscribersBySite returns a list of subscribed emails for a given site.
func (svc *subscriptionSvc) GetSubscribersBySite(site *htracker.Site) (subscribers []*Subscriber, err error) {

	sub, err := svc.storage.FindBySite(site)
	if err != nil {
		return []*Subscriber{}, fmt.Errorf("storage.FindBySite() - %w", err)
	}

	// TODO: avoid this transformation. factor out Subscriber type?
	for _, s := range sub {
		subscribers = append(subscribers, &Subscriber{Email: s.Email, Sites: s.Sites})
	}

	return subscribers, nil
}

// GetSubscribers returns all existing subscribers.
func (svc *subscriptionSvc) GetSubscribers() (subscribers []*Subscriber, err error) {

	sub, err := svc.storage.GetAllSubscribers()
	if err != nil {
		return []*Subscriber{}, fmt.Errorf("storage.GetAllSubscribers() - %w", err)
	}

	// TODO: avoid this transformation. factor out Subscriber type?
	for _, s := range sub {
		subscribers = append(subscribers, &Subscriber{Email: s.Email, Sites: s.Sites})
	}

	return subscribers, nil
}

// Unsubscribe is unsubscribing a subscriber from watching a site.
func (svc *subscriptionSvc) Unsubscribe(email string, site *htracker.Site) error {

	if err := svc.storage.RemoveSubscription(email, site); err != nil {
		return fmt.Errorf("storage.RemoveSubscription() - %w", err)
	}

	return nil
}

// DeleteSubscriber is removing a subscriber with all it's subscriptions.
func (svc *subscriptionSvc) DeleteSubscriber(email string) error {

	if err := svc.storage.RemoveSubscriber(email); err != nil {
		return fmt.Errorf("storage.RemoveSubscriber - %w", err)
	}

	return nil
}
