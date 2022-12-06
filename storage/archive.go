package storage

import "gitlab.com/henri.philipps/htracker"

type ArchiveStorage interface {
	Find(site *htracker.Site) (sa *htracker.SiteArchive, err error)
	Add(sa *htracker.SiteArchive) error
	Update(sa *htracker.SiteArchive) error
}
