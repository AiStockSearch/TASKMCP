package epics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repo
	projects ProjectsResolver
}

type ProjectsResolver interface {
	Resolve(ctx context.Context, repoKey string) (uuid.UUID, error)
}

func NewService(repo *Repo, projects ProjectsResolver) *Service {
	return &Service{repo: repo, projects: projects}
}

func (s *Service) CreateEpic(ctx context.Context, repoKey string, title, desc string) (uuid.UUID, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return uuid.UUID{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return uuid.UUID{}, fmt.Errorf("title cannot be empty")
	}
	id := uuid.New()
	if err := s.repo.Create(ctx, projectID, id, title, strings.TrimSpace(desc)); err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

func (s *Service) ListEpics(ctx context.Context, repoKey string, status string, limit, offset int) ([]EpicDTO, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, projectID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]EpicDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, EpicDTO{
			ID:          r.ID.String(),
			Title:       r.Title,
			Description: r.Description,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func (s *Service) LinkRequirementToEpic(ctx context.Context, repoKey string, requirementID, epicID uuid.UUID) (bool, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return false, err
	}
	return s.repo.LinkRequirement(ctx, projectID, requirementID, epicID)
}

func (s *Service) LinkTaskToEpic(ctx context.Context, repoKey string, taskID, epicID uuid.UUID) (bool, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return false, err
	}
	return s.repo.LinkTask(ctx, projectID, taskID, epicID)
}

