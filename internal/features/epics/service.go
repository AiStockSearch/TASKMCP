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

type AddTasksResult struct {
	Requested      int      `json:"requested"`
	Updated        int      `json:"updated"`
	MissingTaskIDs []string `json:"missing_task_ids"`
}

func (s *Service) AddTasksToEpic(ctx context.Context, repoKey string, epicID uuid.UUID, taskIDs []uuid.UUID) (AddTasksResult, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return AddTasksResult{}, err
	}
	if len(taskIDs) == 0 {
		return AddTasksResult{Requested: 0, Updated: 0}, nil
	}

	updated, existing, err := s.repo.AddTasksToEpic(ctx, projectID, epicID, taskIDs)
	if err != nil {
		return AddTasksResult{}, err
	}

	existsSet := make(map[uuid.UUID]struct{}, len(existing))
	for _, id := range existing {
		existsSet[id] = struct{}{}
	}
	var missing []string
	for _, id := range taskIDs {
		if _, ok := existsSet[id]; !ok {
			missing = append(missing, id.String())
		}
	}

	return AddTasksResult{
		Requested:      len(taskIDs),
		Updated:        int(updated),
		MissingTaskIDs: missing,
	}, nil
}

func (s *Service) ListEpicTasks(ctx context.Context, repoKey string, epicID uuid.UUID, includeFiles bool) ([]EpicTaskDTO, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListEpicTasks(ctx, projectID, epicID, includeFiles)
	if err != nil {
		return nil, err
	}
	out := make([]EpicTaskDTO, 0, len(rows))
	for _, r := range rows {
		dto := EpicTaskDTO{
			ID:          r.ID.String(),
			Title:       r.Title,
			Description: r.Description,
			Status:      r.Status,
			Priority:    r.Priority,
			FilePaths:   r.FilePaths,
		}
		if r.RequirementID != nil {
			s := r.RequirementID.String()
			dto.RequirementID = &s
		}
		if r.EpicID != nil {
			s := r.EpicID.String()
			dto.EpicID = &s
		}
		out = append(out, dto)
	}
	return out, nil
}

