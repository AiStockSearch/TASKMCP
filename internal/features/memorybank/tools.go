package memorybank

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/google/uuid"
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

func repoKeyRequired(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	repoKey := strings.TrimSpace(req.GetString("repo_key", ""))
	if repoKey == "" {
		repoKey = strings.TrimSpace(os.Getenv("DEFAULT_REPO_KEY"))
	}
	if repoKey == "" {
		return "", mcp.NewToolResultError("missing repo_key (or set DEFAULT_REPO_KEY env var)")
	}
	return repoKey, nil
}

func repoKeyOptional(req mcp.CallToolRequest) *string {
	repoKey := strings.TrimSpace(req.GetString("repo_key", ""))
	if repoKey == "" {
		repoKey = strings.TrimSpace(os.Getenv("DEFAULT_REPO_KEY"))
	}
	if repoKey == "" {
		return nil
	}
	return &repoKey
}

func (t *Tools) GetDocument(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyRequired(req)
	if r != nil {
		return r, nil
	}
	docKey, err := req.RequireString("doc_key")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	doc, ok, err := t.svc.GetDocument(ctx, repoKey, docKey)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Err("document not found")
	}
	return mcputil.Structured(doc, doc.DocKey)
}

func (t *Tools) UpsertDocument(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyRequired(req)
	if r != nil {
		return r, nil
	}
	docKey, err := req.RequireString("doc_key")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	docType, err := req.RequireString("doc_type")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	title := strings.TrimSpace(req.GetString("title", ""))
	content, err := req.RequireString("content")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	out, err := t.svc.UpsertDocument(ctx, repoKey, docKey, docType, title, content)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, "document upserted")
}

func (t *Tools) ListDocuments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyRequired(req)
	if r != nil {
		return r, nil
	}
	docType := strings.TrimSpace(req.GetString("doc_type", ""))
	limit := req.GetInt("limit", 50)
	offset := req.GetInt("offset", 0)
	out, err := t.svc.ListDocuments(ctx, repoKey, docType, limit, offset)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, "documents listed")
}

func (t *Tools) GetState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyRequired(req)
	if r != nil {
		return r, nil
	}
	out, err := t.svc.GetState(ctx, repoKey)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, "state fetched")
}

func (t *Tools) SetState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyRequired(req)
	if r != nil {
		return r, nil
	}
	raw := req.GetArguments()["state_json"]
	b, err := json.Marshal(raw)
	if err != nil {
		return mcputil.Err("invalid state_json")
	}
	if err := t.svc.SetState(ctx, repoKey, json.RawMessage(b)); err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Text("state updated")
}

func (t *Tools) RulesList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey := repoKeyOptional(req)
	scopePrefix := strings.TrimSpace(req.GetString("scope_prefix", ""))
	enabledOnly := req.GetBool("enabled_only", true)
	limit := req.GetInt("limit", 200)
	offset := req.GetInt("offset", 0)
	out, err := t.svc.ListRules(ctx, repoKey, scopePrefix, enabledOnly, limit, offset)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, "rules listed")
}

func (t *Tools) RulesUpsert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey := repoKeyOptional(req)

	ruleIDStr := strings.TrimSpace(req.GetString("rule_id", ""))
	var ruleID *uuid.UUID
	if ruleIDStr != "" {
		id, err := uuid.Parse(ruleIDStr)
		if err != nil {
			return mcputil.Err("invalid rule_id (expected UUID)")
		}
		ruleID = &id
	}

	scope, err := req.RequireString("scope")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	priority := req.GetInt("priority", 100)
	title, err := req.RequireString("title")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	enabled := req.GetBool("enabled", true)

	id, err := t.svc.UpsertRule(ctx, repoKey, ruleID, scope, priority, title, content, enabled)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(map[string]any{"rule_id": id.String()}, "rule upserted")
}

func (t *Tools) RulesEnable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return t.setRuleEnabled(ctx, req, true)
}

func (t *Tools) RulesDisable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return t.setRuleEnabled(ctx, req, false)
}

func (t *Tools) setRuleEnabled(ctx context.Context, req mcp.CallToolRequest, enabled bool) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey := repoKeyOptional(req)

	ruleID, r := mcputil.RequireUUID(req, "rule_id")
	if r != nil {
		return r, nil
	}
	ok, err := t.svc.SetRuleEnabled(ctx, repoKey, ruleID, enabled)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Err("rule not found")
	}
	if enabled {
		return mcputil.Text("rule enabled")
	}
	return mcputil.Text("rule disabled")
}

func (t *Tools) ListVersions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyRequired(req)
	if r != nil {
		return r, nil
	}
	docKey, err := req.RequireString("doc_key")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	limit := req.GetInt("limit", 50)
	offset := req.GetInt("offset", 0)
	out, ok, err := t.svc.ListVersions(ctx, repoKey, docKey, limit, offset)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Err("document not found")
	}
	return mcputil.Structured(out, "versions listed")
}

func (t *Tools) GetDocumentVersion(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyRequired(req)
	if r != nil {
		return r, nil
	}
	docKey, err := req.RequireString("doc_key")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	version := req.GetInt("version", 0)
	doc, ok, err := t.svc.GetDocumentVersion(ctx, repoKey, docKey, version)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Err("document version not found")
	}
	return mcputil.Structured(doc, "document version")
}

func (t *Tools) RulesApplyPreview(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey := repoKeyOptional(req)
	scopes, err := req.RequireStringSlice("scopes")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	out, err := t.svc.RulesApplyPreview(ctx, repoKey, scopes)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, "rules preview")
}

