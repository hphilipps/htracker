package storage

import "gitlab.com/henri.philipps/htracker"

// SiteStorage is an interface describing a storage backend for a SiteArchive service.
type SiteStorage interface {
	Get(*htracker.Subscription) (*htracker.Site, error)
	Add(content *htracker.Site) error
	Update(content *htracker.Site) error
}
