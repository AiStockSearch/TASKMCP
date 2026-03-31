package githubissues

import (
	"context"
	"fmt"
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

func (t *Tools) GetIssueLink(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	entityType, err := req.RequireString("entity_type")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	entityType = strings.TrimSpace(entityType)
	if entityType != "epic" && entityType != "requirement" && entityType != "task" {
		return mcputil.Err("invalid entity_type (expected epic|requirement|task)")
	}
	entityID, r := mcputil.RequireUUID(req, "entity_id")
	if r != nil {
		return r, nil
	}

	link, err := t.svc.GetStoredLink(ctx, entityType, entityID)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	if link == nil {
		return mcputil.Text("No GitHub link found for this entity.")
	}
	return mcputil.Structured(link, link.IssueURL)
}

func (t *Tools) CreateIssueForTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.db.Ensure(ctx); err != nil {
		return mcputil.Err(err.Error())
	}
	if err := t.svc.ghApp.ConfigError(); err != nil {
		return mcputil.Err(err.Error())
	}

	taskID, r := mcputil.RequireUUID(req, "task_id")
	if r != nil {
		return r, nil
	}
	owner, err := req.RequireString("repo_owner")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	repo, err := req.RequireString("repo_name")
	if err != nil {
		return mcputil.Err(err.Error())
	}
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return mcputil.Err("repo_owner and repo_name cannot be empty")
	}

	titleOverride := strings.TrimSpace(req.GetString("title_override", ""))
	bodyMode := strings.TrimSpace(req.GetString("body_mode", "from_task_description"))

	link, err := t.svc.CreateIssueForTask(ctx, taskID, owner, repo, titleOverride, bodyMode)
	if err != nil {
		return mcputil.Err(err.Error())
	}
	return mcputil.Structured(link, fmt.Sprintf("Created issue #%d", link.IssueNumber))
}

