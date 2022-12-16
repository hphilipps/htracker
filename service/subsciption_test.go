package service

import (
	"reflect"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"golang.org/x/exp/slog"
)

func TestSubscriptionSvc_Subscribe(t *testing.T) {

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

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := svc.Subscribe(tt.args.email, tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.Subscribe() error = %v, wantErr %v", err, tt.wantErr)
			}

			sites, err := svc.GetSitesBySubscriber(tt.args.email)
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

func TestSubscriptionSvc_Unsubscribe(t *testing.T) {

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

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	svc.Subscribe(email1, site1)
	svc.Subscribe(email1, site2)
	svc.Subscribe(email1, site3)
	svc.Subscribe(email2, site1)
	svc.Subscribe(email2, site2)
	svc.Subscribe(email3, site3)
	svc.Unsubscribe(email3, site3) // should leave email3 with 0 subscriptions

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

			if err := svc.Unsubscribe(tt.args.email, tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.Unsubscribe() error = %v, wantErr %v", err, tt.wantErr)
			}

			sites, err := svc.GetSitesBySubscriber(tt.args.email)
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

func TestSubscriptionSvc_GetSitesBySubscriber(t *testing.T) {

	type args struct {
		email string
	}

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	svc.Subscribe(email1, site1)
	svc.Subscribe(email1, site2)
	svc.Subscribe(email1, site3)
	svc.Subscribe(email2, site1)
	svc.Subscribe(email2, site2)
	svc.Subscribe(email3, site3)
	svc.Unsubscribe(email3, site3) // should leave email3 with 0 subscriptions

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
			gotSites, err := svc.GetSitesBySubscriber(tt.args.email)
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

func TestSubscriptionSvc_GetSubscribersBySite(t *testing.T) {

	type args struct {
		site *htracker.Site
	}

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	svc.Subscribe(email1, site1)
	svc.Subscribe(email1, site2)
	svc.Subscribe(email1, site3)
	svc.Subscribe(email2, site1)
	svc.Subscribe(email2, site2)
	svc.Subscribe(email3, site3)
	svc.Unsubscribe(email3, site3) // should leave email3 with 0 subscriptions

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

			subscribers, err := svc.GetSubscribersBySite(tt.args.site)
			if (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.GetSubscribersBySite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !(len(tt.wantEmails) == 0 && len(subscribers) == 0) {
				gotEmails := []string{}
				for _, s := range subscribers {
					gotEmails = append(gotEmails, s.Email)
				}
				if !reflect.DeepEqual(gotEmails, tt.wantEmails) {
					t.Errorf("MemoryDB.GetSubscribersBySite() = %v, want %v", gotEmails, tt.wantEmails)
				}
			}
		})
	}
}

func TestSubscriptionSvc_GetSubscribers(t *testing.T) {

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	svc.Subscribe(email1, site1)
	svc.Subscribe(email1, site2)
	svc.Subscribe(email1, site3)
	svc.Subscribe(email2, site1)
	svc.Subscribe(email2, site2)
	svc.Subscribe(email3, site3)
	svc.Unsubscribe(email3, site3) // should leave email3 with 0 subscriptions

	subscribers, err := svc.GetSubscribers()
	if err != nil {
		t.Errorf("MemoryDB.GetSubscribers() failed: %v", err)
	}
	gotEmails := []string{}
	for _, sub := range subscribers {
		gotEmails = append(gotEmails, sub.Email)
	}
	if !reflect.DeepEqual(gotEmails, []string{email1, email2, email3}) {
		t.Errorf("MemoryDB.GetSubscribers() expected %v, got %v", []string{email1, email2, email3}, gotEmails)
	}
}

func TestSubscriptionSvc_DeleteSubscriber(t *testing.T) {

	type args struct {
		email string
	}

	email1 := "foo@bar.test"

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	err := svc.Subscribe(email1, &htracker.Site{URL: "some.web.site.test/blah", Filter: "someFilter", ContentType: "text"})
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
			if err := svc.DeleteSubscriber(tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("MemoryDB.DeleteSubscriber() error = %v, wantErr %v", err, tt.wantErr)
			}
			subscribers, err := svc.GetSubscribers()
			if err != nil {
				t.Errorf("MemoryDB.GetSubscribers() failed: %v", err)
			}
			found := false
			for _, sub := range subscribers {
				if (sub.Email == email1) == tt.wantExist {
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
