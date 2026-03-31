package kb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp-vault-bridge/internal/mcputil"
	"mcp-vault-bridge/internal/storage/postgres"
)

type Tools struct {
	svc *Service
	db  *postgres.DBGuard
}

func NewTools(svc *Service, db *postgres.DBGuard) *Tools {
	return &Tools{svc: svc, db: db}
}

func repoKeyFromReq(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	repoKey := strings.TrimSpace(req.GetString("repo_key", ""))
	if repoKey == "" {
		repoKey = strings.TrimSpace(os.Getenv("DEFAULT_REPO_KEY"))
	}
	if repoKey == "" {
		return "", mcp.NewToolResultError("missing repo_key (or set DEFAULT_REPO_KEY env var)")
	}
	return repoKey, nil
}

func (t *Tools) UpsertDocumentChunks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}

	sourceType := strings.TrimSpace(req.GetString("source_type", "markdown"))
	sourcePath, err := req.RequireString("source_path")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	title := strings.TrimSpace(req.GetString("title", ""))
	fullText, err := req.RequireString("full_text")
	if err != nil {
		return mcputil.Err(err.Error())
	}

	raw := req.GetArguments()["chunks"]
	b, err := json.Marshal(raw)
	if err != nil {
		return mcputil.Err(fmt.Sprintf("invalid chunks: %v", err))
	}
	var chunks []ChunkInput
	if err := json.Unmarshal(b, &chunks); err != nil {
		return mcputil.Err(fmt.Sprintf("invalid chunks: %v", err))
	}

	out, err := t.svc.UpsertDocumentChunks(ctx, UpsertDocumentChunksInput{
		RepoKey:    repoKey,
		SourceType: sourceType,
		SourcePath: sourcePath,
		Title:      title,
		FullText:   fullText,
		Chunks:     chunks,
	})
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, "Document chunks upserted.")
}

func (t *Tools) SearchContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}

	raw := req.GetArguments()["query_embedding"]
	b, err := json.Marshal(raw)
	if err != nil {
		return mcputil.Err(fmt.Sprintf("invalid query_embedding: %v", err))
	}
	var emb []float32
	if err := json.Unmarshal(b, &emb); err != nil {
		return mcputil.Err(fmt.Sprintf("invalid query_embedding: %v", err))
	}

	topK := req.GetInt("top_k", 8)
	out, err := t.svc.Search(ctx, repoKey, emb, topK)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, fmt.Sprintf("Found %d chunk(s).", len(out)))
}

func (t *Tools) ChunkMarkdown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// No DB required; this is pure preprocessing for better RAG quality.
	text, err := req.RequireString("text")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	maxChars := req.GetInt("max_chars", 3000)
	overlapChars := req.GetInt("overlap_chars", 300)

	out := chunkMarkdown(text, maxChars, overlapChars)
	return mcputil.Structured(out, fmt.Sprintf("Chunked into %d chunk(s).", len(out)))
}

