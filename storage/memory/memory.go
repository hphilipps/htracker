package memory

import (
	"fmt"
	"os"
	"sync"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

// compiler check of interface implementation
var _ storage.SiteDB = &MemoryDB{}
var _ storage.SubscriberDB = &MemoryDB{}

// MemoryDB is an in-memory implementation of SiteDB - mainly for testing.
type MemoryDB struct {
	sites       []*htracker.SiteArchive
	subscribers []*storage.Subscriber
	logger      slog.Logger
	mu          sync.Mutex
}

// NewMemoryDB returns a new MomeoryDB instance.
func NewMemoryDB() *MemoryDB {
	return &MemoryDB{logger: *slog.New(slog.NewTextHandler(os.Stdout).WithGroup("memory_db"))}
}

/*** methods for implementing SiteDB interface ***/

// UpdateSiteArchive is updating the DB with the results of the latest scrape of a site.
func (db *MemoryDB) UpdateSiteArchive(sa *htracker.SiteArchive) (diff string, err error) {

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if sa.Site.Equals(sarchive.Site) {

			// content unchanged
			if sarchive.Checksum == sa.Checksum {
				sarchive.LastChecked = sa.LastChecked
				return "", nil
			}

			// content changed
			sarchive.LastChecked = sa.LastChecked
			sarchive.LastUpdated = sa.LastChecked
			sarchive.Diff = storage.DiffText(string(sarchive.Content), string(sa.Content))
			sarchive.Content = sa.Content
			sarchive.Checksum = sa.Checksum

			return sarchive.Diff, nil
		}
	}

	// site not found above, making new entry
	db.sites = append(db.sites, &htracker.SiteArchive{
		Site:        sa.Site,
		LastUpdated: sa.LastChecked,
		LastChecked: sa.LastChecked,
		Content:     sa.Content,
		Checksum:    sa.Checksum,
		Diff:        "",
	})

	return "", nil
}

// GetSiteArchive is returning metadata, checksum and content of a site in the DB identified by URL, filter and contentType.
func (db *MemoryDB) GetSiteArchive(site *htracker.Site) (sa *htracker.SiteArchive, err error) {

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if site.Equals(sarchive.Site) {
			return sarchive, nil
		}
	}

	return &htracker.SiteArchive{}, storage.ErrNotExist
}

/*** methods for implementing SubscriberDB interface ***/

// Subscribe is adding a subscription for the given email and will return
// an error if the subscription already exists.
func (db *MemoryDB) Subscribe(email string, site *htracker.Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			for _, s := range subscriber.Sites {
				if s.Equals(site) {
					return fmt.Errorf("subscription already exists, %w", storage.ErrAlreadyExists)
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

// GetSitesBySubscribers returns a list of subscribed sites for the given subscriber.
func (db *MemoryDB) GetSitesBySubscriber(email string) (sites []*htracker.Site, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			return subscriber.Sites, nil
		}
	}

	return nil, storage.ErrNotExist
}

// GetSubscribersBySite returns a list of subscribed emails for a given site.
func (db *MemoryDB) GetSubscribersBySite(site *htracker.Site) (emails []string, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		for _, s := range subscriber.Sites {
			if s.Equals(site) {
				emails = append(emails, subscriber.Email)
				break
			}
		}
	}

	return emails, nil
}

// GetSubscribers returns all existing subscribers.
func (db *MemoryDB) GetSubscribers() (emails []string, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		emails = append(emails, subscriber.Email)
	}

	return emails, nil
}

func (db *MemoryDB) Unsubscribe(email string, site *htracker.Site) error {
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

			return fmt.Errorf("unsubscribe: %s was not subscribed to url %s, filter %s, content type %s, %w",
				email, site.URL, site.Filter, site.ContentType, storage.ErrNotExist)
		}
	}

	return fmt.Errorf("unsubscribe: email %s not found - %w", email, storage.ErrNotExist)
}

func (db *MemoryDB) DeleteSubscriber(email string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, subscriber := range db.subscribers {
		if subscriber.Email == email {
			db.subscribers[i] = db.subscribers[len(db.subscribers)-1]
			db.subscribers = db.subscribers[:len(db.subscribers)-1]
			return nil
		}
	}
	return storage.ErrNotExist
}
