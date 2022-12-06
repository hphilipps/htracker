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

// ArchiveSvc is implementing exporter.Interface and exporting into a SiteArchive service.
type ArchiveSvc struct {
	ctx        context.Context
	archivesvc service.SiteArchive
	logger     slog.Logger
}

func NewExporter(ctx context.Context, archive service.SiteArchive) *ArchiveSvc {
	return &ArchiveSvc{ctx: ctx, archivesvc: archive, logger: *slog.New(slog.NewTextHandler(os.Stdout).WithGroup("exporter"))}
}

func (e *ArchiveSvc) Export(exports chan interface{}) error {
	for res := range exports {

		sarchive, ok := res.(*htracker.SiteArchive)
		if !ok {
			return fmt.Errorf("expected response of type *siteArchive, got %T", res)
		}

		_, err := e.archivesvc.Update(sarchive)
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
