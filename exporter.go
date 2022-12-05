package htracker

import (
	"context"
	"fmt"
	"os"

	"github.com/geziyor/geziyor/export"
	"golang.org/x/exp/slog"
)

// type Exporter interface {
// 	Export(exports chan interface{}) error
// }

type Exporter = export.Exporter

type DBExporter struct {
	ctx    context.Context
	db     SiteDB
	logger slog.Logger
}

func NewExporter(ctx context.Context, db SiteDB) *DBExporter {
	return &DBExporter{ctx: ctx, db: db, logger: *slog.New(slog.NewTextHandler(os.Stdout).WithGroup("exporter"))}
}

func (e *DBExporter) Export(exports chan interface{}) error {
	for res := range exports {

		sarchive, ok := res.(*SiteArchive)
		if !ok {
			return fmt.Errorf("expected response of type *siteArchive, got %T", res)
		}

		_, err := e.db.UpdateSiteArchive(sarchive)
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
