package postgres

import (
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/lib/pq"
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

func Test_DurationValuer_Scan(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		duration time.Duration
		wantErr  bool
	}{
		{name: "1 year", src: "1 year", duration: 24 * 365 * time.Hour},
		{name: "1 month", src: "1 month", duration: 24 * 30 * time.Hour},
		{name: "1 day", src: "1 day", duration: 24 * time.Hour},
		{name: "3:04:11", src: "3:04:11", duration: 3*time.Hour + 4*time.Minute + 11*time.Second},
		{name: "0:00:11", src: "0:0:11", duration: 11 * time.Second},
		{name: "1 year 0:00:11", src: "1 year 0:00:11", duration: 24*365*time.Hour + 11*time.Second},
		{name: "3:04.5:11.5", src: "3:04.5:11.5", duration: 3*time.Hour + 4*time.Minute + 41*time.Second + 500*time.Millisecond},
		{name: "1 year 2 months", src: "1 year 2 months", duration: 24*365*time.Hour + 2*24*30*time.Hour},
		{name: "2 years 2 months 2 days", src: "2 years 2 months 2 days", duration: 2*24*365*time.Hour + 2*24*30*time.Hour + 2*24*time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := DurationValuer(0)
			if err := dv.Scan(tt.src); (err != nil) != tt.wantErr {
				t.Errorf("DurationValues.Scan error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if want, got := tt.duration, time.Duration(dv); want != got {
					t.Errorf("Want duration %v, got %v", want, got)
				}
			}
		})
	}
}
