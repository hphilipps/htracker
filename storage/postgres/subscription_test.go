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
		t.Skipf("set %s env var to run this test", integrationTestVar)
	}

	sub1 := &storage.Subscriber{Email: "email1", SubscriptionLimit: 0}
	sub2 := &storage.Subscriber{Email: "email2", SubscriptionLimit: 10}

	tests := []struct {
		name       string
		subscriber *storage.Subscriber
		wantCount  int
		wantErr    bool
	}{
		{name: "add subscriber1", subscriber: sub1, wantCount: 1},
		{name: "add subscriber1 again", subscriber: sub1, wantCount: 1, wantErr: true},
		{name: "add subscriber2", subscriber: sub2, wantCount: 2},
	}

	ctx := context.Background()
	logger := slog.Default()
	db, err := New(postgresURIfromEnvVars(), logger)
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
				gotCount, err := db.SubscriberCount(ctx)
				if err != nil {
					t.Errorf("Failed to Get subscriber count after Add: %v", err)
					return
				}
				if got, want := gotCount, tt.wantCount; got != want {
					t.Errorf("postgres.AddSubscriber() count = %d, want %d", got, want)
				}
				gotAll, err := db.GetAllSubscribers(ctx)
				if err != nil {
					t.Errorf("Failed to Get all subscribers after Add: %v", err)
					return
				}
				if got, want := len(gotAll), tt.wantCount; got != want {
					t.Errorf("postgres.AddSubscriber() got %d, want %d", got, want)
				}
			}
		})
	}
}

func Test_db_AddSubscription_and_Find(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", integrationTestVar)
	}

	subscriber1 := &storage.Subscriber{Email: "addsubemail1", SubscriptionLimit: 0}
	subscriber2 := &storage.Subscriber{Email: "addsubemail2", SubscriptionLimit: 10}

	subscription1 := &htracker.Subscription{URL: "addsubsite1", Filter: "filter1", ContentType: "text",
		UseChrome: true, Interval: 12340*time.Hour + 6*time.Minute + 11*time.Second}
	subscription2 := &htracker.Subscription{URL: "addsubsite2", Interval: 30 * time.Minute}

	tests := []struct {
		name              string
		email             string
		subscription      *htracker.Subscription
		wantSubscriptions []*htracker.Subscription
		wantSubscribers   []*storage.Subscriber
		wantErr           bool
	}{
		{name: "add subscriber1 subscription1", email: subscriber1.Email, subscription: subscription1,
			wantSubscriptions: []*htracker.Subscription{subscription1}, wantSubscribers: []*storage.Subscriber{subscriber1}},
		{name: "add subscriber1 subscription2", email: subscriber1.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{subscription1, subscription2}, wantSubscribers: []*storage.Subscriber{subscriber1}},
		{name: "add subscription2 again", email: subscriber1.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{subscription1, subscription2}, wantSubscribers: []*storage.Subscriber{subscriber1}, wantErr: true},
		{name: "add subscriber2 subscription2", email: subscriber2.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{subscription2}, wantSubscribers: []*storage.Subscriber{subscriber1, subscriber2}},
		{name: "add subscription to non-existent subscriber", email: "non-existing", subscription: subscription2,
			wantErr: true},
	}

	ctx := context.Background()
	logger := slog.Default()
	db, err := New(postgresURIfromEnvVars(), logger)
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
				gotSubscribers, err := db.FindBySubscription(ctx, tt.subscription)
				if err != nil {
					t.Errorf("Failed to Find subscribers after Add: %v", err)
					return
				}
				if want, got := len(tt.wantSubscribers), len(gotSubscribers); want != got {
					t.Errorf("Expected %d subscribers, got %d subscribers", want, got)
					return
				}
				for i, s := range tt.wantSubscribers {
					if want, got := s.Email, gotSubscribers[i].Email; want != got {
						t.Errorf("Expected email %s, got %s", want, got)
					}
				}
			}
		})
	}
}

func Test_db_RemoveSubscription(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", integrationTestVar)
	}

	subscriber1 := &storage.Subscriber{Email: "rmsubemail1", SubscriptionLimit: 0}
	subscriber2 := &storage.Subscriber{Email: "rmsubemail2", SubscriptionLimit: 10}

	subscription1 := &htracker.Subscription{URL: "rmsubsite1", Filter: "filter1", ContentType: "text",
		UseChrome: true, Interval: 1234*time.Hour + 6*time.Minute + 11*time.Second}
	subscription2 := &htracker.Subscription{URL: "rmsubsite2", Interval: 30 * time.Minute}

	tests := []struct {
		name              string
		email             string
		subscription      *htracker.Subscription
		wantSubscriptions []*htracker.Subscription
		wantErr           bool
	}{
		{name: "remove subscription1 from subscriber1", email: subscriber1.Email, subscription: subscription1,
			wantSubscriptions: []*htracker.Subscription{subscription2}},
		{name: "remove subscription1 from subscriber1 again", email: subscriber1.Email, subscription: subscription1,
			wantSubscriptions: []*htracker.Subscription{subscription2}, wantErr: true},
		{name: "remove subscription2 from subscriber1", email: subscriber1.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{}},
		{name: "remove subscription2 from subscriber2", email: subscriber2.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{}},
		{name: "remove subscription2 from subscriber2 again", email: subscriber2.Email, subscription: subscription2,
			wantSubscriptions: []*htracker.Subscription{}, wantErr: true},
	}

	ctx := context.Background()
	logger := slog.Default()
	db, err := New(postgresURIfromEnvVars(), logger)
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}

	// Setup
	if err := db.AddSubscriber(ctx, subscriber1); err != nil {
		t.Errorf("Setup: failed to add subscriber")
		return
	}
	if err := db.AddSubscriber(ctx, subscriber2); err != nil {
		t.Errorf("Setup: failed to add subscriber")
		return
	}
	if err := db.AddSubscription(ctx, subscriber1.Email, subscription1); err != nil {
		t.Errorf("Setup: failed to add subscription")
		return
	}
	if err := db.AddSubscription(ctx, subscriber1.Email, subscription2); err != nil {
		t.Errorf("Setup: failed to add subscription")
		return
	}
	if err := db.AddSubscription(ctx, subscriber2.Email, subscription2); err != nil {
		t.Errorf("Setup: failed to add subscription")
		return
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.RemoveSubscription(ctx, tt.email, tt.subscription); (err != nil) != tt.wantErr {
				t.Errorf("db.RemoveSubscription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotSubs, err := db.FindBySubscriber(ctx, tt.email)
			if err != nil {
				t.Errorf("Failed to Find subscriptions after Remove: %v", err)
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
			}
		})
	}
}

func Test_db_RemoveSubscriber(t *testing.T) {
	if !runIntegrationTests() {
		t.Skipf("set %s env var to run this test", integrationTestVar)
	}

	subscriber1 := &storage.Subscriber{Email: "rmsubsbemail1", SubscriptionLimit: 0}
	subscriber2 := &storage.Subscriber{Email: "rmsbubsbemail2", SubscriptionLimit: 10}

	subscription1 := &htracker.Subscription{URL: "rmsubsbsite1", Filter: "filter1", ContentType: "text",
		UseChrome: true, Interval: 1234*time.Hour + 6*time.Minute + 11*time.Second}
	subscription2 := &htracker.Subscription{URL: "rmsubsbsite2", Interval: 30 * time.Minute}

	tests := []struct {
		name            string
		email           string
		wantSubscribers []*storage.Subscriber
		wantErr         bool
	}{
		{name: "remove subscriber1", email: subscriber1.Email,
			wantSubscribers: []*storage.Subscriber{subscriber2}},
		{name: "remove subscriber1 again", email: subscriber1.Email,
			wantSubscribers: []*storage.Subscriber{subscriber2}, wantErr: true},
		{name: "remove subscriber2", email: subscriber2.Email,
			wantSubscribers: []*storage.Subscriber{}},
		{name: "remove subscriber2 again", email: subscriber2.Email,
			wantSubscribers: []*storage.Subscriber{}, wantErr: true},
	}

	ctx := context.Background()
	logger := slog.Default()
	db, err := New(postgresURIfromEnvVars(), logger)
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}

	// Setup
	if err := db.AddSubscriber(ctx, subscriber1); err != nil {
		t.Errorf("Setup: failed to add subscriber")
		return
	}
	if err := db.AddSubscriber(ctx, subscriber2); err != nil {
		t.Errorf("Setup: failed to add subscriber")
		return
	}
	if err := db.AddSubscription(ctx, subscriber1.Email, subscription1); err != nil {
		t.Errorf("Setup: failed to add subscription")
		return
	}
	if err := db.AddSubscription(ctx, subscriber1.Email, subscription2); err != nil {
		t.Errorf("Setup: failed to add subscription")
		return
	}
	if err := db.AddSubscription(ctx, subscriber2.Email, subscription2); err != nil {
		t.Errorf("Setup: failed to add subscription")
		return
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.RemoveSubscriber(ctx, tt.email); (err != nil) != tt.wantErr {
				t.Errorf("db.RemoveSubscriber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotSubs, err := db.FindBySubscription(ctx, subscription2)
			if err != nil {
				t.Errorf("Failed to Find subscribers after Remove: %v", err)
				return
			}
			if want, got := len(tt.wantSubscribers), len(gotSubs); want != got {
				t.Errorf("Expected %d subscribers, got %d", want, got)
				return
			}
			for i, s := range tt.wantSubscribers {
				if want, got := s.Email, gotSubs[i].Email; want != got {
					t.Errorf("Expected email %s, got %s", want, got)
				}
			}
		})
	}
}
