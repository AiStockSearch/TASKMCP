package postgres

import (
	"context"
	"database/sql"
)

type DBGuard struct {
	db *sql.DB
}

func NewDBGuard(db *sql.DB) *DBGuard {
	return &DBGuard{db: db}
}

func (g *DBGuard) Ensure(ctx context.Context) error {
	return EnsureReachable(ctx, g.db)
}

