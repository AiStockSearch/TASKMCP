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
	return db, nil
}

func EnsureReachable(ctx context.Context, db *sql.DB) error {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(cctx); err != nil {
		return fmt.Errorf("database not reachable: %w", err)
	}
	return nil
}

