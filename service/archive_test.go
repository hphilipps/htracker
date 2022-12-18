package service

import (
	"errors"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"golang.org/x/exp/slog"
)

func ArchiveService_UpdateSiteArchive(t *testing.T) {

	storage := memory.NewSiteStorage(slog.Default())
	svc := NewSiteArchive(storage)

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site1.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")
	content1Updated := []byte("This is Site1 updated")

	date1 := time.Now()
	date2 := date1.Add(time.Second)

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
			checksum: Checksum(content1), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "add new site2", date: date1, site: site2, content: content2,
			checksum: Checksum(content2), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "site1 unchanged", date: date2, site: site1, content: content1,
			checksum: Checksum(content1), diffExpected: "",
			checkDateExpected: date2, updateDateExpected: date1},
		{name: "update site1", date: date2, site: site1, content: content1Updated,
			checksum: Checksum(content1Updated), diffExpected: DiffText(string(content1), string(content1Updated)),
			checkDateExpected: date2, updateDateExpected: date2},
	}

	for _, tc := range testcases {
		diff, err := svc.Update(&htracker.SiteContent{tc.site, tc.date, tc.date, tc.content, tc.checksum, ""})
		if err != nil {
			t.Fatalf("%s: db.UpdateSiteArchive failed: %v", tc.name, err)
		}

		if want, got := tc.diffExpected, diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, tc.diffExpected, diff)
		}

		sc, err := svc.Get(tc.site)
		if err != nil {
			t.Fatalf("%s: db.GetSiteArchive failed: %v", tc.name, err)
		}

		if want, got := tc.updateDateExpected, sc.LastUpdated; want != got {
			t.Fatalf("%s: Expected lastUpdated %s, got %s", tc.name, want, got)
		}
		if want, got := tc.checkDateExpected, sc.LastChecked; want != got {
			t.Fatalf("%s: Expected lastChecked %s, got %s", tc.name, want, got)
		}
		if want, got := string(tc.content), string(sc.Content); want != got {
			t.Fatalf("%s: Expected content %s, got %s", tc.name, want, got)
		}
		if want, got := tc.checksum, sc.Checksum; want != got {
			t.Fatalf("%s: Expected checksum %s, got %s", tc.name, want, got)
		}
		if want, got := tc.diffExpected, diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, want, got)
		}
	}

	_, err := svc.Get(&htracker.Site{URL: "http://does/not/exist", Filter: "some_filter", ContentType: "some_content_type"})
	if !errors.Is(err, htracker.ErrNotExist) {
		t.Fatalf("GetSiteArchive: Expected ErrNotExist error, got %v", err)
	}
}
