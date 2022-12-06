package memory

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
)

func TestDB_Equal(t *testing.T) {
	site1 := htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := htracker.Site{URL: "http://site1.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Minute}

	if want, got := false, site1.Equals(&site2); want != got {
		t.Fatalf("Expected site1.Equals(site2) == %v, got %v", want, got)
	}
	if want, got := true, site1.Equals(&site3); want != got {
		t.Fatalf("Expected site1.Equals(site3) == %v, got %v", want, got)
	}
}

func TestMemoryDB_UpdateSiteArchive(t *testing.T) {

	db := NewMemoryDB()

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
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1))), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "add new site2", date: date1, site: site2, content: content2,
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content2))), diffExpected: "",
			checkDateExpected: date1, updateDateExpected: date1},
		{name: "site1 unchanged", date: date2, site: site1, content: content1,
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1))), diffExpected: "",
			checkDateExpected: date2, updateDateExpected: date1},
		{name: "update site1", date: date2, site: site1, content: content1Updated,
			checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1Updated))), diffExpected: service.DiffText(string(content1), string(content1Updated)),
			checkDateExpected: date2, updateDateExpected: date2},
	}

	for _, tc := range testcases {
		diff, err := db.Update(&htracker.SiteArchive{tc.site, tc.date, tc.date, tc.content, tc.checksum, ""})
		if err != nil {
			t.Fatalf("%s: db.UpdateSiteArchive failed: %v", tc.name, err)
		}

		if want, got := tc.diffExpected, diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, tc.diffExpected, diff)
		}

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
		if want, got := tc.diffExpected, diff; want != got {
			t.Fatalf("%s: Expected diff %s, got %s", tc.name, want, got)
		}
	}

	_, err := db.Get(&htracker.Site{URL: "http://does/not/exist", Filter: "some_filter", ContentType: "some_content_type"})
	if err != htracker.ErrNotExist {
		t.Fatalf("GetSiteArchive: Expected ErrNotExist error, got %v", err)
	}
}

func TestMemoryDB_Subscribe(t *testing.T) {

	type args struct {
		email string
		site  *htracker.Site
	}

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	tests := []struct {
		name      string
		args      args
		wantSites []*htracker.Site
		wantErr   bool
	}{
		{name: "subscribe email1 to site1", args: args{email: email1, site: site1}, wantSites: []*htracker.Site{site1}, wantErr: false},
		{name: "subscribe email2 to site1", args: args{email: email2, site: site1}, wantSites: []*htracker.Site{site1}, wantErr: false},
		{name: "subscribe email3 to site1", args: args{email: email3, site: site1}, wantSites: []*htracker.Site{site1}, wantErr: false},
		{name: "subscribe email1 to site2", args: args{email: email1, site: site2}, wantSites: []*htracker.Site{site1, site2}, wantErr: false},
		{name: "subscribe email2 to site2", args: args{email: email2, site: site2}, wantSites: []*htracker.Site{site1, site2}, wantErr: false},
		{name: "subscribe email1 to same site again", args: args{email: email1, site: site1}, wantSites: []*htracker.Site{site1, site2}, wantErr: true},
		{name: "subscribe email1 to equal site again", args: args{email: email1, site: site3}, wantSites: []*htracker.Site{site1, site2}, wantErr: true},
	}

	db := NewMemoryDB()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := db.Subscribe(tt.args.email, tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.Subscribe() error = %v, wantErr %v", err, tt.wantErr)
			}

			sites, err := db.GetSitesBySubscriber(tt.args.email)
			if err != nil {
				t.Errorf("MemoryDB.Subscribe() - validation with MemoryDB.GetSitesBySubscriber() failed: %v", err)
			}

			if len(tt.wantSites) != len(sites) {
				t.Errorf("Expected %d subscribed sites for %s, got %d", len(tt.wantSites), tt.args.email, len(sites))
			}
			for _, i := range tt.wantSites {
				found := false
				for _, j := range sites {
					if i.Equals(j) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find site %s in subscriptions of %s", i.URL, tt.args.email)
					break
				}
			}
		})
	}
}

func TestMemoryDB_Unsubscribe(t *testing.T) {

	type args struct {
		email string
		site  *htracker.Site
	}

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	db := NewMemoryDB()

	db.Subscribe(email1, site1)
	db.Subscribe(email1, site2)
	db.Subscribe(email1, site3)
	db.Subscribe(email2, site1)
	db.Subscribe(email2, site2)
	db.Subscribe(email3, site3)
	db.Unsubscribe(email3, site3) // should leave email3 with 0 subscriptions

	tests := []struct {
		name      string
		args      args
		wantSites []*htracker.Site
		wantErr   bool
		wantEmail bool
	}{
		{name: "unsubscribe email1 from site1", args: args{email: email1, site: site1}, wantSites: []*htracker.Site{site2, site3}, wantErr: false, wantEmail: true},
		{name: "unsubscribe email2 from site1", args: args{email: email2, site: site1}, wantSites: []*htracker.Site{site2}, wantErr: false, wantEmail: true},
		{name: "unsubscribe email3 from not subscribed site1", args: args{email: email3, site: site1}, wantSites: []*htracker.Site{}, wantErr: true, wantEmail: true},
		{name: "unsubscribe email1 from site2", args: args{email: email1, site: site2}, wantSites: []*htracker.Site{site3}, wantErr: false, wantEmail: true},
		{name: "unsubscribe email2 from site2", args: args{email: email2, site: site2}, wantSites: []*htracker.Site{}, wantErr: false, wantEmail: true},
		{name: "unsubscribe email1 from site2 again", args: args{email: email1, site: site2}, wantSites: []*htracker.Site{site3}, wantErr: true, wantEmail: true},
		{name: "unsubscribe email3 from not subscribed site3", args: args{email: email3, site: site3}, wantSites: []*htracker.Site{}, wantErr: true, wantEmail: true},
		{name: "unsubscribe nonexistent subscriber from site1", args: args{email: "nothing@foo.test", site: site1}, wantSites: []*htracker.Site{}, wantErr: true, wantEmail: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := db.Unsubscribe(tt.args.email, tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.Unsubscribe() error = %v, wantErr %v", err, tt.wantErr)
			}

			sites, err := db.GetSitesBySubscriber(tt.args.email)
			if err != nil && tt.wantEmail == true {
				t.Errorf("MemoryDB.Unsubscribe() - validation with MemoryDB.GetSitesBySubscriber() failed: %v", err)
			}

			if len(tt.wantSites) != len(sites) {
				t.Errorf("Expected %d subscribed sites for %s, got %d", len(tt.wantSites), tt.args.email, len(sites))
			}
			for _, i := range tt.wantSites {
				found := false
				for _, j := range sites {
					if i.Equals(j) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find site %s in subscriptions of %s", i.URL, tt.args.email)
					break
				}
			}
		})
	}
}

func TestMemoryDB_GetSitesBySubscriber(t *testing.T) {

	type args struct {
		email string
	}

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	db := NewMemoryDB()

	db.Subscribe(email1, site1)
	db.Subscribe(email1, site2)
	db.Subscribe(email1, site3)
	db.Subscribe(email2, site1)
	db.Subscribe(email2, site2)
	db.Subscribe(email3, site3)
	db.Unsubscribe(email3, site3) // should leave email3 with 0 subscriptions

	tests := []struct {
		name      string
		args      args
		wantSites []*htracker.Site
		wantErr   bool
	}{
		{name: "get email1 subscriptions", args: args{email: email1}, wantSites: []*htracker.Site{site1, site2, site3}, wantErr: false},
		{name: "get email2 subscriptions", args: args{email: email2}, wantSites: []*htracker.Site{site1, site2}, wantErr: false},
		{name: "get email3 subscriptions", args: args{email: email3}, wantSites: []*htracker.Site{}, wantErr: false},
		{name: "get subscriptions of nonexistent email", args: args{email: "nonexisting@foo.test"}, wantSites: []*htracker.Site{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSites, err := db.GetSitesBySubscriber(tt.args.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.GetSitesBySubscriber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(gotSites, tt.wantSites) {
					t.Errorf("MemoryDB.GetSitesBySubscriber() = %v, want %v", gotSites, tt.wantSites)
				}
			}
		})
	}
}

func TestMemoryDB_GetSubscribersBySite(t *testing.T) {

	type args struct {
		site *htracker.Site
	}

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	db := NewMemoryDB()

	db.Subscribe(email1, site1)
	db.Subscribe(email1, site2)
	db.Subscribe(email1, site3)
	db.Subscribe(email2, site1)
	db.Subscribe(email2, site2)
	db.Subscribe(email3, site3)
	db.Unsubscribe(email3, site3) // should leave email3 with 0 subscriptions

	tests := []struct {
		name       string
		args       args
		wantEmails []string
		wantErr    bool
	}{
		{name: "get site1 subscribers", args: args{site: site1}, wantEmails: []string{email1, email2}, wantErr: false},
		{name: "get site2 subscribers", args: args{site: site2}, wantEmails: []string{email1, email2}, wantErr: false},
		{name: "get site3 subscribers", args: args{site: site3}, wantEmails: []string{email1}, wantErr: false},
		{name: "get subscribers to nonexistent site", args: args{site: &htracker.Site{URL: "nowhere.test/foo"}}, wantEmails: []string{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gotEmails, err := db.GetSubscribersBySite(tt.args.site)
			if (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.GetSubscribersBySite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !(len(tt.wantEmails) == 0 && len(gotEmails) == 0) {
				if !reflect.DeepEqual(gotEmails, tt.wantEmails) {
					t.Errorf("MemoryDB.GetSubscribersBySite() = %v, want %v", gotEmails, tt.wantEmails)
				}
			}
		})
	}
}

func TestMemoryDB_GetSubscribers(t *testing.T) {

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	db := NewMemoryDB()

	db.Subscribe(email1, site1)
	db.Subscribe(email1, site2)
	db.Subscribe(email1, site3)
	db.Subscribe(email2, site1)
	db.Subscribe(email2, site2)
	db.Subscribe(email3, site3)
	db.Unsubscribe(email3, site3) // should leave email3 with 0 subscriptions

	emails, err := db.GetSubscribers()
	if err != nil {
		t.Errorf("MemoryDB.GetSubscribers() failed: %v", err)
	}
	if !reflect.DeepEqual(emails, []string{email1, email2, email3}) {
		t.Errorf("MemoryDB.GetSubscribers() expected %v, got %v", []string{email1, email2, email3}, emails)
	}
}

func TestMemoryDB_DeleteSubscriber(t *testing.T) {

	type args struct {
		email string
	}

	email1 := "foo@bar.test"

	db := NewMemoryDB()
	err := db.Subscribe(email1, &htracker.Site{URL: "some.web.site.test/blah", Filter: "someFilter", ContentType: "text"})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	tests := []struct {
		name      string
		args      args
		wantErr   bool
		wantExist bool
	}{
		{name: "delete nonexistent subscriber", args: args{email: "notexisting@foo.bar"}, wantErr: true, wantExist: true},
		{name: "delete subscriber", args: args{email: email1}, wantErr: false, wantExist: false},
		{name: "delete subscriber again", args: args{email: email1}, wantErr: true, wantExist: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.DeleteSubscriber(tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.DeleteSubscriber() error = %v, wantErr %v", err, tt.wantErr)
			}
			emails, err := db.GetSubscribers()
			if err != nil {
				t.Errorf("MemoryDB.GetSubscribers() failed: %v", err)
			}
			found := false
			for _, e := range emails {
				if (e == email1) == tt.wantExist {
					found = true
					break
				}
			}
			if !found && tt.wantExist {
				t.Errorf("MemoryDB.DeleteSubscriber() expected entry to still exist but it is gone")
			}
		})
	}
}
