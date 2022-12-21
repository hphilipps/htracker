package watcher

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/scraper"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"golang.org/x/exp/slog"
)

func TestWatcher_GenerateScrapeList(t *testing.T) {

	email1 := "email1@foo.bar"
	email2 := "email2@foo.bar"
	email3 := "email3@foo.bar"

	sub1 := &htracker.Subscription{URL: "site1.test", Filter: "filter1", ContentType: "text"}
	sub1a := &htracker.Subscription{URL: "site1.test", Filter: "filter2", ContentType: "text"}
	sub1b := &htracker.Subscription{URL: "site1.test", Filter: "filter1", ContentType: "html"}
	sub2 := &htracker.Subscription{URL: "site2.test", Filter: "filter1", ContentType: "text"}

	subscriber1 := &service.Subscriber{email1, []*htracker.Subscription{sub1}}
	subscriber2 := &service.Subscriber{email2, []*htracker.Subscription{sub1, sub1a, sub1b}}
	subscriber3 := &service.Subscriber{email3, []*htracker.Subscription{sub1, sub1a, sub1b, sub2}}

	tests := []struct {
		name              string
		subscribers       []*service.Subscriber
		wantSubscriptions []*htracker.Subscription
		wantErr           bool
	}{
		{name: "0 sites", subscribers: []*service.Subscriber{},
			wantSubscriptions: []*htracker.Subscription{}, wantErr: false},
		{name: "1 site", subscribers: []*service.Subscriber{subscriber1},
			wantSubscriptions: []*htracker.Subscription{sub1}, wantErr: false},
		{name: "same site with different filters", subscribers: []*service.Subscriber{subscriber2},
			wantSubscriptions: []*htracker.Subscription{sub1, sub1a, sub1b}, wantErr: false},
		{name: "multiple subscribers", subscribers: []*service.Subscriber{subscriber1, subscriber2, subscriber3},
			wantSubscriptions: []*htracker.Subscription{sub1, sub1a, sub1b, sub2}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.Default()
			db := memory.NewSubscriptionStorage(logger)
			svc := service.NewSubscriptionSvc(db)
			w := &Watcher{
				subSvc: svc,
				logger: logger,
			}

			for _, subscriber := range tt.subscribers {
				for _, subscription := range subscriber.Subscriptions {
					if err := svc.Subscribe(subscriber.Email, subscription); err != nil {
						t.Errorf("subscriber %s failed to subscribe to site %s during setup", subscriber.Email, subscription.URL)
					}
				}
			}

			gotSubsciptions, err := w.GenerateScrapeList()
			if (err != nil) != tt.wantErr {
				t.Errorf("Watcher.GenerateScrapeList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.wantSubscriptions) != 0 && len(gotSubsciptions) != 0 {
				if !reflect.DeepEqual(gotSubsciptions, tt.wantSubscriptions) {
					t.Errorf("Watcher.GenerateScrapeList() = %v, want %v", gotSubsciptions, tt.wantSubscriptions)
				}
			}
		})
	}
}

func TestWatcher_RunScrapers(t *testing.T) {
	type fields struct {
		interval  time.Duration
		batchSize int
		threads   int
	}
	type args struct {
		sites []*htracker.Subscription
	}

	sub1 := &htracker.Subscription{URL: "https://httpbin.org/anything", Filter: "filter1", ContentType: "text"}
	sub1a := &htracker.Subscription{URL: "https://httpbin.org/anything", Filter: "filter2", ContentType: "text"}
	sub1b := &htracker.Subscription{URL: "https://httpbin.org/anything", Filter: "filter1", ContentType: "html"}
	sub2 := &htracker.Subscription{URL: "https://httpbin.org/anything/2", Filter: "filter1", ContentType: "text"}

	handler := slog.HandlerOptions{Level: slog.LevelDebug}.NewTextHandler(os.Stdout)
	logger := slog.New(handler)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "scrape 0, threads: 1, batch size: 1",
			fields: fields{batchSize: 1, threads: 1, interval: time.Hour}, args: args{sites: []*htracker.Subscription{}}},
		{name: "scrape 1, threads: 1, batch size: 1",
			fields: fields{batchSize: 1, threads: 1, interval: time.Hour}, args: args{sites: []*htracker.Subscription{sub1}}},
		{name: "scrape 2, threads: 1, batch size: 1",
			fields: fields{batchSize: 1, threads: 1, interval: time.Hour}, args: args{sites: []*htracker.Subscription{sub1, sub1a}}},
		{name: "scrape 1, threads: 2, batch size: 1",
			fields: fields{batchSize: 1, threads: 2, interval: time.Hour}, args: args{sites: []*htracker.Subscription{sub1}}},
		{name: "scrape 1, threads: 1, batch size: 2",
			fields: fields{batchSize: 2, threads: 1, interval: time.Hour}, args: args{sites: []*htracker.Subscription{sub1}}},
		{name: "scrape 1, threads: 2, batch size: 2",
			fields: fields{batchSize: 2, threads: 2, interval: time.Hour}, args: args{sites: []*htracker.Subscription{sub1}}},
		{name: "scrape 2, threads: 2, batch size: 1",
			fields: fields{batchSize: 1, threads: 2, interval: time.Hour}, args: args{sites: []*htracker.Subscription{sub1, sub1a}}},
		{name: "scrape 4, threads: 4, batch size: 2",
			fields: fields{batchSize: 2, threads: 4, interval: time.Hour}, args: args{sites: []*htracker.Subscription{sub1, sub1a, sub1b, sub2}}},
		{name: "scrape 4, with timeout",
			fields: fields{batchSize: 1, threads: 1, interval: time.Millisecond}, args: args{sites: []*htracker.Subscription{sub1, sub1a, sub1b, sub2}},
			wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWatcher(
				service.NewSiteArchive(memory.NewSiteStorage(logger)), nil,
				WithLogger(logger),
				WithInterval(tt.fields.interval),
				WithBatchSize(tt.fields.batchSize),
				WithThreads(tt.fields.threads),
				WithScraperOpts(scraper.WithTimeout(time.Minute)))

			err := w.RunScrapers(context.Background(), tt.args.sites)
			if (err != nil) != tt.wantErr {
				t.Errorf("Watcher.RunScrapers() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				for _, s := range tt.args.sites {
					if _, err := w.archive.Get(s); err != nil {
						t.Errorf("SiteArchiveSvc.Get(%s) error: %v", s.URL, err)
					}
				}
			}
		})
	}
}
