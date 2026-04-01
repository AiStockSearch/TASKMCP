package memorybank

import "time"

var AllowedDocTypes = map[string]struct{}{
	"tasks":         {},
	"activeContext": {},
	"progress":      {},
	"plan":          {},
	"adr":           {},
	"refactor_plan": {},
	"reflection":    {},
	"archive":       {},
}

type Document struct {
	DocKey    string `json:"doc_key"`
	DocType   string `json:"doc_type"`
	Title     string `json:"title"`
	Version   int    `json:"version"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

type DocumentListItem struct {
	DocKey     string `json:"doc_key"`
	DocType    string `json:"doc_type"`
	Title      string `json:"title"`
	Version    int    `json:"version"`
	UpdatedAt  string `json:"updated_at"`
}

type UpsertDocumentResult struct {
	DocID   string `json:"doc_id"`
	Version int    `json:"version"`
}

type Rule struct {
	ID        string `json:"id"`
	ProjectID *string `json:"project_id,omitempty"`
	Scope     string `json:"scope"`
	Priority  int    `json:"priority"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Enabled   bool   `json:"enabled"`
	UpdatedAt string `json:"updated_at"`
}

type ruleRow struct {
	id        string
	projectID *string
	scope     string
	priority  int
	title     string
	content   string
	enabled   bool
	updatedAt time.Time
}

