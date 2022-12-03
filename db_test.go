package htracker

import (
	"crypto/md5"
	"fmt"
	"testing"
	"time"
)

func TestDB_Equal(t *testing.T) {
	site1 := Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := Site{URL: "http://site1.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Minute}

	if want, got := false, site1.Equals(&site2); want != got {
		t.Fatalf("Expected site1.Equals(site2) == %v, got %v", want, got)
	}
	if want, got := true, site1.Equals(&site3); want != got {
		t.Fatalf("Expected site1.Equals(site3) == %v, got %v", want, got)
	}
}

func TestMemoryDB_UpdateSite(t *testing.T) {

	db := NewMemoryDB()

	site1 := &Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &Site{URL: "http://site1.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")
	content1Updated := []byte("This is Site1 updated")

	date1 := time.Now()
	date2 := date1.Add(time.Second)

	testcases := []struct {
		name               string
		date               time.Time
		site               *Site
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
		{name: "update site1", date: date2, site: site1, content: content1Updated,
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1Updated))), diffExpected: diffText(string(content1), string(content1Updated)),
			checkDateExpected: date2, updateDateExpected: date2},
	}

	for _, tc := range testcases {
		diff, err := db.UpdateSite(tc.date, *tc.site, tc.content, tc.checksum)
		if err != nil {
			t.Fatalf("%s: db.UpdateSite failed: %v", tc.name, err)
		}

		if want, got := tc.diffExpected, diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, tc.diffExpected, diff)
		}

		lastUpdated, lastChecked, content, checksum, diff, err := db.GetSite(tc.site.URL, tc.site.Filter, tc.site.ContentType)
		if err != nil {
			t.Fatalf("%s: db.GetSite failed: %v", tc.name, err)
		}

		if want, got := tc.updateDateExpected, lastUpdated; want != got {
			t.Fatalf("%s: Expected lastUpdated %s, got %s", tc.name, want, got)
		}
		if want, got := tc.checkDateExpected, lastChecked; want != got {
			t.Fatalf("%s: Expected lastChecked %s, got %s", tc.name, want, got)
		}
		if want, got := string(tc.content), string(content); want != got {
			t.Fatalf("%s: Expected content %s, got %s", tc.name, want, got)
		}
		if want, got := tc.checksum, checksum; want != got {
			t.Fatalf("%s: Expected checksum %s, got %s", tc.name, want, got)
		}
		if want, got := tc.diffExpected, diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, want, got)
		}
	}

	_, _, _, _, _, err := db.GetSite("http://does/not/exist", "some_filter", "some_content_type")
	if err != ErrNotExist {
		t.Fatalf("GetSite: Expected ErrNotExist error, got %v", err)
	}

}
