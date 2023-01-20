package postgres

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"gitlab.com/henri.philipps/htracker"
	"golang.org/x/exp/slog"
)

type db struct {
	conn   *sqlx.DB
	logger *slog.Logger
}

func New(uri string, logger *slog.Logger) (*db, error) {
	db := &db{}
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

// wrapError is translating some postgres errors into domain errors.
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

// DurationValuer is a wrapper for time.Duration, implementing
// Scan() and Value(), to be able to convert from and to postgres
// intervals. It is just covering basic use cases (hh:mm:ss)! No
// testing for corner cases (nanosecond precision, intervals > 292 years,
// the meaning of a literal month, negative intervals etc).
// See: https://www.postgresql.org/docs/14/datatype-datetime.html
// IntervalStyle Output Table.
type DurationValuer time.Duration

// Scan implements Scanner.
func (dv *DurationValuer) Scan(src interface{}) error {
	if src == nil {
		*dv = 0
		return nil
	}

	switch srcStr := src.(type) {
	case []byte:
		return unmarshalForIntervalStyle(string(srcStr), "postgres", dv)
	case string:
		return unmarshalForIntervalStyle(srcStr, "postgres", dv)
	}

	return fmt.Errorf("duration column was not text; type %T", src)
}

// Value implements Valuer.
func (dv DurationValuer) Value() (driver.Value, error) {
	return marshalForIntervalStyle(time.Duration(dv), "postgres"), nil
}

// marshalForIntervalStyle converts a time.Duration into a string representation compatible
// with the 'postgres' style interval format (hh:mm:ss).
func marshalForIntervalStyle(duration time.Duration, style string) string {
	switch style {
	case "postgres":
		milliseconds := (duration / time.Millisecond) % 1000
		seconds := (duration / time.Second) % 60
		minutes := (duration / time.Minute) % 60
		hours := duration / time.Hour

		str := strings.Builder{}
		str.WriteString(fmt.Sprintf("%02d", hours))
		str.WriteString(":")
		str.WriteString(fmt.Sprintf("%02d", minutes))
		str.WriteString(":")
		str.WriteString(fmt.Sprintf("%02d", seconds))
		if milliseconds != 0 {
			str.WriteString(fmt.Sprintf(".%03d", milliseconds))
		}
		return str.String()
	}
	return "only postgres style supported for intervals"
}

const (
	// for state machine when parsing a postgres interval string
	stateNumber = iota
	stateUnit
)

// unmarshalForIntervalStyle converts a 'postgres' style interval format
// ('x years y mons z days hh:mm:ss') into the time.Duration wrapped by the
// given *DurationValuer.
func unmarshalForIntervalStyle(src, style string, dv *DurationValuer) error {

	if style != "postgres" {
		return fmt.Errorf("only 'postgres' style interval format supported")
	}

	value := 0 // the value of the last parsed number
	state := -1
	hours := 0 // cummulated number of hours we parsed and converted so far
	parts := strings.Split(src, " ")

	// We will convert and sum up years, mons and days to hours and convert
	// '01:02:03' to the time.ParseDuration format '1h2m3s'.
	i := 0
	for {
		if i >= len(parts) {
			// we reached the end without finding 'hh:mm:ss' and just return the hours we parsed so far
			if state == stateNumber {
				return fmt.Errorf("could not parse interval string")
			}
			duration, err := time.ParseDuration(strconv.Itoa(hours) + "h")
			if err != nil {
				return fmt.Errorf("could not parse interval string")
			}
			*dv = DurationValuer(duration)
			return nil
		}

		// try to parse a number
		n, err := strconv.Atoi(strings.TrimLeft(parts[i], "0"))
		if err == nil {
			// we found a number and just memorize it for the next loop where we evaluate the time unit
			if state == stateNumber {
				return fmt.Errorf("could not parse interval string")
			}
			value = n
			state = stateNumber
			i++
			continue
		}

		if i > 0 {
			// we either found a unit and need to convert the value we found before
			// accordingly and add it to hours, or we found the final 'hh:mm:ss'
			switch {
			case strings.Contains(parts[i], "year") && state == stateNumber:
				hours = hours + value*24*365
			case strings.Contains(parts[i], "mon") && state == stateNumber:
				hours = hours + value*24*30
			case strings.Contains(parts[i], "day") && state == stateNumber:
				hours = hours + value*24
			case strings.Contains(parts[i], ":") && state == stateUnit:
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
			state = stateUnit
			i++
			continue
		}

		// we probably found a single hh:mm:ss string right at the beginning
		if i+1 != len(parts) {
			// there shouldn't be more parts after hh:mm:ss
			return fmt.Errorf("could not parse interval string")
		}
		duration, err := parseHMS(parts[i])
		if err != nil {
			return err
		}
		*dv = DurationValuer(duration)
		return nil
	}
}

// parseHMS is converting '01:02:03' to '1h2m3s'.
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

const (
	// env var names
	postgresUserVar = "POSTGRES_USER"
	postgresPWVar   = "POSTGRES_PW"
	postgresHostVar = "POSTGRES_HOST"
	postgresPortVar = "POSTGRES_PORT"
	postgresDBVar   = "POSTGRES_DB"
	postgresOptsVar = "POSTGRES_OPTS"
)

// PostgresURIfromEnvVars is constructing a postgres uri string from env vars.
func PostgresURIfromEnvVars() string {
	user := os.Getenv(postgresUserVar)
	pw := os.Getenv(postgresPWVar)
	host := os.Getenv(postgresHostVar)
	port := os.Getenv(postgresPortVar)
	db := os.Getenv(postgresDBVar)
	opts := os.Getenv(postgresOptsVar)

	if user == "" {
		user = "postgres"
	}
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if db == "" {
		db = "postgres"
	}

	pwStr := ""
	if pw != "" {
		pwStr = ":" + pw
	}

	optStr := "?sslmode=disable"
	if opts != "" {
		optStr = opts
	}

	return fmt.Sprintf("postgres://%s%s@%s:%s/%s%s", user, pwStr, host, port, db, optStr)
}
