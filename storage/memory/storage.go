package memory

import (
	"fmt"
	"sync"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

// DB is an in-memory implementation of the Archive and Subscription storage interfaces - mainly for testing.
type DB struct {
	sites       []*htracker.SiteArchive
	subscribers []*storage.Subscriber
	logger      slog.Logger
	mu          sync.Mutex
}

// NewArchiveStorage returns a new in-memory site archive storage which can be used by a SiteArchive service.
func NewArchiveStorage(logger slog.Logger) *DB {
	return &DB{logger: logger}
}

// NewSubscriptionStorage returns a new in-memory SubscriptionStorage which can be used by a Subscription service.
func NewSubscriptionStorage(logger slog.Logger) *DB {
	return &DB{logger: logger}
}

/*** Implementation of SubscriptionStorage interface ***/

// compile time check of interface implementation
var _ storage.ArchiveStorage = &DB{}

// Find is returning the site archive for the given site or ErrNotExist if not found.
func (db *DB) Find(site *htracker.Site) (sa *htracker.SiteArchive, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if site.Equals(sarchive.Site) {
			return sarchive, nil
		}
	}

	return &htracker.SiteArchive{}, htracker.ErrNotExist
}

// Add is adding a new site to the archive.
func (db *DB) Add(sa *htracker.SiteArchive) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if sa.Site.Equals(sarchive.Site) {
			return htracker.ErrAlreadyExists
		}
	}

	db.sites = append(db.sites, &htracker.SiteArchive{
		Site:        sa.Site,
		LastUpdated: sa.LastChecked,
		LastChecked: sa.LastChecked,
		Content:     sa.Content,
		Checksum:    sa.Checksum,
		Diff:        "",
	})

	return nil
}

// Update is updating a site in the archive if found.
func (db *DB) Update(sa *htracker.SiteArchive) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if sa.Site.Equals(sarchive.Site) {
			sarchive.Site = sa.Site
			sarchive.LastChecked = sa.LastChecked
			sarchive.LastUpdated = sa.LastUpdated
			sarchive.Diff = sa.Diff
			sarchive.Content = sa.Content
			sarchive.Checksum = sa.Checksum

			return nil
		}
	}
	return htracker.ErrNotExist
}

/*** Implementation of SubscriptionStorage interface ***/

// compile time check of interface implementation
var _ storage.SubscriptionStorage = &DB{}

// FindBySubscriber is returning all subscribed sites for a given subscriber.
func (db *DB) FindBySubscriber(email string) (sites []*htracker.Site, err error) {
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
func (db *DB) FindBySite(site *htracker.Site) (subscribers []*storage.Subscriber, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

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
func (db *DB) GetAllSubscribers() (subscribers []*storage.Subscriber, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.subscribers, nil
}

// AddSubscription is adding a new subscription if it doesn't exist yet.
func (db *DB) AddSubscription(email string, site *htracker.Site) error {
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
func (db *DB) RemoveSubscription(email string, site *htracker.Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {

			for i, s := range subscriber.Sites {
				if s.Equals(site) {
					//remove element i from list
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
func (db *DB) RemoveSubscriber(email string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, subscriber := range db.subscribers {
		if subscriber.Email == email {
			//remove element i from list
			db.subscribers[i] = db.subscribers[len(db.subscribers)-1]
			db.subscribers = db.subscribers[:len(db.subscribers)-1]
			return nil
		}
	}
	return htracker.ErrNotExist
}
