package postgres

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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

type DurationValuer time.Duration

func (dv *DurationValuer) Scan(src interface{}) error {
	if src == nil {
		*dv = 0
		return nil
	}

	switch srcStr := src.(type) {
	case string:
		return unmarshalForIntervalStyle(srcStr, "postgres", dv)
	case []byte:
		return unmarshalForIntervalStyle(string(srcStr), "postgres", dv)
	}

	return fmt.Errorf("duration column was not text; type %T", src)
}

func (dv DurationValuer) Value() (driver.Value, error) {
	return marshalForIntervalStyle(time.Duration(dv), "postgres"), nil
}

func marshalForIntervalStyle(duration time.Duration, style string) string {
	switch style {
	case "postgres":
		seconds := (duration / time.Second) % 60
		minutes := (duration / time.Minute) % 60
		hours := duration / time.Hour

		str := strings.Builder{}
		str.WriteString(fmt.Sprintf("%02d", hours))
		str.WriteString(":")
		str.WriteString(fmt.Sprintf("%02d", minutes))
		str.WriteString(":")
		str.WriteString(fmt.Sprintf("%02d", seconds))
		return str.String()
	}
	return "only postgres style supported for intervals"
}

// See: https://www.postgresql.org/docs/14/datatype-datetime.html
// IntervalStyle Output Table
func unmarshalForIntervalStyle(src, style string, dv *DurationValuer) error {

	if style != "postgres" {
		return fmt.Errorf("only postgres style duration format supported")
	}

	value := 0
	hours := 0
	parts := strings.Split(src, " ")

	i := 0
	for {
		if i >= len(parts) {
			// we reached the end and return the hours we parsed so far
			duration, err := time.ParseDuration(strconv.Itoa(hours) + "h")
			if err != nil {
				return fmt.Errorf("could not parse interval string")
			}
			*dv = DurationValuer(duration)
			return nil
		}
		n, err := strconv.Atoi(strings.TrimLeft(parts[i], "0"))
		if err != nil {
			if i > 0 {
				// we already parsed values before and need to add up hours
				switch {
				case strings.Contains(parts[i], "year"):
					hours = hours + value*24*365
				case strings.Contains(parts[i], "mon"):
					hours = hours + value*24*30
				case strings.Contains(parts[i], "day"):
					hours = hours + value*24
				case strings.Contains(parts[i], ":"):
					duration, err := parseHMS(parts[i])
					if err != nil {
						return err
					}
					duration = duration + time.Duration(hours)*time.Hour
					*dv = DurationValuer(duration)
					return nil
				default:
					return fmt.Errorf("could not parse interval string")
				}
			} else {
				// if we find h:m:s at the beginning
				duration, err := parseHMS(parts[i])
				if err != nil {
					return err
				}
				*dv = DurationValuer(duration)
				return nil
			}
		} else {
			// we found a number and memorize it for the next step
			value = n
		}
		i++
	}
}

func parseHMS(str string) (time.Duration, error) {
	hms := strings.Split(str, ":")
	if len(hms) != 3 {
		return time.Duration(0), fmt.Errorf("could not parse h:m:s")
	}
	dstr := hms[0] + "h" + hms[1] + "m" + hms[2] + "s"
	duration, err := time.ParseDuration(dstr)
	if err != nil {
		return time.Duration(0), fmt.Errorf("could not parse interval string %s: %v", dstr, err)
	}
	return duration, nil
}
