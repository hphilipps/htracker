package postgres

import (
	"context"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

func Test_db_AddSubscriber(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	sub1 := &storage.Subscriber{Email: "email1", SubscriptionLimit: 0}
	sub2 := &storage.Subscriber{Email: "email2", SubscriptionLimit: 10}

	tests := []struct {
		name       string
		subscriber *storage.Subscriber
		wantErr    bool
	}{
		{name: "add subscriber1", subscriber: sub1},
		{name: "add subscriber1 again", subscriber: sub1, wantErr: true},
		{name: "add subscriber2", subscriber: sub2},
	}

	ctx := context.Background()
	logger := slog.Default()
	db, err := New("postgresql://postgres:pg1pw@localhost?sslmode=disable", logger)
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.AddSubscriber(ctx, tt.subscriber); (err != nil) != tt.wantErr {
				t.Errorf("db.AddSubscriber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotSub, err := db.GetSubscriber(ctx, tt.subscriber.Email)
				if err != nil {
					t.Errorf("Failed to Get subscriber after Add: %v", err)
					return
				}
				if got, want := gotSub.Email, tt.subscriber.Email; got != want {
					t.Errorf("postgres.AddSubscriber() email = %v, want %v", got, want)
				}
				if got, want := gotSub.SubscriptionLimit, tt.subscriber.SubscriptionLimit; got != want {
					t.Errorf("postgres.AddSubscriber() SubscriptionLimit = %v, want %v", got, want)
				}
			}
		})
	}
}

func Test_db_AddSubscription(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", intTestVarName)
	}

	subscriber1 := &storage.Subscriber{Email: "addsubemail1", SubscriptionLimit: 0}
	subscriber2 := &storage.Subscriber{Email: "addsubemail2", SubscriptionLimit: 10}

	subscription1 := &htracker.Subscription{URL: "addsubsite1", Filter: "filter1", ContentType: "text",
		UseChrome: true, Interval: 1234*time.Hour + 6*time.Minute + 11*time.Second}
	subscription2 := &htracker.Subscription{URL: "addsubsite2", Interval: 30 * time.Minute}

	tests := []struct {
		name              string
		email             string
		subscription      *htracker.Subscription
		wantSubscriptions []*htracker.Subscription
		wantErr           bool
	}{
		{name: "add subscriber1 subscription1", email: subscriber1.Email, subscription: subscription1,
			wantSubscriptions: []*htracker.Subscription{subscription1}},
		{name: "add subscriber1 subscription2", email: subscriber1.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{subscription1, subscription2}},
		{name: "add subscription2 again", email: subscriber1.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{subscription1, subscription2}, wantErr: true},
		{name: "add subscriber2 subscription2", email: subscriber2.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{subscription2}},
		{name: "add subscription to non-existent subscriber", email: "non-existing", subscription: subscription2,
			wantErr: true},
	}

	ctx := context.Background()
	logger := slog.Default()
	db, err := New("postgresql://postgres:pg1pw@localhost?sslmode=disable", logger)
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}

	if err := db.AddSubscriber(ctx, subscriber1); err != nil {
		t.Errorf("Setup: failed to add subscriber")
		return
	}
	if err := db.AddSubscriber(ctx, subscriber2); err != nil {
		t.Errorf("Setup: failed to add subscriber")
		return
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.AddSubscription(ctx, tt.email, tt.subscription); (err != nil) != tt.wantErr {
				t.Errorf("db.AddSubscription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotSubs, err := db.FindBySubscriber(ctx, tt.email)
				if err != nil {
					t.Errorf("Failed to Find subscriptions after Add: %v", err)
					return
				}
				if want, got := len(tt.wantSubscriptions), len(gotSubs); want != got {
					t.Errorf("Expected %d subscriptions, got %d subscription", want, got)
					return
				}
				for i, s := range tt.wantSubscriptions {
					if want, got := s.URL, gotSubs[i].URL; want != got {
						t.Errorf("Expected URL %s, got %s", want, got)
					}
					if want, got := s.Filter, gotSubs[i].Filter; want != got {
						t.Errorf("Expected Filter %s, got %s", want, got)
					}
					if want, got := s.ContentType, gotSubs[i].ContentType; want != got {
						t.Errorf("Expected ContentType %s, got %s", want, got)
					}
					if want, got := s.Interval, gotSubs[i].Interval; want != got {
						t.Errorf("Expected Interval %v, got %v", want, got)
					}
				}
			}
		})
	}
}
