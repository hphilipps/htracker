package htracker

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/exp/slog"
)

var ErrNotExist = errors.New("the item could not be found")
var ErrAlreadyExists = errors.New("the item already exists")

// SiteDB is an interface for a DB that can store the state of scraped web sites (content, checksum etc).
type SiteDB interface {
	UpdateSiteArchive(*SiteArchive) (diff string, err error)
	GetSiteArchive(site *Site) (sa *SiteArchive, err error)
}

// SubscriberDB is an interface for a DB to store subscribers to updates of web sites
// to be scraped.
type SubscriberDB interface {
	Subscribe(email string, site *Site) error
	GetSitesBySubscriber(email string) (sites []*Site, err error)
	GetSubscribersBySite(site *Site) (emails []string, err error)
	GetSubscribers() (emails []string, err error)
	Unsubscribe(email string, site *Site) error
	DeleteSubscriber(email string) error
}

// subscriber is holding the list of subscribed sites of a subscriber.
type subscriber struct {
	email string
	sites []*Site
}

// diffText is a helper function for comparing the content of two sites.
func diffText(s1, s2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(s1, s2, true)
	return dmp.DiffPrettyText(dmp.DiffCleanupSemantic(diffs))
}

// compiler check of interface implementation
var _ SiteDB = &MemoryDB{}
var _ SubscriberDB = &MemoryDB{}

// MemoryDB is an in-memory implementation of SiteDB - mainly for testing.
type MemoryDB struct {
	sites       []*SiteArchive
	subscribers []*subscriber
	logger      slog.Logger
	mu          sync.Mutex
}

// NewMemoryDB returns a new MomeoryDB instance.
func NewMemoryDB() *MemoryDB {
	return &MemoryDB{logger: *slog.New(slog.NewTextHandler(os.Stdout).WithGroup("memory_db"))}
}

/*** methods for implementing SiteDB interface ***/

// UpdateSiteArchive is updating the DB with the results of the latest scrape of a site.
func (db *MemoryDB) UpdateSiteArchive(sa *SiteArchive) (diff string, err error) {

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
			sarchive.Diff = diffText(string(sarchive.Content), string(sa.Content))
			sarchive.Content = sa.Content
			sarchive.Checksum = sa.Checksum

			return sarchive.Diff, nil
		}
	}

	// site not found above, making new entry
	db.sites = append(db.sites, &SiteArchive{
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
func (db *MemoryDB) GetSiteArchive(site *Site) (sa *SiteArchive, err error) {

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if site.Equals(sarchive.Site) {
			return sarchive, nil
		}
	}

	return &SiteArchive{}, ErrNotExist
}

/*** methods for implementing SubscriberDB interface ***/

// Subscribe is adding a subscription for the given email and will return
// an error if the subscription already exists.
func (db *MemoryDB) Subscribe(email string, site *Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.email == email {
			for _, s := range subscriber.sites {
				if s.Equals(site) {
					return fmt.Errorf("subscription already exists, %w", ErrAlreadyExists)
				}
			}
			// subscription not found above - adding site to list of sites
			subscriber.sites = append(subscriber.sites, site)
			return nil
		}
	}

	// subscriber not found above - adding new subscriber
	db.subscribers = append(db.subscribers, &subscriber{email: email, sites: []*Site{site}})

	return nil
}

// GetSitesBySubscribers returns a list of subscribed sites for the given subscriber.
func (db *MemoryDB) GetSitesBySubscriber(email string) (sites []*Site, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.email == email {
			return subscriber.sites, nil
		}
	}

	return nil, ErrNotExist
}

// GetSubscribersBySite returns a list of subscribed emails for a given site.
func (db *MemoryDB) GetSubscribersBySite(site *Site) (emails []string, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		for _, s := range subscriber.sites {
			if s.Equals(site) {
				emails = append(emails, subscriber.email)
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
		emails = append(emails, subscriber.email)
	}

	return emails, nil
}

func (db *MemoryDB) Unsubscribe(email string, site *Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.email == email {

			for i, s := range subscriber.sites {
				if s.Equals(site) {
					//remove element i from list
					subscriber.sites[i] = subscriber.sites[len(subscriber.sites)-1]
					subscriber.sites = subscriber.sites[:len(subscriber.sites)-1]
					return nil
				}
			}

			return fmt.Errorf("unsubscribe: %s was not subscribed to url %s, filter %s, content type %s, %w",
				email, site.URL, site.Filter, site.ContentType, ErrNotExist)
		}
	}

	return fmt.Errorf("unsubscribe: email %s not found - %w", email, ErrNotExist)
}

func (db *MemoryDB) DeleteSubscriber(email string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, subscriber := range db.subscribers {
		if subscriber.email == email {
			db.subscribers[i] = db.subscribers[len(db.subscribers)-1]
			db.subscribers = db.subscribers[:len(db.subscribers)-1]
			return nil
		}
	}
	return ErrNotExist
}
