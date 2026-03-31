package mcputil

import (
	"strings"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

func Err(msg string) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(msg), nil
}

func Text(msg string) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(msg), nil
}

func Structured(v any, fallbackText string) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultStructured(v, fallbackText), nil
}

func RequireUUID(req mcp.CallToolRequest, key string) (uuid.UUID, *mcp.CallToolResult) {
	s, err := req.RequireString(key)
	if err != nil {
		r := mcp.NewToolResultError(err.Error())
		return uuid.UUID{}, r
	}
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		r := mcp.NewToolResultError("invalid " + key + " (expected UUID)")
		return uuid.UUID{}, r
	}
	return id, nil
}

func OptionalUUID(req mcp.CallToolRequest, key string) (*uuid.UUID, *mcp.CallToolResult) {
	s := strings.TrimSpace(req.GetString(key, ""))
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		r := mcp.NewToolResultError("invalid " + key + " (expected UUID)")
		return nil, r
	}
	return &id, nil
}

