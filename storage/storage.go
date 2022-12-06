package storage

import (
	"errors"

	"github.com/sergi/go-diff/diffmatchpatch"
	"gitlab.com/henri.philipps/htracker"
)

var ErrNotExist = errors.New("the item could not be found")
var ErrAlreadyExists = errors.New("the item already exists")

// SiteDB is an interface for a DB that can store the state of scraped web sites (content, checksum etc).
type SiteDB interface {
	UpdateSiteArchive(*htracker.SiteArchive) (diff string, err error)
	GetSiteArchive(site *htracker.Site) (sa *htracker.SiteArchive, err error)
}

// SubscriberDB is an interface for a DB to store subscribers to updates of web sites
// to be scraped.
type SubscriberDB interface {
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

// DiffText is a helper function for comparing the content of two sites.
func DiffText(s1, s2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(s1, s2, true)
	return dmp.DiffPrettyText(dmp.DiffCleanupSemantic(diffs))
}
