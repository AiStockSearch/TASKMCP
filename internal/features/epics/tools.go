package epics

import (
	"context"
	"fmt"
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

func (t *Tools) CreateEpic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	desc := strings.TrimSpace(req.GetString("description", ""))

	id, err := t.svc.CreateEpic(ctx, repoKey, title, desc)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(map[string]any{"epic_id": id.String()}, "Epic created.")
}

func (t *Tools) ListEpics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}

	status := strings.TrimSpace(req.GetString("status", ""))
	if status != "" && status != "open" && status != "done" && status != "archived" {
		return mcputil.Err("invalid status (expected open|done|archived)")
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

	out, err := t.svc.ListEpics(ctx, repoKey, status, limit, offset)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	resp := map[string]any{"epics": out, "count": len(out)}
	return mcputil.Structured(resp, fmt.Sprintf("Listed %d epic(s).", len(out)))
}

func (t *Tools) LinkRequirementToEpic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}
	requirementID, r := mcputil.RequireUUID(req, "requirement_id")
	if r != nil {
		return r, nil
	}
	epicID, r := mcputil.RequireUUID(req, "epic_id")
	if r != nil {
		return r, nil
	}

	ok, err := t.svc.LinkRequirementToEpic(ctx, repoKey, requirementID, epicID)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Err("requirement not found")
	}
	return mcputil.Text("Requirement linked to epic successfully.")
}

func (t *Tools) LinkTaskToEpic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	epicID, r := mcputil.RequireUUID(req, "epic_id")
	if r != nil {
		return r, nil
	}

	ok, err := t.svc.LinkTaskToEpic(ctx, repoKey, taskID, epicID)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if !ok {
		return mcputil.Err("task not found")
	}
	return mcputil.Text("Task linked to epic successfully.")
}

func (t *Tools) EpicAddTasks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}
	epicID, r := mcputil.RequireUUID(req, "epic_id")
	if r != nil {
		return r, nil
	}

	ids, err := req.RequireStringSlice("task_ids")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	var taskIDs []uuid.UUID
	for _, s := range ids {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return mcputil.Err("invalid task_ids (expected UUIDs)")
		}
		taskIDs = append(taskIDs, id)
	}

	out, err := t.svc.AddTasksToEpic(ctx, repoKey, epicID, taskIDs)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(out, "tasks linked to epic")
}

func (t *Tools) EpicListTasks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	repoKey, r := repoKeyFromReq(req)
	if r != nil {
		return r, nil
	}
	epicID, r := mcputil.RequireUUID(req, "epic_id")
	if r != nil {
		return r, nil
	}
	includeFiles := req.GetBool("include_files", false)

	out, err := t.svc.ListEpicTasks(ctx, repoKey, epicID, includeFiles)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	resp := map[string]any{"tasks": out, "count": len(out)}
	return mcputil.Structured(resp, "epic tasks listed")
}

