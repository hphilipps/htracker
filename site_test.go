package htracker

import (
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
