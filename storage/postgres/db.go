package postgres

import (
	"github.com/jmoiron/sqlx"
	"golang.org/x/exp/slog"

	_ "github.com/lib/pq"
)

type db struct {
	conn   *sqlx.DB
	logger *slog.Logger
}

func New(uri string, logger *slog.Logger) (db, error) {
	var db db
	conn, err := sqlx.Open("postgres", uri)
	if err != nil {
		return db, err
	}
	if err := conn.Ping(); err != nil {
		return db, err
	}
	db.conn = conn
	db.logger = logger.With(slog.String("driver", "postgresql"))
	return db, nil
}
