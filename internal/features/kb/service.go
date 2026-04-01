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

type HybridSearchInput struct {
	RepoKey        string
	QueryText      string
	QueryEmbedding []float32
	TopK           int
	FTSWeight      float64
	VecWeight      float64
}

func (s *Service) HybridSearch(ctx context.Context, in HybridSearchInput) ([]SearchResult, error) {
	_, _, rk, err := projects.ParseRepoKey(in.RepoKey)
	if err != nil {
		return nil, err
	}
	projectID, err := s.projects.Resolve(ctx, rk)
	if err != nil {
		return nil, err
	}

	topK := in.TopK
	if topK <= 0 {
		topK = 8
	}
	if topK > 50 {
		topK = 50
	}
	ftsW := in.FTSWeight
	vecW := in.VecWeight
	if ftsW == 0 && vecW == 0 {
		ftsW = 0.3
		vecW = 0.7
	}

	var fts []SearchResult
	if strings.TrimSpace(in.QueryText) != "" {
		fts, err = s.repo.SearchFTS(ctx, projectID, in.QueryText, topK*4)
		if err != nil {
			return nil, err
		}
	}

	if len(in.QueryEmbedding) == 0 {
		// FTS-only mode
		if len(fts) > topK {
			return fts[:topK], nil
		}
		return fts, nil
	}
	if s.dim > 0 && len(in.QueryEmbedding) != s.dim {
		return nil, fmt.Errorf("query_embedding dimension mismatch: got %d want %d", len(in.QueryEmbedding), s.dim)
	}

	vec, err := s.repo.Search(ctx, projectID, pgvector.NewVector(in.QueryEmbedding), topK*4)
	if err != nil {
		return nil, err
	}

	return mergeHybrid(fts, vec, topK, ftsW, vecW), nil
}

func mergeHybrid(fts []SearchResult, vec []SearchResult, topK int, ftsW, vecW float64) []SearchResult {
	type key struct {
		doc uuid.UUID
		idx int
	}
	type agg struct {
		base SearchResult
		fts  float64
		vec  float64
	}

	m := make(map[key]*agg, len(fts)+len(vec))
	maxFTS := 0.0
	for _, r := range fts {
		if r.Score > maxFTS {
			maxFTS = r.Score
		}
		k := key{doc: r.DocumentID, idx: r.ChunkIndex}
		a := m[k]
		if a == nil {
			cp := r
			a = &agg{base: cp}
			m[k] = a
		}
		a.fts = r.Score
	}
	maxVec := 0.0
	for _, r := range vec {
		if r.Score > maxVec {
			maxVec = r.Score
		}
		k := key{doc: r.DocumentID, idx: r.ChunkIndex}
		a := m[k]
		if a == nil {
			cp := r
			a = &agg{base: cp}
			m[k] = a
		}
		a.vec = r.Score
	}

	denFTS := maxFTS
	if denFTS == 0 {
		denFTS = 1
	}
	denVec := maxVec
	if denVec == 0 {
		denVec = 1
	}

	out := make([]SearchResult, 0, len(m))
	for _, a := range m {
		nFTS := a.fts / denFTS
		nVec := a.vec / denVec
		a.base.Score = ftsW*nFTS + vecW*nVec
		out = append(out, a.base)
	}

	// sort by score desc
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Score > out[i].Score {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > topK {
		out = out[:topK]
	}
	return out
}

