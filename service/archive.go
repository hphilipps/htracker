package service

import (
	"github.com/sergi/go-diff/diffmatchpatch"
	"gitlab.com/henri.philipps/htracker"
)

// SiteArchive is an interface for a service that can store the state of scraped web sites (content, checksum etc).
type SiteArchive interface {
	Update(*htracker.SiteArchive) (diff string, err error)
	Get(site *htracker.Site) (sa *htracker.SiteArchive, err error)
}

// DiffText is a helper function for comparing the content of two sites.
func DiffText(s1, s2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(s1, s2, true)
	return dmp.DiffPrettyText(dmp.DiffCleanupSemantic(diffs))
}
