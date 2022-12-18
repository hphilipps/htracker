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
	archive     []*htracker.SiteContent
	subscribers []*storage.Subscriber
	logger      *slog.Logger
	mu          sync.Mutex
}

// NewSiteStorage returns a new in-memory site content storage which can be used by a SiteArchive service.
func NewSiteStorage(logger *slog.Logger) *memDB {
	return &memDB{logger: logger}
}

// NewSubscriptionStorage returns a new in-memory SubscriptionStorage which can be used by a Subscription service.
func NewSubscriptionStorage(logger *slog.Logger) *memDB {
	return &memDB{logger: logger}
}

/*** Implementation of SubscriptionStorage interface ***/

// compile time check of interface implementation.
var _ storage.SiteStorage = &memDB{}

// Find is returning the site content for the given site or ErrNotExist if not found.
func (db *memDB) Find(site *htracker.Site) (*htracker.SiteContent, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sc := range db.archive {
		if site.Equals(sc.Site) {
			return sc, nil
		}
	}

	return &htracker.SiteContent{}, htracker.ErrNotExist
}

// Add is adding a new site to the archive.
func (db *memDB) Add(content *htracker.SiteContent) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sc := range db.archive {
		if content.Site.Equals(sc.Site) {
			return htracker.ErrAlreadyExists
		}
	}

	db.archive = append(db.archive, &htracker.SiteContent{
		Site:        content.Site,
		LastUpdated: content.LastChecked,
		LastChecked: content.LastChecked,
		Content:     content.Content,
		Checksum:    content.Checksum,
		Diff:        "",
	})

	return nil
}

// Update is updating a site in the site archive if found.
func (db *memDB) Update(content *htracker.SiteContent) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, acontent := range db.archive {
		if content.Site.Equals(acontent.Site) {
			acontent.Site = content.Site
			acontent.LastChecked = content.LastChecked
			acontent.LastUpdated = content.LastUpdated
			acontent.Diff = content.Diff
			acontent.Content = content.Content
			acontent.Checksum = content.Checksum

			return nil
		}
	}
	return htracker.ErrNotExist
}

/*** Implementation of SubscriptionStorage interface ***/

// compile time check of interface implementation.
var _ storage.SubscriptionStorage = &memDB{}

// FindBySubscriber is returning all subscribed sites for a given subscriber.
func (db *memDB) FindBySubscriber(email string) ([]*htracker.Site, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			return subscriber.Sites, nil
		}
	}

	return nil, htracker.ErrNotExist
}

// FindBySite is returning all subscribers subscribed to the given site.
func (db *memDB) FindBySite(site *htracker.Site) ([]*storage.Subscriber, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	subscribers := []*storage.Subscriber{}
	for _, subscriber := range db.subscribers {
		for _, s := range subscriber.Sites {
			if s.Equals(site) {
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
func (db *memDB) AddSubscription(email string, site *htracker.Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			for _, s := range subscriber.Sites {
				if s.Equals(site) {
					return fmt.Errorf("subscription already exists, %w", htracker.ErrAlreadyExists)
				}
			}
			// subscription not found above - adding site to list of sites
			subscriber.Sites = append(subscriber.Sites, site)
			return nil
		}
	}

	// subscriber not found above - adding new subscriber
	db.subscribers = append(db.subscribers, &storage.Subscriber{Email: email, Sites: []*htracker.Site{site}})

	return nil
}

// RemoveSubscription is removing the subscription of a subscriber to a site.
func (db *memDB) RemoveSubscription(email string, site *htracker.Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			for i, s := range subscriber.Sites {
				if s.Equals(site) {
					// remove element i from list
					subscriber.Sites[i] = subscriber.Sites[len(subscriber.Sites)-1]
					subscriber.Sites = subscriber.Sites[:len(subscriber.Sites)-1]
					return nil
				}
			}

			return fmt.Errorf("%s was not subscribed to url %s, filter %s, content type %s, %w",
				email, site.URL, site.Filter, site.ContentType, htracker.ErrNotExist)
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
