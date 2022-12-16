package watcher

import (
	"os"
	"reflect"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"golang.org/x/exp/slog"
)

func TestWatcher_GenerateScrapeList(t *testing.T) {

	email1 := "email1@foo.bar"
	email2 := "email2@foo.bar"
	email3 := "email3@foo.bar"

	site1 := &htracker.Site{URL: "site1.test", Filter: "filter1", ContentType: "text"}
	site1a := &htracker.Site{URL: "site1.test", Filter: "filter2", ContentType: "text"}
	site1b := &htracker.Site{URL: "site1.test", Filter: "filter1", ContentType: "html"}
	site2 := &htracker.Site{URL: "site2.test", Filter: "filter1", ContentType: "text"}

	subscriber1 := &service.Subscriber{email1, []*htracker.Site{site1}}
	subscriber2 := &service.Subscriber{email2, []*htracker.Site{site1, site1a, site1b}}
	subscriber3 := &service.Subscriber{email3, []*htracker.Site{site1, site1a, site1b, site2}}

	tests := []struct {
		name        string
		subscribers []*service.Subscriber
		wantSites   []*htracker.Site
		wantErr     bool
	}{
		{name: "0 sites",
			subscribers: []*service.Subscriber{}, wantSites: []*htracker.Site{}, wantErr: false},
		{name: "1 site",
			subscribers: []*service.Subscriber{subscriber1}, wantSites: []*htracker.Site{site1}, wantErr: false},
		{name: "same site with different filters",
			subscribers: []*service.Subscriber{subscriber2}, wantSites: []*htracker.Site{site1, site1a, site1b}, wantErr: false},
		{name: "multiple subscribers",
			subscribers: []*service.Subscriber{subscriber1, subscriber2, subscriber3}, wantSites: []*htracker.Site{site1, site1a, site1b, site2}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.Default()
			db := memory.NewSubscriptionStorage(logger)
			svc := service.NewSubscriptionSvc(db)
			w := &Watcher{
				subscriptions: svc,
				logger:        logger,
			}

			for _, s := range tt.subscribers {
				for _, site := range s.Sites {
					if err := svc.Subscribe(s.Email, site); err != nil {
						t.Errorf("subscriber %s failed to subscribe to site %s during setup", s.Email, site.URL)
					}
				}
			}

			gotSites, err := w.GenerateScrapeList()
			if (err != nil) != tt.wantErr {
				t.Errorf("Watcher.GenerateScrapeList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.wantSites) != 0 && len(gotSites) != 0 {
				if !reflect.DeepEqual(gotSites, tt.wantSites) {
					t.Errorf("Watcher.GenerateScrapeList() = %v, want %v", gotSites, tt.wantSites)
				}
			}
		})
	}
}

func TestWatcher_RunScrapers(t *testing.T) {
	type fields struct {
		archive        service.SiteArchive
		subscriptions  service.Subscription
		logger         slog.Logger
		interval       time.Duration
		scraperTimeout time.Duration
		batchSize      int
		threads        int
	}
	type args struct {
		sites []*htracker.Site
	}

	site1 := &htracker.Site{URL: "https://httpbin.org/anything", Filter: "filter1", ContentType: "text"}
	site1a := &htracker.Site{URL: "https://httpbin.org/anything", Filter: "filter2", ContentType: "text"}
	site1b := &htracker.Site{URL: "https://httpbin.org/anything", Filter: "filter1", ContentType: "html"}
	site2 := &htracker.Site{URL: "https://httpbin.org/anything/2", Filter: "filter1", ContentType: "text"}

	handler := slog.HandlerOptions{Level: slog.DebugLevel}.NewTextHandler(os.Stdout)
	logger := slog.New(handler)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "scrape 1, threads: 1, batch size: 1", fields: fields{scraperTimeout: time.Minute, batchSize: 1, threads: 1, interval: time.Hour}, args: args{sites: []*htracker.Site{site1}}},
		{name: "scrape 2, threads: 1, batch size: 1", fields: fields{scraperTimeout: time.Minute, batchSize: 1, threads: 1, interval: time.Hour}, args: args{sites: []*htracker.Site{site1, site1a}}},
		{name: "scrape 1, threads: 2, batch size: 1", fields: fields{scraperTimeout: time.Minute, batchSize: 1, threads: 2, interval: time.Hour}, args: args{sites: []*htracker.Site{site1}}},
		{name: "scrape 1, threads: 1, batch size: 2", fields: fields{scraperTimeout: time.Minute, batchSize: 2, threads: 1, interval: time.Hour}, args: args{sites: []*htracker.Site{site1}}},
		{name: "scrape 1, threads: 2, batch size: 2", fields: fields{scraperTimeout: time.Minute, batchSize: 2, threads: 2, interval: time.Hour}, args: args{sites: []*htracker.Site{site1}}},
		{name: "scrape 2, threads: 2, batch size: 1", fields: fields{scraperTimeout: time.Minute, batchSize: 1, threads: 2, interval: time.Hour}, args: args{sites: []*htracker.Site{site1, site1a}}},
		{name: "scrape 4, threads: 4, batch size: 2", fields: fields{scraperTimeout: time.Minute, batchSize: 2, threads: 4, interval: time.Hour}, args: args{sites: []*htracker.Site{site1, site1a, site1b, site2}}},
		{name: "scrape 4, with timeout", fields: fields{scraperTimeout: time.Minute, batchSize: 1, threads: 1, interval: time.Millisecond}, args: args{sites: []*htracker.Site{site1, site1a, site1b, site2}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Watcher{
				archive:        service.NewSiteArchive(memory.NewSiteStorage(logger)),
				logger:         logger,
				interval:       tt.fields.interval,
				scraperTimeout: tt.fields.scraperTimeout,
				batchSize:      tt.fields.batchSize,
				threads:        tt.fields.threads,
			}
			err := w.RunScrapers(tt.args.sites)
			if (err != nil) != tt.wantErr {
				t.Errorf("Watcher.RunScrapers() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				for _, s := range tt.args.sites {
					if _, err := w.archive.Get(s); err != nil {
						t.Errorf("SiteArchive.Get(%s) error: %v", s.URL, err)
					}
				}
			}
		})
	}
}
