package storage

import "gitlab.com/henri.philipps/htracker"

// SiteStorage is an interface describing a storage backend for a SiteArchive service.
type SiteStorage interface {
	Find(site *htracker.Site) (content *htracker.SiteContent, err error)
	Add(content *htracker.SiteContent) error
	Update(content *htracker.SiteContent) error
}
