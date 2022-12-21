package htracker

import (
	"testing"
	"time"
)

func Test_SubscriptionEqual(t *testing.T) {
	sub1 := Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := Subscription{URL: "http://site1.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Minute}
	sub4 := Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", UseChrome: true, Interval: time.Minute}

	if want, got := false, sub1.Equals(&sub2); want != got {
		t.Fatalf("Expected sub1.Equals(sub2) == %v, got %v", want, got)
	}
	if want, got := true, sub1.Equals(&sub3); want != got {
		t.Fatalf("Expected sub1.Equals(sub3) == %v, got %v", want, got)
	}
	if want, got := false, sub1.Equals(&sub4); want != got {
		t.Fatalf("Expected sub1.Equals(sub4) == %v, got %v", want, got)
	}
}
