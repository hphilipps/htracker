package storage

import (
	"context"

	"gitlab.com/henri.philipps/htracker"
)

// SiteStorage is an interface describing a storage backend for a SiteArchive service.
type SiteStorage interface {
	Get(context.Context, *htracker.Subscription) (*htracker.Site, error)
	Add(context.Context, *htracker.Site) error
	Update(context.Context, *htracker.Site) error
}
