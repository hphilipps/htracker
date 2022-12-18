package exporter

import (
	"context"
	"crypto/md5"
	"fmt"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/service/memory"
)

func TestExporter_Export(t *testing.T) {

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Minute}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")
	content1Updated := []byte("This is Site1 updated")

	date1 := time.Now()
	date2 := date1.Add(time.Second)
	date3 := date2.Add(time.Second)

	testcases := []struct {
		name               string
		date               time.Time
		site               *htracker.Site
		content            []byte
		checksum           string
		diffExpected       string
		checkDateExpected  time.Time
		updateDateExpected time.Time
	}{
		{name: "add new site1", date: date1, site: site1, content: content1,
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1))), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "add new site2", date: date1, site: site2, content: content2,
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content2))), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "site1 unchanged", date: date2, site: site1, content: content1,
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1))), diffExpected: "",
			checkDateExpected: date2, updateDateExpected: date1},
		{name: "update site1", date: date3, site: site3, content: content1Updated,
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1Updated))), diffExpected: service.DiffText(string(content1), string(content1Updated)),
			checkDateExpected: date3, updateDateExpected: date3},
	}

	ctx := context.Background()
	exports := make(chan interface{}, 1)
	db := memory.NewMemoryDB()
	exporter := NewExporter(ctx, db)

	// run exporter in background
	go func() {
		err := exporter.Export(exports)
		if err != nil {
			t.Errorf("Exporter failed to export: %v", err)
		}
	}()

	for _, tc := range testcases {

		// simulate sending result from scraper and wait a bit for the DB to get updated
		exports <- &htracker.SiteArchive{Site: tc.site, LastUpdated: tc.date, LastChecked: tc.date, Content: tc.content, Checksum: fmt.Sprintf("%x", md5.Sum([]byte(tc.content)))}
		time.Sleep(time.Millisecond)

		sa, err := db.Get(tc.site)
		if err != nil {
			t.Fatalf("%s: db.GetSiteArchive failed: %v", tc.name, err)
		}

		if want, got := tc.updateDateExpected, sa.LastUpdated; want != got {
			t.Fatalf("%s: Expected lastUpdated %s, got %s", tc.name, want, got)
		}
		if want, got := tc.checkDateExpected, sa.LastChecked; want != got {
			t.Fatalf("%s: Expected lastChecked %s, got %s", tc.name, want, got)
		}
		if want, got := string(tc.content), string(sa.Content); want != got {
			t.Fatalf("%s: Expected content %s, got %s", tc.name, want, got)
		}
		if want, got := tc.checksum, sa.Checksum; want != got {
			t.Fatalf("%s: Expected checksum %s, got %s", tc.name, want, got)
		}
		if want, got := tc.diffExpected, sa.Diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, want, got)
		}
	}
}
