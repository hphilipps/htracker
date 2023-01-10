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

func TestGet(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	date := time.Now()
	sub1 := &htracker.Subscription{URL: "site1"}
	sub2 := &htracker.Subscription{URL: "site2"}
	site1 := &htracker.Site{Subscription: sub1, LastUpdated: date, LastChecked: date, Content: []byte("content1")}

	tests := []struct {
		name         string
		subscription *htracker.Subscription
		wantSite     *htracker.Site
		wantErr      bool
	}{
		{name: "get site1", subscription: sub1, wantSite: site1},
		{name: "get non-existing site", subscription: sub2, wantErr: true},
	}

	ctx := context.Background()
	logger := slog.Default()
	conn, err := New("postgresql://postgres:pg1pw@localhost?sslmode=disable", logger)
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}

	if err := conn.Add(ctx, site1); err != nil {
		t.Errorf("Setup: failed to Add site: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conn.Get(ctx, tt.subscription)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got.Subscription, tt.wantSite.Subscription) {
					t.Errorf("postgres.Add() = %v, want %v", got.Subscription, tt.wantSite.Subscription)
				}
				if string(tt.wantSite.Content) != string(got.Content) {
					t.Errorf("postgres.Add() = %v, want %v", got.Content, tt.wantSite.Content)
				}
				if tt.wantSite.LastUpdated != got.LastUpdated {
					t.Errorf("postgres.Add() = %v, want %v", got.LastUpdated, tt.wantSite.LastUpdated)
				}
			}
		})
	}
}
