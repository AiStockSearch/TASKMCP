package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// Avoid stale pooled connections ("driver: bad connection") after idle timeouts or DB restarts.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(90 * time.Second)
	return db, nil
}

func EnsureReachable(ctx context.Context, db *sql.DB) error {
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := db.PingContext(cctx)
		cancel()
		if err == nil {
			return nil
		}
		last = err
		if attempt == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(75 * time.Millisecond):
			}
		}
	}
	return fmt.Errorf("database not reachable: %w", last)
}

