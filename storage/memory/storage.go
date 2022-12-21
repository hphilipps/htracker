package memory

import (
	"fmt"
	"sync"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

// memDB is an in-memory implementation of the Archive and Subscription storage interfaces - mainly for testing.
type memDB struct {
	archive     []*htracker.Site
	subscribers []*storage.Subscriber
	logger      *slog.Logger
	mu          sync.Mutex
}

// NewSiteStorage returns a new in-memory site content storage which can be used by a SiteArchive service.
func NewSiteStorage(logger *slog.Logger) *memDB {
	return &memDB{logger: logger}
}

// NewSubscriptionStorage returns a new in-memory SubscriptionStorage which can be used by a SubscriptionSvc.
func NewSubscriptionStorage(logger *slog.Logger) *memDB {
	return &memDB{logger: logger}
}

/*** Implementation of SubscriptionStorage interface ***/

// compile time check of interface implementation.
var _ storage.SiteStorage = &memDB{}

// Get is returning the site for the given subscription or ErrNotExist if not found.
func (db *memDB) Get(subscription *htracker.Subscription) (*htracker.Site, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, site := range db.archive {
		if subscription.Equals(site.Subscription) {
			return site, nil
		}
	}

	return &htracker.Site{}, htracker.ErrNotExist
}

// Add is adding a new site to the archive.
func (db *memDB) Add(site *htracker.Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, s := range db.archive {
		if site.Subscription.Equals(s.Subscription) {
			return htracker.ErrAlreadyExists
		}
	}

	db.archive = append(db.archive, &htracker.Site{
		Subscription: site.Subscription,
		LastUpdated:  site.LastChecked,
		LastChecked:  site.LastChecked,
		Content:      site.Content,
		Checksum:     site.Checksum,
		Diff:         "",
	})

	return nil
}

// Update is updating a site in the site archive if found.
func (db *memDB) Update(site *htracker.Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, asite := range db.archive {
		if site.Subscription.Equals(asite.Subscription) {
			asite.Subscription = site.Subscription
			asite.LastChecked = site.LastChecked
			asite.LastUpdated = site.LastUpdated
			asite.Diff = site.Diff
			asite.Content = site.Content
			asite.Checksum = site.Checksum

			return nil
		}
	}
	return htracker.ErrNotExist
}

/*** Implementation of SubscriptionStorage interface ***/

// compile time check of interface implementation.
var _ storage.SubscriptionStorage = &memDB{}

// FindBySubscriber is returning all subscribed sites for a given subscriber.
func (db *memDB) FindBySubscriber(email string) ([]*htracker.Subscription, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			return subscriber.Subscriptions, nil
		}
	}

	return nil, htracker.ErrNotExist
}

// FindBySubscription is returning all subscribers subscribed to the given site.
func (db *memDB) FindBySubscription(subscription *htracker.Subscription) ([]*storage.Subscriber, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	subscribers := []*storage.Subscriber{}
	for _, subscriber := range db.subscribers {
		for _, s := range subscriber.Subscriptions {
			if s.Equals(subscription) {
				subscribers = append(subscribers, subscriber)
				break
			}
		}
	}

	return subscribers, nil
}

// GetAllSubscribers is returning all subscribers.
func (db *memDB) GetAllSubscribers() ([]*storage.Subscriber, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.subscribers, nil
}

// AddSubscription is adding a new subscription if it doesn't exist yet.
func (db *memDB) AddSubscription(email string, subscription *htracker.Subscription) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			for _, s := range subscriber.Subscriptions {
				if s.Equals(subscription) {
					return fmt.Errorf("subscription already exists, %w", htracker.ErrAlreadyExists)
				}
			}
			// subscription not found above - adding site to list of sites
			subscriber.Subscriptions = append(subscriber.Subscriptions, subscription)
			return nil
		}
	}

	// subscriber not found above - adding new subscriber
	db.subscribers = append(db.subscribers, &storage.Subscriber{Email: email, Subscriptions: []*htracker.Subscription{subscription}})

	return nil
}

// RemoveSubscription is removing the subscription of a subscriber to a site.
func (db *memDB) RemoveSubscription(email string, subscription *htracker.Subscription) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			for i, s := range subscriber.Subscriptions {
				if s.Equals(subscription) {
					// remove element i from list
					subscriber.Subscriptions[i] = subscriber.Subscriptions[len(subscriber.Subscriptions)-1]
					subscriber.Subscriptions = subscriber.Subscriptions[:len(subscriber.Subscriptions)-1]
					return nil
				}
			}

			return fmt.Errorf("%s was not subscribed to url %s, filter %s, content type %s, %w",
				email, subscription.URL, subscription.Filter, subscription.ContentType, htracker.ErrNotExist)
		}
	}

	return fmt.Errorf("email %s not found - %w", email, htracker.ErrNotExist)
}

// RemoveSubscriber is removing a subscriber with all it's subscriptions.
func (db *memDB) RemoveSubscriber(email string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, subscriber := range db.subscribers {
		if subscriber.Email == email {
			// remove element i from list
			db.subscribers[i] = db.subscribers[len(db.subscribers)-1]
			db.subscribers = db.subscribers[:len(db.subscribers)-1]
			return nil
		}
	}
	return htracker.ErrNotExist
}
