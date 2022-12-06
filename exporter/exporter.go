package exporter

import (
	"context"
	"fmt"
	"os"

	"github.com/geziyor/geziyor/export"
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"golang.org/x/exp/slog"
)

// wrapping geziyor export.Exporter to avoid needing to import it in depending packages
type Interface = export.Exporter

// SiteDB is implementing exporter.Interface and exporting into a SiteDB
type SiteDB struct {
	ctx    context.Context
	db     service.SiteArchive
	logger slog.Logger
}

func NewExporter(ctx context.Context, db service.SiteArchive) *SiteDB {
	return &SiteDB{ctx: ctx, db: db, logger: *slog.New(slog.NewTextHandler(os.Stdout).WithGroup("exporter"))}
}

func (e *SiteDB) Export(exports chan interface{}) error {
	for res := range exports {

		sarchive, ok := res.(*htracker.SiteArchive)
		if !ok {
			return fmt.Errorf("expected response of type *siteArchive, got %T", res)
		}

		_, err := e.db.Update(sarchive)
		if err != nil {
			e.logger.Error("failed to update site in db", err)
		}

		select {
		case <-e.ctx.Done():
			e.logger.Warn("was signaled to stop via context - some scrape results might not have been exported to storage")
			return e.ctx.Err()
		default:
		}
	}

	return nil
}
