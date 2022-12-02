package htracker

import (
	"context"
	"fmt"
	"log"
)

type Exporter struct {
	ctx context.Context
	db  SiteDB
}

func NewExporter(ctx context.Context, db SiteDB) *Exporter {
	return &Exporter{ctx: ctx, db: db}
}

func (e *Exporter) Export(exports chan interface{}) error {
	for res := range exports {

		sarchive, ok := res.(*siteArchive)
		if !ok {
			return fmt.Errorf("expected response of type *siteArchive, got %T", res)
		}

		_, err := e.db.UpdateSite(sarchive.lastChecked, *sarchive.site, sarchive.content, sarchive.checksum)
		if err != nil {
			log.Printf("exporter: failed to update site in db - %v", err)
		}

		select {
		case <-e.ctx.Done():
			log.Printf("exporter: was signaled to stop via context - some results might not have been stored")
			return nil
		default:
		}
	}

	return nil
}
