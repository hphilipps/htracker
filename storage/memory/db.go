package memory

import (
	"fmt"
	"sync"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

func NewArchiveStorage(logger slog.Logger) *DB {
	return &DB{logger: logger}
}

func NewSubscriptionStorage(logger slog.Logger) *DB {
	return &DB{logger: logger}
}

// DB is an in-memory implementation of the Archive and Subscription storage interfaces - mainly for testing.
type DB struct {
	sites       []*htracker.SiteArchive
	subscribers []*storage.Subscriber
	logger      slog.Logger
	mu          sync.Mutex
}

/*** Implementation of SubscriptionStorage interface ***/

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

func (db *DB) GetAllSubscribers() (subscribers []*storage.Subscriber, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.subscribers, nil
}

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
