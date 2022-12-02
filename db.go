package htracker

import (
	"errors"
	"sync"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var ErrNotExist = errors.New("the item could not be found")

// Site contains the meta data necessary to describe a web site to be scraped.
type Site struct {
	URL         string
	Filter      string
	ContentType string
	Interval    time.Duration
}

// Equal is a method for comparing identifying metadata for this site with the given site.
// The combination of URL, Filter and ContentType must be equal for sites to be equal.
// It is not meant to compare the _content_ of web sites.
func (s *Site) Equals(site *Site) bool {
	return s.URL == site.URL && s.Filter == site.Filter && s.ContentType == site.ContentType
}

// SiteDB is an interface for a DB that can store the state of scraped web sites (content, checksum etc).
type SiteDB interface {
	UpdateSite(date time.Time, site Site, content []byte, checksum string) (diff string, err error)
	GetSite(url, filter, contentType string) (lastUpdated, lastChecked time.Time, content []byte, checksum, diff string, err error)
}

// SubscriberDB is an interface for a DB to store subscribers to updates of web sites
// to be scraped.
type SubscriberDB interface {
	UpdateSubscriber(email string, sites []Site) error
	GetSitesBySubscriber(email string) (sites []Site, err error)
	GetSubscribersBySite(site Site) (emails []string, err error)
	GetSubscribers() (emails []string, err error)
	Unsubscribe(email string, site Site) error
	DeleteSubscriber(email string) error
}

// siteArchive is holding metadata, checksum and content of a scraped web site.
type siteArchive struct {
	site        *Site
	lastUpdated time.Time
	lastChecked time.Time
	content     []byte
	checksum    string
	diff        string
}

// subscriber is holding the list of subscribed sites of a subscriber.
type subscriber struct {
	email string
	sites []Site
}

// diffText is a helper function for comparing the content of two sites.
func diffText(s1, s2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(s1, s2, true)
	return dmp.DiffPrettyText(dmp.DiffCleanupSemantic(diffs))
}

// compiler check of interface implementation
var _ SiteDB = &MemoryDB{}

// MemoryDB is an in-memory implementation of SiteDB - mainly for testing.
type MemoryDB struct {
	sites       []*siteArchive
	subscribers []*subscriber
	mu          sync.Mutex
}

// UpdateSite is updating the DB with the results of the latest scrape of a site.
func (db *MemoryDB) UpdateSite(date time.Time, site Site, content []byte, checksum string) (diff string, err error) {

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if site.Equals(sarchive.site) {

			// content unchanged
			if sarchive.checksum == checksum {
				sarchive.lastChecked = date
				return "", nil
			}

			// content changed
			sarchive.lastChecked = date
			sarchive.lastUpdated = date
			sarchive.diff = diffText(string(sarchive.content), string(content))
			sarchive.content = content
			sarchive.checksum = checksum

			return sarchive.diff, nil
		}
	}

	// site not found above, making new entry
	db.sites = append(db.sites, &siteArchive{
		site:        &site,
		lastUpdated: date,
		lastChecked: date,
		content:     content,
		checksum:    checksum,
		diff:        "",
	})

	return "", nil
}

// GetSite is returning metadata, checksum and content of a site in the DB identified by URL, filter and contentType.
func (db *MemoryDB) GetSite(url, filter, contentType string) (lastUpdated, lastChecked time.Time, content []byte, checksum, diff string, err error) {

	db.mu.Lock()
	defer db.mu.Unlock()

	site := &Site{URL: url, Filter: filter, ContentType: contentType}

	for _, sarchive := range db.sites {
		if site.Equals(sarchive.site) {
			return sarchive.lastUpdated, sarchive.lastChecked, sarchive.content, sarchive.checksum, sarchive.diff, nil
		}
	}

	return time.Time{}, time.Time{}, []byte{}, "", "", ErrNotExist
}
