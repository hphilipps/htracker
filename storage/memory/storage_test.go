package memory

import (
	"context"
	"reflect"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

func Test_memDB_Get(t *testing.T) {
	type fields struct {
		archive []*htracker.Site
	}
	type args struct {
		subscription *htracker.Subscription
	}

	ctx := context.Background()
	date := time.Now()

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := &htracker.Subscription{URL: "http://site3.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")

	site1 := &htracker.Site{Subscription: sub1, LastUpdated: date, LastChecked: date,
		Content: content1, Checksum: service.Checksum(content1)}
	site2 := &htracker.Site{Subscription: sub2, LastUpdated: date, LastChecked: date,
		Content: content2, Checksum: service.Checksum(content2)}

	tests := []struct {
		name     string
		fields   fields
		args     args
		wantSite *htracker.Site
		wantErr  bool
	}{
		{name: "find 1 in 1", fields: fields{archive: []*htracker.Site{site1}},
			args: args{sub1}, wantSite: site1, wantErr: false},
		{name: "find 1 in 2", fields: fields{archive: []*htracker.Site{site1, site2}},
			args: args{sub2}, wantSite: site2, wantErr: false},
		{name: "find 0 in 2", fields: fields{archive: []*htracker.Site{site1, site2}},
			args: args{sub3}, wantSite: &htracker.Site{}, wantErr: true},
		{name: "find 0 in 0", fields: fields{archive: []*htracker.Site{}},
			args: args{sub3}, wantSite: &htracker.Site{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &memDB{
				archive: tt.fields.archive,
				logger:  slog.Default(),
			}
			gotSite, err := db.Get(ctx, tt.args.subscription)
			if (err != nil) != tt.wantErr {
				t.Errorf("memDB.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotSite, tt.wantSite) {
				t.Errorf("memDB.Get() = %v, want %v", gotSite, tt.wantSite)
			}
		})
	}
}

func Test_memDB_Add(t *testing.T) {
	type args struct {
		site *htracker.Site
	}

	ctx := context.Background()
	date := time.Now()

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")

	site1 := &htracker.Site{Subscription: sub1, LastUpdated: date, LastChecked: date, Content: content1, Checksum: service.Checksum(content1)}
	site2 := &htracker.Site{Subscription: sub2, LastUpdated: date, LastChecked: date, Content: content2, Checksum: service.Checksum(content2)}

	tests := []struct {
		name      string
		args      args
		wantErr   bool
		wantSites []*htracker.Site
	}{
		{name: "add site1", wantSites: []*htracker.Site{site1},
			args: args{site1}, wantErr: false},
		{name: "add site2", wantSites: []*htracker.Site{site1, site2},
			args: args{site2}, wantErr: false},
		{name: "add site2 again", wantSites: []*htracker.Site{site1, site2},
			args: args{site2}, wantErr: true},
	}

	db := &memDB{
		archive: []*htracker.Site{},
		logger:  slog.Default(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.Add(ctx, tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("memDB.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := db.archive; !reflect.DeepEqual(got, tt.wantSites) {
				t.Errorf("memDB.Add() = %v, want %v", got, tt.wantSites)
			}
		})
	}
}

func Test_memDB_GetSubscriber(t *testing.T) {
	type args struct {
		email string
	}

	ctx := context.Background()

	email1 := "email1"
	email2 := "email2"
	sub1 := &storage.Subscriber{Email: email1}
	sub2 := &storage.Subscriber{Email: email2}
	subscribers := []*storage.Subscriber{sub1, sub2}

	tests := []struct {
		name    string
		args    args
		want    *storage.Subscriber
		wantErr bool
	}{
		{name: "find email1", args: args{email: email1}, want: sub1},
		{name: "find email2", args: args{email: email2}, want: sub2},
		{name: "find nonexistent", args: args{email: "not_existing"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &memDB{
				subscribers: subscribers,
			}
			got, err := db.GetSubscriber(ctx, tt.args.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("memDB.GetSubscriber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("memDB.GetSubscriber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_memDB_AddSubscriber(t *testing.T) {
	type args struct {
		subscriber *storage.Subscriber
	}

	ctx := context.Background()

	sub1 := &storage.Subscriber{Email: "email1"}
	sub2 := &storage.Subscriber{Email: "email2"}

	tests := []struct {
		name            string
		args            args
		wantSubscribers []*storage.Subscriber
		wantErr         bool
	}{
		{name: "add subscriber1", args: args{subscriber: sub1}, wantSubscribers: []*storage.Subscriber{sub1}, wantErr: false},
		{name: "add subscriber2", args: args{subscriber: sub2}, wantSubscribers: []*storage.Subscriber{sub1, sub2}, wantErr: false},
		{name: "add existing subscriber", args: args{subscriber: sub1}, wantSubscribers: []*storage.Subscriber{sub1, sub2}, wantErr: true},
	}

	db := &memDB{
		subscribers: []*storage.Subscriber{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.AddSubscriber(ctx, tt.args.subscriber); (err != nil) != tt.wantErr {
				t.Errorf("memDB.AddSubscriber() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(db.subscribers, tt.wantSubscribers) {
				t.Errorf("Expected subscribers %v, got %v", db.subscribers, tt.wantSubscribers)
			}
		})
	}
}
