package githubissues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"mcp-vault-bridge/internal/features/tasks"
)

type Service struct {
	repo       *Repo
	tasksSvc   *tasks.Service
	projects   ProjectsResolver
	ghApp      *GitHubApp
	httpClient *http.Client
}

type ProjectsResolver interface {
	Resolve(ctx context.Context, repoKey string) (uuid.UUID, error)
}

func NewService(repo *Repo, tasksSvc *tasks.Service, projects ProjectsResolver, ghApp *GitHubApp, httpClient *http.Client) *Service {
	return &Service{repo: repo, tasksSvc: tasksSvc, projects: projects, ghApp: ghApp, httpClient: httpClient}
}

func (s *Service) GetStoredLink(ctx context.Context, repoKey string, entityType string, entityID uuid.UUID) (*LinkDTO, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return nil, err
	}
	return s.repo.GetLink(ctx, projectID, entityType, entityID)
}

func (s *Service) CreateIssueForTask(ctx context.Context, repoKey string, taskID uuid.UUID, owner, repo, titleOverride, bodyMode string) (*LinkDTO, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return nil, err
	}

	if link, err := s.repo.GetLink(ctx, projectID, "task", taskID); err == nil && link != nil {
		return link, nil
	}

	task, ok, err := s.tasksSvc.GetTask(ctx, repoKey, taskID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("task not found")
	}

	issueTitle, body, err := buildIssuePayload(task.Title, task.Description, task.ID, titleOverride, bodyMode)
	if err != nil {
		return nil, err
	}

	token, err := s.ghApp.InstallationToken(ctx, s.httpClient)
	if err != nil {
		return nil, err
	}

	type createIssueReq struct {
		Title string `json:"title"`
		Body  string `json:"body,omitempty"`
	}
	payload, err := json.Marshal(createIssueReq{Title: issueTitle, Body: body})
	if err != nil {
		return nil, fmt.Errorf("encode issue request: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", owner, repo)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("create issue request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read issue response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("create issue failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var issueResp struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(b, &issueResp); err != nil {
		return nil, fmt.Errorf("decode issue response: %w", err)
	}
	if issueResp.Number == 0 || issueResp.HTMLURL == "" {
		return nil, fmt.Errorf("invalid issue response from GitHub")
	}

	link := LinkDTO{
		EntityType:  "task",
		EntityID:    taskID.String(),
		RepoOwner:   owner,
		RepoName:    repo,
		IssueNumber: issueResp.Number,
		IssueURL:    issueResp.HTMLURL,
	}
	if err := s.repo.InsertLinkIfAbsent(ctx, projectID, link); err != nil {
		return nil, err
	}

	return &link, nil
}

