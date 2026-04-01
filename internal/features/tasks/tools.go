package tasks

import (
	"context"
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

func (t *Tools) GetNextTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}

	resp, ok, err := t.svc.GetNextTask(ctx, repoKey)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Text("No todo tasks available.")
	}
	return mcputil.Structured(resp, fmt.Sprintf("Task %s: %s", resp.TaskID, resp.Title))
}

func (t *Tools) CompleteTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}
	taskID, r := mcputil.RequireUUID(req, "task_id")
	if r != nil {
		return r, nil
	}
	report, err := req.RequireString("report")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	ok, err := t.svc.CompleteTask(ctx, repoKey, taskID, report)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Err("task not found or not in_progress")
	}
	return mcputil.Text("Task completed successfully.")
}

func (t *Tools) AddContextFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}
	taskID, r := mcputil.RequireUUID(req, "task_id")
	if r != nil {
		return r, nil
	}
	filePath, err := req.RequireString("file_path")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if err := t.svc.AddContextFile(ctx, repoKey, taskID, filePath); err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Text("Context file added successfully.")
}

func (t *Tools) ListTasks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}

	status := strings.TrimSpace(req.GetString("status", ""))
	if status != "" && status != "todo" && status != "in_progress" && status != "done" {
		return mcputil.Err("invalid status (expected todo|in_progress|done)")
	}

	requirementID, r := mcputil.OptionalUUID(req, "requirement_id")
	if r != nil {
		return r, nil
	}
	epicID, r := mcputil.OptionalUUID(req, "epic_id")
	if r != nil {
		return r, nil
	}

	limit := req.GetInt("limit", 50)
	offset := req.GetInt("offset", 0)
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	order := strings.TrimSpace(req.GetString("order", "priority"))
	if order == "" {
		order = "priority"
	}
	if order != "priority" && order != "created_at" {
		return mcputil.Err("invalid order (expected priority|created_at)")
	}

	includeFiles := req.GetBool("include_files", false)
	includeReqSpec := req.GetBool("include_requirement_spec", false)

	out, err := t.svc.ListTasks(ctx, repoKey, ListInput{
		Status:                 status,
		RequirementID:          requirementID,
		EpicID:                 epicID,
		Limit:                  limit,
		Offset:                 offset,
		Order:                  order,
		IncludeFiles:           includeFiles,
		IncludeRequirementSpec: includeReqSpec,
	})
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, fmt.Sprintf("Listed %d task(s).", len(out)))
}

func (t *Tools) GetTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}
	taskID, r := mcputil.RequireUUID(req, "task_id")
	if r != nil {
		return r, nil
	}

	dto, ok, err := t.svc.GetTask(ctx, repoKey, taskID)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Err("task not found")
	}
	resp := map[string]any{"task": dto}
	return mcputil.Structured(resp, fmt.Sprintf("Task %s: %s", dto.ID, dto.Title))
}

