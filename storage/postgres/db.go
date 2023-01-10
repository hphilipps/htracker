package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"gitlab.com/henri.philipps/htracker"
	"golang.org/x/exp/slog"
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

func wrapError(err error) error {
	switch e := err.(type) {
	case *pq.Error:
		if e.Code == "23505" {
			return fmt.Errorf("%w: %v", htracker.ErrAlreadyExists, err)
		}
	default:
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %v", htracker.ErrNotExist, err)
		}
	}
	return err
}
