package service

import (
	"errors"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"golang.org/x/exp/slog"
)

func Test_ArchiveService_Update(t *testing.T) {

	storage := memory.NewSiteStorage(slog.Default())
	svc := NewSiteArchive(storage)

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site1.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")
	content1Updated := []byte("This is Site1 updated")

	date1 := time.Now()
	date2 := date1.Add(time.Second)

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
			checksum: Checksum(content1), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "add new site2", date: date1, subscription: sub2, content: content2,
			checksum: Checksum(content2), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "site1 unchanged", date: date2, subscription: sub1, content: content1,
			checksum: Checksum(content1), diffExpected: "",
			checkDateExpected: date2, updateDateExpected: date1},
		{name: "update site1", date: date2, subscription: sub1, content: content1Updated,
			checksum: Checksum(content1Updated), diffExpected: DiffText(string(content1), string(content1Updated)),
			checkDateExpected: date2, updateDateExpected: date2},
	}

	for _, tc := range testcases {
		diff, err := svc.Update(&htracker.Site{tc.subscription, tc.date, tc.date, tc.content, tc.checksum, ""})
		if err != nil {
			t.Fatalf("%s: archivesvc.Update() failed: %v", tc.name, err)
		}

		if want, got := tc.diffExpected, diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, tc.diffExpected, diff)
		}

		site, err := svc.Get(tc.subscription)
		if err != nil {
			t.Fatalf("%s: archivesvc.Get() failed: %v", tc.name, err)
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
		if want, got := tc.diffExpected, diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, want, got)
		}
	}

	_, err := svc.Get(&htracker.Subscription{URL: "http://does/not/exist", Filter: "some_filter", ContentType: "some_content_type"})
	if !errors.Is(err, htracker.ErrNotExist) {
		t.Fatalf("svc.Get(): Expected ErrNotExist error, got %v", err)
	}
}
