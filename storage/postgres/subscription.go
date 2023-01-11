package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/storage"
	"golang.org/x/exp/slog"
)

type subscription struct {
	ID          int
	URL         string
	Filter      string
	ContentType string `db:"content_type"`
	UseChrome   bool   `db:"use_chrome"`
	Interval    DurationValuer
}

type subscriber struct {
	Email             string
	SubscriptionLimit int `db:"subscription_limit"`
}

func (db *db) FindBySubscriber(ctx context.Context, email string) ([]*htracker.Subscription, error) {
	subs := []*subscription{}

	query := `SELECT s.*, ss.interval FROM
	 subscriptions s,
	 (SELECT subscription_id, interval FROM subscriber_subscription WHERE subscriber_email = $1) ss
	 WHERE ss.subscription_id = s.id;`

	if err := db.conn.SelectContext(ctx, &subs, query, email); err != nil {
		db.logger.Error("query failed", err, slog.String("method", "FindBySubscriber"), slog.String("email", email))
		return []*htracker.Subscription{}, wrapError(err)
	}

	subscriptions := make([]*htracker.Subscription, len(subs))
	for i, s := range subs {
		subscriptions[i] = &htracker.Subscription{
			URL:         s.URL,
			Filter:      s.Filter,
			ContentType: s.ContentType,
			UseChrome:   s.UseChrome,
			Interval:    time.Duration(s.Interval),
		}
	}

	return subscriptions, nil
}

func (db *db) FindBySubscription(ctx context.Context, subscription *htracker.Subscription) ([]*storage.Subscriber, error) {
	subs := []*subscriber{}

	query := `SELECT * FROM subscribers WHERE email IN
		(SELECT subscriber_email FROM subscriber_subscription WHERE subscription_id IN
			(SELECT id FROM subscriptions WHERE url = $1 AND filter = $2 AND content_type = $3)
		)`

	if err := db.conn.SelectContext(ctx, &subs, query, subscription.URL, subscription.Filter, subscription.ContentType); err != nil {
		db.logger.Error("query failed", err, slog.String("method", "FindBySubscription"),
			slog.String("url", subscription.URL), slog.String("filter", subscription.Filter), slog.String("content_type", subscription.ContentType))
		return []*storage.Subscriber{}, wrapError(err)
	}

	subscribers := make([]*storage.Subscriber, len(subs))
	for i, s := range subs {
		// TODO: avoid this N+1 query if possible
		// TODO: Wrap this in a transaction
		subscriptions, err := db.FindBySubscriber(ctx, s.Email)
		if err != nil {
			return subscribers, err
		}
		subscribers[i] = &storage.Subscriber{
			Email:             s.Email,
			Subscriptions:     subscriptions,
			SubscriptionLimit: s.SubscriptionLimit,
		}
	}

	return subscribers, nil
}

func (db *db) SubscriberCount(ctx context.Context) (int, error) {
	var count int

	if err := db.conn.GetContext(ctx, &count, `SELECT count(email) FROM subscribers`); err != nil {
		db.logger.Error("query failed", err, slog.String("method", "SubscriberCount"))
		return count, wrapError(err)
	}

	return count, nil
}

func (db *db) AddSubscriber(ctx context.Context, subscriber *storage.Subscriber) error {
	if _, err := db.conn.ExecContext(ctx, `INSERT INTO subscribers(email, subscription_limit) VALUES ($1, $2)`,
		subscriber.Email, subscriber.SubscriptionLimit); err != nil {
		db.logger.Error("query failed", err, slog.String("method", "AddSubscriber"), slog.String("email", subscriber.Email))
		return wrapError(err)
	}
	return nil
}

func (db *db) GetAllSubscribers(ctx context.Context) ([]*storage.Subscriber, error) {
	subs := []*subscriber{}

	if err := db.conn.SelectContext(ctx, &subs, `SELECT * FROM subscribers`); err != nil {
		db.logger.Error("query failed", err, slog.String("method", "GetAllSubscribers"))
		return []*storage.Subscriber{}, wrapError(err)
	}

	subscribers := make([]*storage.Subscriber, len(subs))
	for i, s := range subs {
		// TODO: avoid this N+1 query if possible
		// TODO: Wrap this in a transaction
		subscriptions, err := db.FindBySubscriber(ctx, s.Email)
		if err != nil {
			return subscribers, err
		}
		subscribers[i] = &storage.Subscriber{
			Email:             s.Email,
			Subscriptions:     subscriptions,
			SubscriptionLimit: s.SubscriptionLimit,
		}
	}

	return subscribers, nil
}

func (db *db) GetSubscriber(ctx context.Context, email string) (*storage.Subscriber, error) {
	sub := subscriber{}

	err := db.conn.GetContext(ctx, &sub, `SELECT * FROM subscribers WHERE email = $1`, email)
	if err != nil {
		db.logger.Error("query failed", err, slog.String("method", "GetSubscriber"), slog.String("email", email))
		return &storage.Subscriber{}, wrapError(err)
	}

	// TODO: avoid this N+1 query if possible
	// TODO: Wrap this in a transaction
	subscriptions, err := db.FindBySubscriber(ctx, sub.Email)
	if err != nil {
		return &storage.Subscriber{}, err
	}

	return &storage.Subscriber{
		Email:             sub.Email,
		Subscriptions:     subscriptions,
		SubscriptionLimit: sub.SubscriptionLimit,
	}, nil
}

func (db *db) AddSubscription(ctx context.Context, email string, subscription *htracker.Subscription) error {

	logger := slog.New(db.logger.Handler().WithAttrs([]slog.Attr{
		slog.String("method", "AddSubscription"), slog.String("email", email), slog.String("url", subscription.URL),
		slog.String("filter", subscription.Filter), slog.String("content_type", subscription.ContentType)}))

	tx, err := db.conn.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		logger.Error("failed to begin a transaction", err)
		return err
	}

	query := `SELECT id FROM subscriptions WHERE url = $1 AND filter = $2 AND content_type = $3`

	var id int64

	// first try to find an existing subscription
	if err := tx.GetContext(ctx, &id, query, subscription.URL, subscription.Filter, subscription.ContentType); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logger.Error("query failed, rolling back transaction", err)
			if err := tx.Rollback(); err != nil {
				logger.Error("rollback failed", err)
			}
			return err
		} else {
			// we didn't find a subscription so we create one now
			query = `INSERT INTO subscriptions(url, filter, content_type, use_chrome)
				VALUES($1, $2, $3, $4) ON CONFLICT(url, content_type, filter) DO UPDATE
				SET url = $1, filter = $2, content_type = $3, use_chrome = $4
				RETURNING id`

			row := tx.QueryRowxContext(ctx, query, subscription.URL, subscription.Filter, subscription.ContentType, subscription.UseChrome)
			err := row.Scan(&id)
			if err != nil {
				logger.Error("query failed, rolling back transaction", err)
				if err := tx.Rollback(); err != nil {
					logger.Error("rollback failed", err)
				}
				return wrapError(err)
			}
		}
	}

	query = `INSERT INTO subscriber_subscription(subscriber_email, subscription_id, interval) VALUES($1, $2, $3)`

	_, err = tx.ExecContext(ctx, query, email, id, DurationValuer(subscription.Interval))
	if err != nil {
		logger.Error("query failed, rolling back transaction", err)
		if err := tx.Rollback(); err != nil {
			logger.Error("rollback failed", err)
		}
		return wrapError(err)
	}

	if err := tx.Commit(); err != nil {
		logger.Error("failed to commit the transaction", err)
		return err
	}

	return nil
}

func (db *db) RemoveSubscription(ctx context.Context, email string, subscription *htracker.Subscription) error {
	logger := slog.New(db.logger.Handler().WithAttrs([]slog.Attr{
		slog.String("method", "RemoveSubscription"), slog.String("email", email), slog.String("url", subscription.URL),
		slog.String("filter", subscription.Filter), slog.String("content_type", subscription.ContentType)}))

	tx, err := db.conn.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		logger.Error("failed to begin a transaction", err)
		return err
	}

	query := `DELETE FROM subscriber_subscription WHERE subscriber_email = $1 AND subscription_id IN
				(SELECT id FROM subscriptions WHERE url = $2 AND filter = $3 AND content_type =$4)`

	if _, err := tx.ExecContext(ctx, query, email, subscription.URL, subscription.Filter, subscription.ContentType); err != nil {
		logger.Error("query failed, rolling back transaction", err)
		if err := tx.Rollback(); err != nil {
			logger.Error("rollback failed", err)
		}
		return wrapError(err)
	}

	// cleanup non-referenced subscriptions
	query = `DELETE FROM subscriptions s
				WHERE NOT EXISTS (
					SELECT FROM subscriber_subscription ss
					WHERE s.id = ss.subscription_id
					)`

	if _, err := tx.ExecContext(ctx, query); err != nil {
		logger.Error("query failed, rolling back transaction", err)
		if err := tx.Rollback(); err != nil {
			logger.Error("rollback failed", err)
		}
		return wrapError(err)
	}

	if err := tx.Commit(); err != nil {
		logger.Error("failed to commit the transaction", err)
		return err
	}

	return nil
}

func (db *db) RemoveSubscriber(ctx context.Context, email string) error {
	query := `DELETE FROM subscribers WHERE email = $1`

	if _, err := db.conn.ExecContext(ctx, query, email); err != nil {
		db.logger.Error("query failed", err, slog.String("method", "RemoveSubscriber"), slog.String("email", email))
		return wrapError(err)
	}

	return nil
}
