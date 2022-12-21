package exporter

import (
	"context"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"golang.org/x/exp/slog"
)

func TestExporter_Export(t *testing.T) {

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Minute}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")
	content1Updated := []byte("This is Site1 updated")

	date1 := time.Now()
	date2 := date1.Add(time.Second)
	date3 := date2.Add(time.Second)

	testcases := []struct {
		name               string
		date               time.Time
		subscription       *htracker.Subscription
		content            []byte
		checksum           string
		diffExpected       string
		checkDateExpected  time.Time
		updateDateExpected time.Time
	}{
		{name: "add new site1", date: date1, subscription: sub1, content: content1,
			checksum: service.Checksum(content1), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "add new site2", date: date1, subscription: sub2, content: content2,
			checksum: service.Checksum(content2), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "site1 unchanged", date: date2, subscription: sub1, content: content1,
			checksum: service.Checksum(content1), diffExpected: "",
			checkDateExpected: date2, updateDateExpected: date1},
		{name: "update site1", date: date3, subscription: sub3, content: content1Updated,
			checksum: service.Checksum(content1Updated), diffExpected: service.DiffText(string(content1),
				string(content1Updated)), checkDateExpected: date3, updateDateExpected: date3},
	}

	ctx := context.Background()
	exports := make(chan interface{}, 1)
	storage := memory.NewSiteStorage(slog.Default())
	archive := service.NewSiteArchive(storage)
	exporter := NewExporter(ctx, archive)

	// run exporter in background
	go func() {
		err := exporter.Export(exports)
		if err != nil {
			t.Errorf("Exporter failed to export: %v", err)
		}
	}()

	for _, tc := range testcases {
		// simulate sending result from scraper and wait a bit for the DB to get updated
		exports <- &htracker.Site{Subscription: tc.subscription, LastUpdated: tc.date, LastChecked: tc.date,
			Content: tc.content, Checksum: service.Checksum(tc.content)}
		time.Sleep(time.Millisecond)

		site, err := archive.Get(tc.subscription)
		if err != nil {
			t.Fatalf("%s: svc.Get() failed: %v", tc.name, err)
		}

		if want, got := tc.updateDateExpected, site.LastUpdated; want != got {
			t.Fatalf("%s: Expected lastUpdated %s, got %s", tc.name, want, got)
		}
		if want, got := tc.checkDateExpected, site.LastChecked; want != got {
			t.Fatalf("%s: Expected lastChecked %s, got %s", tc.name, want, got)
		}
		if want, got := string(tc.content), string(site.Content); want != got {
			t.Fatalf("%s: Expected content %s, got %s", tc.name, want, got)
		}
		if want, got := tc.checksum, site.Checksum; want != got {
			t.Fatalf("%s: Expected checksum %s, got %s", tc.name, want, got)
		}
		if want, got := tc.diffExpected, site.Diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, want, got)
		}
	}
}
