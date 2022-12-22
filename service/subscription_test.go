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
		email        string
		subscription *htracker.Subscription
	}

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Minute}
	sub4 := &htracker.Subscription{URL: "http://site4.example/blah", Filter: "foo", ContentType: "text", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"
	email4 := "toomuch@foo.test"

	subscriberLimit := 3
	subscriptionLimit := 2

	tests := []struct {
		name              string
		args              args
		wantSubscriptions []*htracker.Subscription
		wantErr           bool
	}{
		{name: "subscribe email1 to site1",
			args: args{email: email1, subscription: sub1}, wantSubscriptions: []*htracker.Subscription{sub1}, wantErr: false},
		{name: "subscribe email2 to site1",
			args: args{email: email2, subscription: sub1}, wantSubscriptions: []*htracker.Subscription{sub1}, wantErr: false},
		{name: "subscribe email3 to site1",
			args: args{email: email3, subscription: sub1}, wantSubscriptions: []*htracker.Subscription{sub1}, wantErr: false},
		{name: "subscribe email1 to site2",
			args: args{email: email1, subscription: sub2}, wantSubscriptions: []*htracker.Subscription{sub1, sub2}, wantErr: false},
		{name: "subscribe email2 to site2",
			args: args{email: email2, subscription: sub2}, wantSubscriptions: []*htracker.Subscription{sub1, sub2}, wantErr: false},
		{name: "subscribe email1 to same site again",
			args: args{email: email1, subscription: sub1}, wantSubscriptions: []*htracker.Subscription{sub1, sub2}, wantErr: true},
		{name: "subscribe email1 to equal site again",
			args: args{email: email1, subscription: sub3}, wantSubscriptions: []*htracker.Subscription{sub1, sub2}, wantErr: true},
		{name: "go over subscription limit",
			args: args{email: email1, subscription: sub4}, wantSubscriptions: []*htracker.Subscription{sub1, sub2}, wantErr: true},
		{name: "subscribe with unknown subscriber",
			args: args{email: email4, subscription: sub4}, wantSubscriptions: nil, wantErr: true},
	}

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage, WithSubscriberLimit(subscriberLimit), WithSubscriptionLimit(subscriptionLimit))
	svc.AddSubscriber(&Subscriber{Email: email1, SubscriptionLimit: subscriptionLimit})
	svc.AddSubscriber(&Subscriber{Email: email2})
	svc.AddSubscriber(&Subscriber{Email: email3})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subErr := svc.Subscribe(tt.args.email, tt.args.subscription)
			if (subErr != nil) != tt.wantErr {
				t.Errorf("svc.Subscribe() error = %v, wantErr %v", subErr, tt.wantErr)
			}

			if subErr == nil {
				gotSubscriptions, err := svc.GetSubscriptionsBySubscriber(tt.args.email)
				if err != nil {
					t.Errorf("svc.Subscribe() - validation with svc.GetSubscriptionsBySubscriber() failed: %v", err)
				}

				if len(tt.wantSubscriptions) != len(gotSubscriptions) {
					t.Errorf("Expected %d subscriptions for %s, got %d", len(tt.wantSubscriptions), tt.args.email, len(gotSubscriptions))
				}
				for _, i := range tt.wantSubscriptions {
					found := false
					for _, j := range gotSubscriptions {
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
			}
		})
	}
}

func TestSubscriptionSvc_Unsubscribe(t *testing.T) {
	type args struct {
		email        string
		subscription *htracker.Subscription
	}

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	svc.AddSubscriber(&Subscriber{Email: email1})
	svc.AddSubscriber(&Subscriber{Email: email2})
	svc.AddSubscriber(&Subscriber{Email: email3})
	svc.Subscribe(email1, sub1)
	svc.Subscribe(email1, sub2)
	svc.Subscribe(email1, sub3)
	svc.Subscribe(email2, sub1)
	svc.Subscribe(email2, sub2)
	svc.Subscribe(email3, sub3)
	svc.Unsubscribe(email3, sub3) // should leave email3 with 0 subscriptions

	tests := []struct {
		name              string
		args              args
		wantSubscriptions []*htracker.Subscription
		wantErr           bool
		wantEmail         bool
	}{
		{name: "unsubscribe email1 from sub1",
			args: args{email: email1, subscription: sub1}, wantSubscriptions: []*htracker.Subscription{sub2, sub3}, wantErr: false, wantEmail: true},
		{name: "unsubscribe email2 from sub1",
			args: args{email: email2, subscription: sub1}, wantSubscriptions: []*htracker.Subscription{sub2}, wantErr: false, wantEmail: true},
		{name: "unsubscribe email3 from not subscribed sub1",
			args: args{email: email3, subscription: sub1}, wantSubscriptions: []*htracker.Subscription{}, wantErr: true, wantEmail: true},
		{name: "unsubscribe email1 from sub2",
			args: args{email: email1, subscription: sub2}, wantSubscriptions: []*htracker.Subscription{sub3}, wantErr: false, wantEmail: true},
		{name: "unsubscribe email2 from sub2",
			args: args{email: email2, subscription: sub2}, wantSubscriptions: []*htracker.Subscription{}, wantErr: false, wantEmail: true},
		{name: "unsubscribe email1 from sub2 again",
			args: args{email: email1, subscription: sub2}, wantSubscriptions: []*htracker.Subscription{sub3}, wantErr: true, wantEmail: true},
		{name: "unsubscribe email3 from not subscribed sub3",
			args: args{email: email3, subscription: sub3}, wantSubscriptions: []*htracker.Subscription{}, wantErr: true, wantEmail: true},
		{name: "unsubscribe nonexistent subscriber from sub1",
			args: args{email: "nothing@foo.test", subscription: sub1}, wantSubscriptions: []*htracker.Subscription{}, wantErr: true, wantEmail: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.Unsubscribe(tt.args.email, tt.args.subscription); (err != nil) != tt.wantErr {
				t.Errorf("svc.Unsubscribe() error = %v, wantErr %v", err, tt.wantErr)
			}

			gotSubscriptions, err := svc.GetSubscriptionsBySubscriber(tt.args.email)
			if err != nil && tt.wantEmail == true {
				t.Errorf("svc.Unsubscribe() - validation with svc.GetSubscriptionsBySubscriber() failed: %v", err)
			}

			if len(tt.wantSubscriptions) != len(gotSubscriptions) {
				t.Errorf("Expected %d subscriptions for %s, got %d", len(tt.wantSubscriptions), tt.args.email, len(gotSubscriptions))
			}
			for _, i := range tt.wantSubscriptions {
				found := false
				for _, j := range gotSubscriptions {
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

func TestSubscriptionSvc_GetSubscriptionsBySubscriber(t *testing.T) {
	type args struct {
		email string
	}

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	svc.AddSubscriber(&Subscriber{Email: email1})
	svc.AddSubscriber(&Subscriber{Email: email2})
	svc.AddSubscriber(&Subscriber{Email: email3})
	svc.Subscribe(email1, sub1)
	svc.Subscribe(email1, sub2)
	svc.Subscribe(email1, sub3)
	svc.Subscribe(email2, sub1)
	svc.Subscribe(email2, sub2)
	svc.Subscribe(email3, sub3)
	svc.Unsubscribe(email3, sub3) // should leave email3 with 0 subscriptions

	tests := []struct {
		name              string
		args              args
		wantSubscriptions []*htracker.Subscription
		wantErr           bool
	}{
		{name: "get email1 subscriptions",
			args: args{email: email1}, wantSubscriptions: []*htracker.Subscription{sub1, sub2, sub3}, wantErr: false},
		{name: "get email2 subscriptions",
			args: args{email: email2}, wantSubscriptions: []*htracker.Subscription{sub1, sub2}, wantErr: false},
		{name: "get email3 subscriptions",
			args: args{email: email3}, wantSubscriptions: []*htracker.Subscription{}, wantErr: false},
		{name: "get subscriptions of nonexistent email",
			args: args{email: "nonexisting@foo.test"}, wantSubscriptions: []*htracker.Subscription{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSubscriptions, err := svc.GetSubscriptionsBySubscriber(tt.args.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("svc.GetSubscriptionsBySubscriber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(gotSubscriptions, tt.wantSubscriptions) {
					t.Errorf("svc.GetSubscriptionsBySubscriber() = %v, want %v", gotSubscriptions, tt.wantSubscriptions)
				}
			}
		})
	}
}

func TestSubscriptionSvc_GetSubscribersBySubscription(t *testing.T) {
	type args struct {
		subscription *htracker.Subscription
	}

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	svc.AddSubscriber(&Subscriber{Email: email1})
	svc.AddSubscriber(&Subscriber{Email: email2})
	svc.AddSubscriber(&Subscriber{Email: email3})
	svc.Subscribe(email1, sub1)
	svc.Subscribe(email1, sub2)
	svc.Subscribe(email1, sub3)
	svc.Subscribe(email2, sub1)
	svc.Subscribe(email2, sub2)
	svc.Subscribe(email3, sub3)
	svc.Unsubscribe(email3, sub3) // should leave email3 with 0 subscriptions

	tests := []struct {
		name       string
		args       args
		wantEmails []string
		wantErr    bool
	}{
		{name: "get sub1 subscribers", args: args{subscription: sub1}, wantEmails: []string{email1, email2}, wantErr: false},
		{name: "get sub2 subscribers", args: args{subscription: sub2}, wantEmails: []string{email1, email2}, wantErr: false},
		{name: "get sub3 subscribers", args: args{subscription: sub3}, wantEmails: []string{email1}, wantErr: false},
		{name: "get subscribers to nonexistent subscription",
			args: args{subscription: &htracker.Subscription{URL: "nowhere.test/foo"}}, wantEmails: []string{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			subscribers, err := svc.GetSubscribersBySubscription(tt.args.subscription)
			if (err != nil) != tt.wantErr {
				t.Errorf("svc.GetSubscribersBySubscription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !(len(tt.wantEmails) == 0 && len(subscribers) == 0) {
				gotEmails := []string{}
				for _, s := range subscribers {
					gotEmails = append(gotEmails, s.Email)
				}
				if !reflect.DeepEqual(gotEmails, tt.wantEmails) {
					t.Errorf("svc.GetSubscribersBySubscription() = %v, want %v", gotEmails, tt.wantEmails)
				}
			}
		})
	}
}

func TestSubscriptionSvc_GetSubscribers(t *testing.T) {

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site2.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "byte", Interval: time.Minute}

	email1 := "email1@foo.test"
	email2 := "email2@foo.test"
	email3 := "email3@foo.test"

	logger := slog.Default()
	storage := memory.NewSubscriptionStorage(logger)
	svc := NewSubscriptionSvc(storage)

	svc.AddSubscriber(&Subscriber{Email: email1})
	svc.AddSubscriber(&Subscriber{Email: email2})
	svc.AddSubscriber(&Subscriber{Email: email3})
	svc.Subscribe(email1, sub1)
	svc.Subscribe(email1, sub2)
	svc.Subscribe(email1, sub3)
	svc.Subscribe(email2, sub1)
	svc.Subscribe(email2, sub2)
	svc.Subscribe(email3, sub3)
	svc.Unsubscribe(email3, sub3) // should leave email3 with 0 subscriptions

	subscribers, err := svc.GetSubscribers()
	if err != nil {
		t.Errorf("svc.GetSubscribers() failed: %v", err)
	}
	gotEmails := []string{}
	for _, sub := range subscribers {
		gotEmails = append(gotEmails, sub.Email)
	}
	if !reflect.DeepEqual(gotEmails, []string{email1, email2, email3}) {
		t.Errorf("svc.GetSubscribers() expected %v, got %v", []string{email1, email2, email3}, gotEmails)
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

	svc.AddSubscriber(&Subscriber{Email: email1})
	err := svc.Subscribe(email1, &htracker.Subscription{URL: "some.web.site.test/blah", Filter: "someFilter", ContentType: "text"})
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
				t.Errorf("svc.DeleteSubscriber() error = %v, wantErr %v", err, tt.wantErr)
			}
			subscribers, err := svc.GetSubscribers()
			if err != nil {
				t.Errorf("svc.GetSubscribers() failed: %v", err)
			}
			found := false
			for _, sub := range subscribers {
				if (sub.Email == email1) == tt.wantExist {
					found = true
					break
				}
			}
			if !found && tt.wantExist {
				t.Errorf("svc.DeleteSubscriber() expected entry to still exist but it is gone")
			}
		})
	}
}

func Test_subscriptionSvc_AddSubscriber(t *testing.T) {
	type args struct {
		subscriber *Subscriber
	}

	sub1 := &Subscriber{Email: "email1"}
	sub2 := &Subscriber{Email: "email2"}
	sub3 := &Subscriber{Email: "email3"}

	tests := []struct {
		name            string
		args            args
		wantSubscribers []*Subscriber
		wantErr         bool
	}{
		{name: "add subscriber1", args: args{subscriber: sub1}, wantSubscribers: []*Subscriber{sub1}, wantErr: false},
		{name: "add subscriber2", args: args{subscriber: sub2}, wantSubscribers: []*Subscriber{sub1, sub2}, wantErr: false},
		{name: "go over limit", args: args{subscriber: sub3}, wantSubscribers: []*Subscriber{sub1, sub2}, wantErr: true},
		{name: "add existing", args: args{subscriber: sub1}, wantSubscribers: []*Subscriber{sub1, sub2}, wantErr: true},
	}

	svc := &subscriptionSvc{
		storage:         memory.NewSubscriptionStorage(slog.Default()),
		subscriberLimit: 2,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.AddSubscriber(tt.args.subscriber); (err != nil) != tt.wantErr {
				t.Errorf("subscriptionSvc.AddSubscriber() error = %v, wantErr %v", err, tt.wantErr)
			}
			gotSubscribers, err := svc.GetSubscribers()
			if err != nil {
				t.Errorf("GetSubscribers() failes: %v", err)
			}
			if !reflect.DeepEqual(gotSubscribers, tt.wantSubscribers) {
				t.Errorf("Expected subscribers %v, got %v", tt.wantSubscribers, gotSubscribers)
			}
		})
	}
}
