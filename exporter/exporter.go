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

// wrapping geziyor export.Exporter here to avoid needing to import it in depending packages
type Interface = export.Exporter

// ArchiveSvc is implementing the Exporter interface and exporting scrape results into a SiteArchive service.
type ArchiveSvc struct {
	ctx        context.Context
	archivesvc service.SiteArchive
	logger     slog.Logger
}

// NewExporter is returning a new exporter which is exporting scrape results into the given SiteArchive service.
func NewExporter(ctx context.Context, archive service.SiteArchive) *ArchiveSvc {
	return &ArchiveSvc{ctx: ctx, archivesvc: archive, logger: *slog.New(slog.NewTextHandler(os.Stdout).WithGroup("exporter"))}
}

// Export is reading from the given exports channel and exporting the data into a SiteArchive.
func (e *ArchiveSvc) Export(exports chan interface{}) error {
	for res := range exports {

		sc, ok := res.(*htracker.SiteContent)
		if !ok {
			return fmt.Errorf("expected response of type *SiteContent, got %T", res)
		}

		_, err := e.archivesvc.Update(sc)
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
