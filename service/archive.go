package service

import (
	"errors"
	"fmt"

	"github.com/sergi/go-diff/diffmatchpatch"
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
)

// SiteArchive is an interface for a service that can store the state of scraped web sites (content, checksum etc).
type SiteArchive interface {
	Update(*htracker.SiteArchive) (diff string, err error)
	Get(site *htracker.Site) (sa *htracker.SiteArchive, err error)
}

// NewSiteArchive is returning a new SiteArchive using the given storage backend.
func NewSiteArchive(storage storage.ArchiveStorage) *siteArchive {
	return &siteArchive{storage: storage}
}

// siteArchive is implementing SiteArchive
type siteArchive struct {
	storage storage.ArchiveStorage
}

// Update is updating the DB with the results of the latest scrape of a site.
func (archive *siteArchive) Update(sa *htracker.SiteArchive) (diff string, err error) {

	sarchive, err := archive.storage.Find(sa.Site)
	if err != nil {
		if errors.Is(err, htracker.ErrNotExist) {
			// site archive not found - create new entry
			if err := archive.storage.Add(sa); err != nil {
				return "", fmt.Errorf("ArchiveStorage.Add() - %w", err)
			}
			return "", nil
		}
		return "", fmt.Errorf("ArchiveStorage.Find() - %w", err)
	}

	// content unchanged
	if sarchive.Checksum == sa.Checksum {
		sarchive.LastChecked = sa.LastChecked
		if err := archive.storage.Update(sarchive); err != nil {
			return "", fmt.Errorf("ArchiveStorage.Update() - %w", err)
		}
		return "", nil
	}

	// content changed
	sarchive.LastChecked = sa.LastChecked
	sarchive.LastUpdated = sa.LastChecked
	sarchive.Diff = DiffText(string(sarchive.Content), string(sa.Content))
	sarchive.Content = sa.Content
	sarchive.Checksum = sa.Checksum

	if err := archive.storage.Update(sarchive); err != nil {
		return sarchive.Diff, fmt.Errorf("ArchiveStorage.Update() - %w", err)
	}
	return sarchive.Diff, nil
}

// Get is returning metadata, checksum and content of a site in the DB identified by URL, filter and contentType.
func (archive *siteArchive) Get(site *htracker.Site) (sa *htracker.SiteArchive, err error) {

	sa, err = archive.storage.Find(site)
	if err != nil {
		return &htracker.SiteArchive{}, fmt.Errorf("ArchiveStorage.Find() - %w", err)
	}

	return sa, nil
}

// DiffText is a helper function for comparing the content of two sites.
func DiffText(s1, s2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(s1, s2, true)
	return dmp.DiffPrettyText(dmp.DiffCleanupSemantic(diffs))
}
