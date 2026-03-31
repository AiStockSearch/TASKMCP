package githubissues

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) GetLink(ctx context.Context, entityType string, entityID uuid.UUID) (*LinkDTO, error) {
	var dto LinkDTO
	row := r.db.QueryRowContext(ctx, `
SELECT entity_type, entity_id, repo_owner, repo_name, issue_number, issue_url
FROM github_links
WHERE entity_type = $1 AND entity_id = $2
`, entityType, entityID)
	if err := row.Scan(&dto.EntityType, &dto.EntityID, &dto.RepoOwner, &dto.RepoName, &dto.IssueNumber, &dto.IssueURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query github link: %w", err)
	}
	return &dto, nil
}

func (r *Repo) InsertLinkIfAbsent(ctx context.Context, link LinkDTO) error {
	id := uuid.New()
	_, err := r.db.ExecContext(ctx, `
INSERT INTO github_links (id, entity_type, entity_id, repo_owner, repo_name, issue_number, issue_url)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (entity_type, entity_id) DO NOTHING
`, id, link.EntityType, link.EntityID, link.RepoOwner, link.RepoName, link.IssueNumber, link.IssueURL)
	if err != nil {
		return fmt.Errorf("insert github link: %w", err)
	}
	return nil
}

