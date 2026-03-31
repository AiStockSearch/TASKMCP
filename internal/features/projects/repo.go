package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func ParseRepoKey(repoKey string) (owner string, name string, key string, err error) {
	repoKey = strings.TrimSpace(repoKey)
	parts := strings.Split(repoKey, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", "", fmt.Errorf("invalid repo_key (expected owner/repo)")
	}
	owner = strings.TrimSpace(parts[0])
	name = strings.TrimSpace(parts[1])
	return owner, name, owner + "/" + name, nil
}

func (r *Repo) EnsureProject(ctx context.Context, repoKey string) (uuid.UUID, error) {
	owner, name, key, err := ParseRepoKey(repoKey)
	if err != nil {
		return uuid.UUID{}, err
	}

	var id uuid.UUID
	err = r.db.QueryRowContext(ctx, `SELECT id FROM projects WHERE repo_key = $1`, key).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.UUID{}, fmt.Errorf("query project: %w", err)
	}

	id = uuid.New()
	_, err = r.db.ExecContext(ctx, `
INSERT INTO projects (id, repo_owner, repo_name, repo_key)
VALUES ($1, $2, $3, $4)
ON CONFLICT (repo_key) DO NOTHING
`, id, owner, name, key)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("insert project: %w", err)
	}

	// Re-read to handle concurrent inserts.
	err = r.db.QueryRowContext(ctx, `SELECT id FROM projects WHERE repo_key = $1`, key).Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("requery project: %w", err)
	}
	return id, nil
}

