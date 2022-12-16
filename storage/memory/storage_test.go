package memory

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"golang.org/x/exp/slog"
)

func Test_memDB_Find(t *testing.T) {
	type fields struct {
		archive []*htracker.SiteContent
	}
	type args struct {
		site *htracker.Site
	}

	date := time.Now()

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	site3 := &htracker.Site{URL: "http://site3.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")

	sc1 := &htracker.SiteContent{Site: site1, LastUpdated: date, LastChecked: date,
		Content: content1, Checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1)))}
	sc2 := &htracker.SiteContent{Site: site2, LastUpdated: date, LastChecked: date,
		Content: content2, Checksum: fmt.Sprintf("%x", md5.Sum([]byte(content2)))}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantContent *htracker.SiteContent
		wantErr     bool
	}{
		{name: "find 1 in 1", fields: fields{archive: []*htracker.SiteContent{sc1}},
			args: args{site1}, wantContent: sc1, wantErr: false},
		{name: "find 1 in 2", fields: fields{archive: []*htracker.SiteContent{sc1, sc2}},
			args: args{site2}, wantContent: sc2, wantErr: false},
		{name: "find 0 in 2", fields: fields{archive: []*htracker.SiteContent{sc1, sc2}},
			args: args{site3}, wantContent: &htracker.SiteContent{}, wantErr: true},
		{name: "find 0 in 0", fields: fields{archive: []*htracker.SiteContent{}},
			args: args{site3}, wantContent: &htracker.SiteContent{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &memDB{
				archive: tt.fields.archive,
				logger:  slog.Default(),
			}
			gotContent, err := db.Find(tt.args.site)
			if (err != nil) != tt.wantErr {
				t.Errorf("memDB.Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotContent, tt.wantContent) {
				t.Errorf("memDB.Find() = %v, want %v", gotContent, tt.wantContent)
			}
		})
	}
}

func Test_memDB_Add(t *testing.T) {
	type args struct {
		content *htracker.SiteContent
	}

	date := time.Now()

	site1 := &htracker.Site{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	site2 := &htracker.Site{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")

	sc1 := &htracker.SiteContent{Site: site1, LastUpdated: date, LastChecked: date, Content: content1, Checksum: fmt.Sprintf("%x", md5.Sum([]byte(content1)))}
	sc2 := &htracker.SiteContent{Site: site2, LastUpdated: date, LastChecked: date, Content: content2, Checksum: fmt.Sprintf("%x", md5.Sum([]byte(content2)))}

	tests := []struct {
		name      string
		args      args
		wantErr   bool
		wantSites []*htracker.SiteContent
	}{
		{name: "add site1", wantSites: []*htracker.SiteContent{sc1},
			args: args{sc1}, wantErr: false},
		{name: "add site2", wantSites: []*htracker.SiteContent{sc1, sc2},
			args: args{sc2}, wantErr: false},
		{name: "add site2 again", wantSites: []*htracker.SiteContent{sc1, sc2},
			args: args{sc2}, wantErr: true},
	}

	db := &memDB{
		archive: []*htracker.SiteContent{},
		logger:  slog.Default(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.Add(tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("memDB.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := db.archive; !reflect.DeepEqual(got, tt.wantSites) {
				t.Errorf("memDB.Add() = %v, want %v", got, tt.wantSites)
			}
		})
	}
}
