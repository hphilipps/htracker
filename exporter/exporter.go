package exporter

import (
	"context"
	"fmt"

	"github.com/geziyor/geziyor/export"
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"golang.org/x/exp/slog"
)

// wrapping geziyor export.Exporter here to avoid needing to import it in depending packages
type Interface = export.Exporter

// archiveExporter is implementing the Exporter interface and exporting scrape results into a SiteArchive service.
type archiveExporter struct {
	ctx        context.Context
	archivesvc service.SiteArchive
	logger     slog.Logger
}

type Opt func(*archiveExporter)

// WithLogger configures the logger of the archiveExporter
func WithLogger(logger *slog.Logger) Opt {
	return func(exp *archiveExporter) {
		exp.logger = *logger
	}
}

// NewExporter is returning a new exporter which is exporting scrape results into the given SiteArchive service.
func NewExporter(ctx context.Context, archive service.SiteArchive, opts ...Opt) *archiveExporter {
	exp := &archiveExporter{
		ctx:        ctx,
		archivesvc: archive,
		logger:     *slog.Default(),
	}

	for _, opt := range opts {
		opt(exp)
	}

	return exp
}

// Export is reading from the given exports channel and exporting the data into a SiteArchive.
func (e *archiveExporter) Export(exports chan interface{}) error {
	for res := range exports {

		sc, ok := res.(*htracker.SiteContent)
		if !ok {
			return fmt.Errorf("exporter.Export(): expected response of type *SiteContent, got %T", res)
		}

		_, err := e.archivesvc.Update(sc)
		if err != nil {
			e.logger.Error("exporter.Export(): failed to update site in db", err)
		}

		select {
		case <-e.ctx.Done():
			e.logger.Warn("exporter.Export(): was signaled to stop via context - some scrape results might not have been exported to storage")
			return e.ctx.Err()
		default:
		}
	}

	return nil
}
