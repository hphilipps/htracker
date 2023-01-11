package postgres

import (
	"context"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"golang.org/x/exp/slog"
)

type site struct {
	URL         string
	Filter      string
	ContentType string    `db:"content_type"`
	LastUpdated time.Time `db:"last_updated"`
	LastChecked time.Time `db:"last_checked"`
	Content     []byte
	Diff        string
	Checksum    string
}

func (db *db) Get(ctx context.Context, subscription *htracker.Subscription) (*htracker.Site, error) {
	site := &site{}

	if err := db.conn.GetContext(ctx, site, "SELECT * FROM sites WHERE url=$1 AND filter=$2 AND content_type=$3",
		subscription.URL, subscription.Filter, subscription.ContentType); err != nil {
		db.logger.Error("query failed", err, slog.String("method", "Get"), slog.String("url", subscription.URL),
			slog.String("filter", subscription.Filter), slog.String("content_type", subscription.ContentType))
		return &htracker.Site{}, wrapError(err)
	}

	return &htracker.Site{
		Subscription: &htracker.Subscription{URL: site.URL, Filter: site.Filter, ContentType: site.ContentType},
		LastUpdated:  site.LastUpdated,
		LastChecked:  site.LastChecked,
		Content:      site.Content,
		Diff:         site.Diff,
		Checksum:     site.Checksum,
	}, nil
}

func (db *db) Add(ctx context.Context, s *htracker.Site) error {

	query := `
	INSERT INTO sites
	(url, filter, content_type, last_updated, last_checked, content, diff, checksum)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.conn.ExecContext(ctx, query, s.Subscription.URL, s.Subscription.Filter,
		s.Subscription.ContentType, s.LastUpdated, s.LastChecked, s.Content, s.Diff, s.Checksum)
	if err != nil {
		db.logger.Error("query failed", err, slog.String("method", "Add"), slog.String("url", s.Subscription.URL),
			slog.String("filter", s.Subscription.Filter), slog.String("content_type", s.Subscription.ContentType))
		return wrapError(err)
	}
	return nil
}

func (db *db) Update(ctx context.Context, s *htracker.Site) error {

	query := `
	UPDATE sites SET
	last_updated = $1, last_checked = $2, content = $3, diff = $4, checksum = $5
	WHERE url = $6 AND filter = $7 AND content_type = $8`

	res, err := db.conn.ExecContext(ctx, query, s.LastUpdated, s.LastChecked,
		s.Content, s.Diff, s.Checksum, s.Subscription.URL, s.Subscription.Filter, s.Subscription.ContentType)
	if err != nil {
		db.logger.Error("query failed", err, slog.String("method", "Update"), slog.String("url", s.Subscription.URL),
			slog.String("filter", s.Subscription.Filter), slog.String("content_type", s.Subscription.ContentType))
		return wrapError(err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return htracker.ErrNotExist
	}

	return nil
}
