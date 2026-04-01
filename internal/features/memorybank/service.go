package memorybank

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"mcp-vault-bridge/internal/features/projects"
)

type ProjectsResolver interface {
	Resolve(ctx context.Context, repoKey string) (uuid.UUID, error)
}

type Service struct {
	repo     *Repo
	projects ProjectsResolver
}

func NewService(repo *Repo, projects ProjectsResolver) *Service {
	return &Service{repo: repo, projects: projects}
}

func (s *Service) resolveProject(ctx context.Context, repoKey string) (uuid.UUID, error) {
	_, _, rk, err := projects.ParseRepoKey(repoKey)
	if err != nil {
		return uuid.UUID{}, err
	}
	return s.projects.Resolve(ctx, rk)
}

func (s *Service) GetDocument(ctx context.Context, repoKey, docKey string) (Document, bool, error) {
	pid, err := s.resolveProject(ctx, repoKey)
	if err != nil {
		return Document{}, false, err
	}
	return s.repo.GetDocument(ctx, pid, strings.TrimSpace(docKey))
}

func (s *Service) UpsertDocument(ctx context.Context, repoKey, docKey, docType, title, content string) (UpsertDocumentResult, error) {
	pid, err := s.resolveProject(ctx, repoKey)
	if err != nil {
		return UpsertDocumentResult{}, err
	}
	docKey = strings.TrimSpace(docKey)
	if docKey == "" {
		return UpsertDocumentResult{}, fmt.Errorf("doc_key cannot be empty")
	}
	docType = strings.TrimSpace(docType)
	if _, ok := AllowedDocTypes[docType]; !ok {
		return UpsertDocumentResult{}, fmt.Errorf("invalid doc_type")
	}
	return s.repo.UpsertDocument(ctx, pid, docKey, docType, strings.TrimSpace(title), content)
}

func (s *Service) ListDocuments(ctx context.Context, repoKey, docType string, limit, offset int) ([]DocumentListItem, error) {
	pid, err := s.resolveProject(ctx, repoKey)
	if err != nil {
		return nil, err
	}
	docType = strings.TrimSpace(docType)
	if docType != "" {
		if _, ok := AllowedDocTypes[docType]; !ok {
			return nil, fmt.Errorf("invalid doc_type")
		}
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListDocuments(ctx, pid, docType, limit, offset)
}

func (s *Service) GetState(ctx context.Context, repoKey string) (json.RawMessage, error) {
	pid, err := s.resolveProject(ctx, repoKey)
	if err != nil {
		return nil, err
	}
	state, _, err := s.repo.GetState(ctx, pid)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(state), nil
}

func (s *Service) SetState(ctx context.Context, repoKey string, state json.RawMessage) error {
	pid, err := s.resolveProject(ctx, repoKey)
	if err != nil {
		return err
	}
	if !json.Valid(state) {
		return fmt.Errorf("state_json must be valid JSON")
	}
	return s.repo.SetState(ctx, pid, string(state))
}

func (s *Service) ListRules(ctx context.Context, repoKey *string, scopePrefix string, enabledOnly bool, limit, offset int) ([]Rule, error) {
	var pid *uuid.UUID
	if repoKey != nil && strings.TrimSpace(*repoKey) != "" {
		p, err := s.resolveProject(ctx, *repoKey)
		if err != nil {
			return nil, err
		}
		pid = &p
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListRules(ctx, pid, strings.TrimSpace(scopePrefix), enabledOnly, limit, offset)
}

func (s *Service) UpsertRule(ctx context.Context, repoKey *string, ruleID *uuid.UUID, scope string, priority int, title string, content string, enabled bool) (uuid.UUID, error) {
	var pid *uuid.UUID
	if repoKey != nil && strings.TrimSpace(*repoKey) != "" {
		p, err := s.resolveProject(ctx, *repoKey)
		if err != nil {
			return uuid.UUID{}, err
		}
		pid = &p
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return uuid.UUID{}, fmt.Errorf("scope cannot be empty")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return uuid.UUID{}, fmt.Errorf("title cannot be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return uuid.UUID{}, fmt.Errorf("content cannot be empty")
	}
	return s.repo.UpsertRule(ctx, pid, ruleID, scope, priority, title, content, enabled)
}

func (s *Service) SetRuleEnabled(ctx context.Context, repoKey *string, ruleID uuid.UUID, enabled bool) (bool, error) {
	var pid *uuid.UUID
	if repoKey != nil && strings.TrimSpace(*repoKey) != "" {
		p, err := s.resolveProject(ctx, *repoKey)
		if err != nil {
			return false, err
		}
		pid = &p
	}
	return s.repo.SetRuleEnabled(ctx, pid, ruleID, enabled)
}

func (s *Service) ListVersions(ctx context.Context, repoKey, docKey string, limit, offset int) ([]DocumentVersionItem, bool, error) {
	pid, err := s.resolveProject(ctx, repoKey)
	if err != nil {
		return nil, false, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListVersions(ctx, pid, strings.TrimSpace(docKey), limit, offset)
}

func (s *Service) GetDocumentVersion(ctx context.Context, repoKey, docKey string, version int) (Document, bool, error) {
	pid, err := s.resolveProject(ctx, repoKey)
	if err != nil {
		return Document{}, false, err
	}
	if version <= 0 {
		return Document{}, false, fmt.Errorf("version must be >= 1")
	}
	return s.repo.GetDocumentVersion(ctx, pid, strings.TrimSpace(docKey), version)
}

type RulesPreview struct {
	Scopes   []string `json:"scopes"`
	Rules    []Rule   `json:"rules"`
	RulesPack string  `json:"rules_pack"`
}

func (s *Service) RulesApplyPreview(ctx context.Context, repoKey *string, scopes []string) (RulesPreview, error) {
	scopePrefix := "" // we will filter in-memory for exact/prefix
	rules, err := s.ListRules(ctx, repoKey, scopePrefix, true, 200, 0)
	if err != nil {
		return RulesPreview{}, err
	}

	want := make([]string, 0, len(scopes))
	for _, sc := range scopes {
		sc = strings.TrimSpace(sc)
		if sc != "" {
			want = append(want, sc)
		}
	}
	if len(want) == 0 {
		return RulesPreview{Scopes: scopes, Rules: nil, RulesPack: ""}, nil
	}

	var picked []Rule
	for _, r := range rules {
		for _, sc := range want {
			// Treat provided scope as prefix for convenience (phase:PLAN matches phase:PLAN:sub)
			if strings.HasPrefix(r.Scope, sc) {
				picked = append(picked, r)
				break
			}
		}
	}

	var b strings.Builder
	for i, r := range picked {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(r.Title)
		b.WriteString("\n")
		b.WriteString(r.Content)
	}

	return RulesPreview{
		Scopes:   want,
		Rules:    picked,
		RulesPack: b.String(),
	}, nil
}

