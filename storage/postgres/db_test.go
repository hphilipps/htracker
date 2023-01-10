package postgres

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"golang.org/x/exp/slog"
)

const intTestVarName = "INTEGRATION_TESTS"

func runIntegrationTests() bool {
	intTestVar := os.Getenv(intTestVarName)

	if run, err := strconv.ParseBool(intTestVar); err != nil || !run {
		return false
	}

	return true
}

func TestNew_Integration(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{name: "connect uri", uri: "postgresql://postgres:pg1pw@localhost?sslmode=disable"},
		{name: "connect string", uri: "host=localhost port=5432 dbname=postgres user=postgres password=pg1pw sslmode=disable"},
		{name: "wrong password", uri: "postgresql://postgres:wrong_pw@localhost?sslmode=disable", wantErr: true},
	}

	logger := slog.Default()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.uri, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestAdd(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	date1 := time.Now()
	date2 := date1.Add(time.Second)
	sub1 := &htracker.Subscription{URL: "site1"}
	sub2 := &htracker.Subscription{URL: "site2"}
	sub3 := &htracker.Subscription{URL: "nonexisting"}
	site1 := &htracker.Site{
		Subscription: sub1,
		LastUpdated:  date1,
		LastChecked:  date1,
		Content:      []byte("content1ä😎"),
		Checksum:     "1234",
		Diff:         "diff1ä",
	}
	site2 := &htracker.Site{
		Subscription: sub2,
		LastUpdated:  date2,
		LastChecked:  date2,
		Content:      []byte(""),
		Checksum:     "5678",
		Diff:         "",
	}

	tests := []struct {
		name         string
		subscription *htracker.Subscription
		wantSite     *htracker.Site
		wantAddErr   bool
		wantGetErr   bool
	}{
		{name: "add site1", subscription: sub1, wantSite: site1},
		{name: "add site2", subscription: sub2, wantSite: site2},
		{name: "add duplicate site", subscription: sub1, wantSite: site1, wantAddErr: true},
		{name: "get non-existing site", subscription: sub3, wantSite: site1, wantAddErr: true, wantGetErr: true},
	}

	ctx := context.Background()
	logger := slog.Default()
	conn, err := New("postgresql://postgres:pg1pw@localhost?sslmode=disable", logger)
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := conn.Add(ctx, tt.wantSite)
			if (err != nil) != tt.wantAddErr {
				t.Errorf("Add() error = %v, wantAddErr %v", err, tt.wantAddErr)
				return
			}

			got, err := conn.Get(ctx, tt.subscription)
			if (err != nil) != tt.wantGetErr {
				t.Errorf("Get() error = %v, wantGetErr %v", err, tt.wantGetErr)
				return
			}
			if !tt.wantGetErr {
				if !reflect.DeepEqual(got.Subscription, tt.subscription) {
					t.Errorf("postgres.Add() subscription = %v, want %v", got.Subscription, tt.subscription)
				}
				if string(tt.wantSite.Content) != string(got.Content) {
					t.Errorf("postgres.Add() content = %v, want %v", got.Content, tt.wantSite.Content)
				}
				if !tt.wantSite.LastUpdated.Equal(got.LastUpdated) {
					t.Errorf("postgres.Add() LastUpdated = %v, want %v", got.LastUpdated, tt.wantSite.LastUpdated)
				}
				if !tt.wantSite.LastChecked.Equal(got.LastChecked) {
					t.Errorf("postgres.Add() LastChecked = %v, want %v", got.LastChecked, tt.wantSite.LastChecked)
				}
				if tt.wantSite.Checksum != got.Checksum {
					t.Errorf("postgres.Add() checksum = %v, want %v", got.Checksum, tt.wantSite.Checksum)
				}
				if tt.wantSite.Diff != got.Diff {
					t.Errorf("postgres.Add() diff = %v, want %v", got.Diff, tt.wantSite.Diff)
				}
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	date1 := time.Now()
	date2 := date1.Add(time.Second)
	sub1 := &htracker.Subscription{URL: "updsite1"}
	sub2 := &htracker.Subscription{URL: "updsite2"}
	site1 := &htracker.Site{
		Subscription: sub1,
		LastUpdated:  date1,
		LastChecked:  date1,
		Content:      []byte("content1ä😎"),
		Checksum:     "1234",
		Diff:         "diff1ä",
	}
	site1_updated := &htracker.Site{
		Subscription: sub1,
		LastUpdated:  date1,
		LastChecked:  date1,
		Content:      []byte("content1_updated"),
		Checksum:     "12345",
		Diff:         "diff1",
	}
	site2 := &htracker.Site{
		Subscription: sub2,
		LastUpdated:  date2,
		LastChecked:  date2,
		Content:      []byte(""),
		Checksum:     "5678",
		Diff:         "",
	}
	site2_updated := &htracker.Site{
		Subscription: sub2,
		LastUpdated:  date2,
		LastChecked:  date2,
		Content:      []byte("content2_updated"),
		Checksum:     "56789",
		Diff:         "content2_updated",
	}

	tests := []struct {
		name         string
		subscription *htracker.Subscription
		wantSite     *htracker.Site
		addSite      bool
		wantErr      bool
		wantGetErr   bool
	}{
		{name: "update site1 with same content", subscription: sub1, addSite: true, wantSite: site1},
		{name: "update site1 with new content", subscription: sub1, wantSite: site1_updated},
		{name: "update non-existing site2", subscription: sub2, wantSite: site2, wantErr: true, wantGetErr: true},
		{name: "update site2", subscription: sub2, addSite: true, wantSite: site2_updated},
	}

	ctx := context.Background()
	logger := slog.Default()
	conn, err := New("postgresql://postgres:pg1pw@localhost?sslmode=disable", logger)
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.addSite {
				if err := conn.Add(ctx, tt.wantSite); err != nil {
					t.Errorf("Failed to add site: %v", err)
					return
				}
			}

			err := conn.Update(ctx, tt.wantSite)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := conn.Get(ctx, tt.subscription)
			if (err != nil) != tt.wantGetErr {
				t.Errorf("Get() error = %v, wantGetErr %v", err, tt.wantGetErr)
				return
			}
			if !tt.wantGetErr {
				if !reflect.DeepEqual(got.Subscription, tt.subscription) {
					t.Errorf("postgres.Add() subscription = %v, want %v", got.Subscription, tt.subscription)
				}
				if string(tt.wantSite.Content) != string(got.Content) {
					t.Errorf("postgres.Add() content = %v, want %v", got.Content, tt.wantSite.Content)
				}
				if !tt.wantSite.LastUpdated.Equal(got.LastUpdated) {
					t.Errorf("postgres.Add() LastUpdated = %v, want %v", got.LastUpdated, tt.wantSite.LastUpdated)
				}
				if !tt.wantSite.LastChecked.Equal(got.LastChecked) {
					t.Errorf("postgres.Add() LastChecked = %v, want %v", got.LastChecked, tt.wantSite.LastChecked)
				}
				if tt.wantSite.Checksum != got.Checksum {
					t.Errorf("postgres.Add() checksum = %v, want %v", got.Checksum, tt.wantSite.Checksum)
				}
				if tt.wantSite.Diff != got.Diff {
					t.Errorf("postgres.Add() diff = %v, want %v", got.Diff, tt.wantSite.Diff)
				}
			}
		})
	}
}
