package service

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/sergi/go-diff/diffmatchpatch"
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
)

// SiteArchive is an interface for a service that can store the state of scraped web sites (content, checksum etc).
type SiteArchive interface {
	Update(*htracker.SiteContent) (diff string, err error)
	Get(site *htracker.Site) (content *htracker.SiteContent, err error)
}

// NewSiteArchive is returning a new SiteArchive using the given storage backend.
func NewSiteArchive(storage storage.SiteStorage) *siteArchive {
	return &siteArchive{storage: storage}
}

// siteArchive is implementing SiteArchive.
type siteArchive struct {
	storage storage.SiteStorage
}

// Update is updating the DB with the results of the latest scrape of a site.
func (archive *siteArchive) Update(content *htracker.SiteContent) (diff string, err error) {
	acontent, err := archive.storage.Find(content.Site)
	if err != nil {
		if errors.Is(err, htracker.ErrNotExist) {
			// site archive not found - create new entry
			if err := archive.storage.Add(content); err != nil {
				return "", fmt.Errorf("ArchiveStorage.Add() - %w", err)
			}
			return "", nil
		}
		return "", fmt.Errorf("ArchiveStorage.Find() - %w", err)
	}

	// content unchanged
	if acontent.Checksum == content.Checksum {
		acontent.LastChecked = content.LastChecked
		if err := archive.storage.Update(acontent); err != nil {
			return "", fmt.Errorf("ArchiveStorage.Update() - %w", err)
		}
		return "", nil
	}

	// content changed
	acontent.LastChecked = content.LastChecked

	diff = DiffText(string(acontent.Content), string(content.Content))
	if diff == "" {
		// The diff function is ignoring whitespace changes as sometimes
		// whitespace is rendered randomly. So it can happen that we see
		// a changed checksum, but no diff. In this case we treat the site
		// as not changed.
		return "", nil
	}

	acontent.Diff = diff
	acontent.LastUpdated = content.LastChecked
	acontent.Content = content.Content
	acontent.Checksum = content.Checksum

	if err := archive.storage.Update(acontent); err != nil {
		return acontent.Diff, fmt.Errorf("ArchiveStorage.Update() - %w", err)
	}
	return acontent.Diff, nil
}

// Get is returning metadata, checksum and content of a site in the DB identified by URL, filter and contentType.
func (archive *siteArchive) Get(site *htracker.Site) (*htracker.SiteContent, error) {
	content, err := archive.storage.Find(site)
	if err != nil {
		return &htracker.SiteContent{}, fmt.Errorf("ArchiveStorage.Find() - %w", err)
	}

	return content, nil
}

func DiffPrintAsText(diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer
	for _, diff := range diffs {
		text := diff.Text

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString("\x1b[32m")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("\x1b[0m")
		case diffmatchpatch.DiffDelete:
			_, _ = buff.WriteString("\x1b[31m")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("\x1b[0m")
		case diffmatchpatch.DiffEqual:
		}
	}

	return buff.String()
}

// stripStringsBuilder is stripping whitespace from the given string.
func stripStringsBuilder(str string) string {
	var builder strings.Builder
	builder.Grow(len(str))
	for _, rune := range str {
		if !unicode.IsSpace(rune) {
			builder.WriteRune(rune)
		}
	}
	return builder.String()
}

// DiffText is a helper function for comparing the content of sites.
// We try to ignore whitespace changes, as sometimes whitespace seems to be rendered randomly.
func DiffText(str1, str2 string) string {
	if stripStringsBuilder(str1) == stripStringsBuilder(str2) {
		return ""
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(str1, str2, false)
	return DiffPrintAsText(dmp.DiffCleanupSemantic(diffs))
}

func Checksum(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
