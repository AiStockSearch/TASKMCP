package kb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

type UpsertDocumentInput struct {
	ProjectID   uuid.UUID
	SourceType  string
	SourcePath  string
	Title       string
	ContentHash string
}

func (r *Repo) EnsureDocument(ctx context.Context, in UpsertDocumentInput) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.db.ExecContext(ctx, `
INSERT INTO documents (id, project_id, source_type, source_path, title, content_hash)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
ON CONFLICT (project_id, source_type, source_path, content_hash) DO NOTHING
`, id, in.ProjectID, in.SourceType, in.SourcePath, in.Title, in.ContentHash)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("insert document: %w", err)
	}

	// Read the canonical id.
	err = r.db.QueryRowContext(ctx, `
SELECT id
FROM documents
WHERE project_id = $1 AND source_type = $2 AND source_path = $3 AND content_hash = $4
`, in.ProjectID, in.SourceType, in.SourcePath, in.ContentHash).Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("query document: %w", err)
	}
	return id, nil
}

type UpsertChunkInput struct {
	ProjectID   uuid.UUID
	DocumentID  uuid.UUID
	ChunkIndex  int
	Content     string
	Embedding   pgvector.Vector
	MetadataJSON []byte
}

func (r *Repo) UpsertChunk(ctx context.Context, in UpsertChunkInput) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO document_chunks (id, document_id, project_id, chunk_index, content, embedding, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
ON CONFLICT (document_id, chunk_index)
DO UPDATE SET content = EXCLUDED.content, embedding = EXCLUDED.embedding, metadata = EXCLUDED.metadata
`, uuid.New(), in.DocumentID, in.ProjectID, in.ChunkIndex, in.Content, in.Embedding, string(in.MetadataJSON))
	if err != nil {
		return fmt.Errorf("upsert chunk: %w", err)
	}
	return nil
}

type SearchResult struct {
	DocumentID uuid.UUID `json:"document_id"`
	SourceType string    `json:"source_type"`
	SourcePath string    `json:"source_path"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content"`
	Score      float64   `json:"score"`
	CreatedAt  time.Time `json:"created_at"`
}

func (r *Repo) Search(ctx context.Context, projectID uuid.UUID, query pgvector.Vector, topK int) ([]SearchResult, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT c.document_id, d.source_type, d.source_path, c.chunk_index, c.content,
       1 - (c.embedding <=> $2) AS score,
       c.created_at
FROM document_chunks c
JOIN documents d ON d.id = c.document_id
WHERE c.project_id = $1
ORDER BY c.embedding <=> $2
LIMIT $3
`, projectID, query, topK)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []SearchResult
	for rows.Next() {
		var r0 SearchResult
		if err := rows.Scan(&r0.DocumentID, &r0.SourceType, &r0.SourcePath, &r0.ChunkIndex, &r0.Content, &r0.Score, &r0.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		out = append(out, r0)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}
	return out, nil
}

func (r *Repo) SearchFTS(ctx context.Context, projectID uuid.UUID, queryText string, topK int) ([]SearchResult, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT c.document_id, d.source_type, d.source_path, c.chunk_index, c.content,
       ts_rank(c.content_tsv, plainto_tsquery('simple', $2)) AS score,
       c.created_at
FROM document_chunks c
JOIN documents d ON d.id = c.document_id
WHERE c.project_id = $1 AND c.content_tsv @@ plainto_tsquery('simple', $2)
ORDER BY score DESC, c.created_at DESC
LIMIT $3
`, projectID, queryText, topK)
	if err != nil {
		return nil, fmt.Errorf("fts search chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []SearchResult
	for rows.Next() {
		var r0 SearchResult
		if err := rows.Scan(&r0.DocumentID, &r0.SourceType, &r0.SourcePath, &r0.ChunkIndex, &r0.Content, &r0.Score, &r0.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan fts result: %w", err)
		}
		out = append(out, r0)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fts results: %w", err)
	}
	return out, nil
}

