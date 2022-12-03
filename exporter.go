package htracker

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/exp/slog"
)

type Exporter struct {
	ctx    context.Context
	db     SiteDB
	logger slog.Logger
}

func NewExporter(ctx context.Context, db SiteDB) *Exporter {
	return &Exporter{ctx: ctx, db: db, logger: *slog.New(slog.NewTextHandler(os.Stdout).WithGroup("exporter"))}
}

func (e *Exporter) Export(exports chan interface{}) error {
	for res := range exports {

		sarchive, ok := res.(*siteArchive)
		if !ok {
			return fmt.Errorf("expected response of type *siteArchive, got %T", res)
		}

		_, err := e.db.UpdateSite(sarchive.lastChecked, *sarchive.site, sarchive.content, sarchive.checksum)
		if err != nil {
			e.logger.Error("failed to update site in db", err)
		}

		select {
		case <-e.ctx.Done():
			e.logger.Warn("was signaled to stop via context - some scrape results might not have been exported to storage")
			return nil
		default:
		}
	}

	return nil
}
