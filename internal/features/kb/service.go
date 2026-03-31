package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"mcp-vault-bridge/internal/features/projects"
)

type Service struct {
	repo     *Repo
	projects ProjectsResolver
	dim     int
}

type ProjectsResolver interface {
	Resolve(ctx context.Context, repoKey string) (uuid.UUID, error)
}

func NewService(repo *Repo, projects ProjectsResolver) *Service {
	return &Service{repo: repo, projects: projects, dim: 1536}
}

func (s *Service) SetEmbeddingDim(dim int) {
	if dim > 0 {
		s.dim = dim
	}
}

type UpsertDocumentChunksInput struct {
	RepoKey    string
	SourceType string
	SourcePath string
	Title      string
	FullText   string
	Chunks     []ChunkInput
}

type ChunkInput struct {
	ChunkIndex int             `json:"chunk_index"`
	Content    string          `json:"content"`
	Embedding  []float32       `json:"embedding"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type UpsertDocumentChunksResult struct {
	DocumentID string `json:"document_id"`
	ChunksUpserted int `json:"chunks_upserted"`
}

func (s *Service) UpsertDocumentChunks(ctx context.Context, in UpsertDocumentChunksInput) (UpsertDocumentChunksResult, error) {
	_, _, repoKey, err := projects.ParseRepoKey(in.RepoKey)
	if err != nil {
		return UpsertDocumentChunksResult{}, err
	}
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return UpsertDocumentChunksResult{}, err
	}

	st := strings.TrimSpace(in.SourceType)
	if st == "" {
		st = "markdown"
	}

	hash := sha256.Sum256([]byte(in.FullText))
	contentHash := hex.EncodeToString(hash[:])

	docID, err := s.repo.EnsureDocument(ctx, UpsertDocumentInput{
		ProjectID:   projectID,
		SourceType:  st,
		SourcePath:  strings.TrimSpace(in.SourcePath),
		Title:       strings.TrimSpace(in.Title),
		ContentHash: contentHash,
	})
	if err != nil {
		return UpsertDocumentChunksResult{}, err
	}

	count := 0
	for _, c := range in.Chunks {
		if c.ChunkIndex < 0 {
			return UpsertDocumentChunksResult{}, fmt.Errorf("chunk_index must be >= 0")
		}
		if strings.TrimSpace(c.Content) == "" {
			return UpsertDocumentChunksResult{}, fmt.Errorf("chunk content cannot be empty")
		}
		if len(c.Embedding) == 0 {
			return UpsertDocumentChunksResult{}, fmt.Errorf("embedding cannot be empty")
		}
		if s.dim > 0 && len(c.Embedding) != s.dim {
			return UpsertDocumentChunksResult{}, fmt.Errorf("embedding dimension mismatch: got %d want %d", len(c.Embedding), s.dim)
		}
		meta := c.Metadata
		if len(meta) == 0 {
			meta = json.RawMessage(`{}`)
		}

		if err := s.repo.UpsertChunk(ctx, UpsertChunkInput{
			ProjectID:    projectID,
			DocumentID:   docID,
			ChunkIndex:   c.ChunkIndex,
			Content:      c.Content,
			Embedding:    pgvector.NewVector(c.Embedding),
			MetadataJSON: []byte(meta),
		}); err != nil {
			return UpsertDocumentChunksResult{}, err
		}
		count++
	}

	return UpsertDocumentChunksResult{DocumentID: docID.String(), ChunksUpserted: count}, nil
}

func (s *Service) Search(ctx context.Context, repoKey string, queryEmbedding []float32, topK int) ([]SearchResult, error) {
	_, _, rk, err := projects.ParseRepoKey(repoKey)
	if err != nil {
		return nil, err
	}
	projectID, err := s.projects.Resolve(ctx, rk)
	if err != nil {
		return nil, err
	}
	if topK <= 0 {
		topK = 8
	}
	if topK > 50 {
		topK = 50
	}
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("query_embedding cannot be empty")
	}
	if s.dim > 0 && len(queryEmbedding) != s.dim {
		return nil, fmt.Errorf("query_embedding dimension mismatch: got %d want %d", len(queryEmbedding), s.dim)
	}
	return s.repo.Search(ctx, projectID, pgvector.NewVector(queryEmbedding), topK)
}

