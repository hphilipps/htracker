package memory

import (
	"sync"

	"gitlab.com/henri.philipps/htracker"
	"golang.org/x/exp/slog"
)

func NewArchiveStorage(logger slog.Logger) *DB {
	return &DB{logger: logger}
}

// Subscriber is holding the list of subscribed sites of a Subscriber.
type Subscriber struct {
	Email string
	Sites []*htracker.Site
}

// DB is an in-memory implementation of the Archive and Subscription storage interfaces - mainly for testing.
type DB struct {
	sites       []*htracker.SiteArchive
	subscribers []*Subscriber
	logger      slog.Logger
	mu          sync.Mutex
}

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
