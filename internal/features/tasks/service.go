package tasks

import (
	"context"
	"fmt"
	"strings"

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

func (s *Service) GetNextTask(ctx context.Context, repoKey string) (NextTaskResponse, bool, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return NextTaskResponse{}, false, err
	}

	row, paths, ok, err := s.repo.GetNextTodoAndLock(ctx, projectID)
	if err != nil {
		return NextTaskResponse{}, false, err
	}
	if !ok {
		return NextTaskResponse{}, false, nil
	}
	return NextTaskResponse{
		TaskID:      row.ID.String(),
		Title:       row.Title,
		Description: row.Description,
		FilePaths:   paths,
	}, true, nil
}

func (s *Service) CompleteTask(ctx context.Context, repoKey string, taskID uuid.UUID, report string) (bool, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return false, err
	}
	appendix := "\n\n### Execution Report:\n" + report
	return s.repo.CompleteInProgress(ctx, projectID, taskID, appendix)
}

func (s *Service) AddContextFile(ctx context.Context, repoKey string, taskID uuid.UUID, filePath string) error {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return err
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return fmt.Errorf("file_path cannot be empty")
	}
	return s.repo.AddContextFile(ctx, projectID, taskID, filePath, uuid.New())
}

type ListInput struct {
	Status        string
	RequirementID *uuid.UUID
	EpicID        *uuid.UUID
	Limit         int
	Offset        int
	Order         string
	IncludeFiles  bool
}

func (s *Service) ListTasks(ctx context.Context, repoKey string, in ListInput) ([]TaskDTO, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, ListOptions{
		ProjectID:     projectID,
		Status:        in.Status,
		RequirementID: in.RequirementID,
		EpicID:        in.EpicID,
		Limit:         in.Limit,
		Offset:        in.Offset,
		Order:         in.Order,
	})
	if err != nil {
		return nil, err
	}

	var ids []uuid.UUID
	out := make([]TaskDTO, 0, len(rows))
	for _, r := range rows {
		dto := TaskDTO{
			ID:          r.ID.String(),
			Title:       r.Title,
			Description: r.Description,
			Status:      r.Status,
			Priority:    r.Priority,
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
		ids = append(ids, r.ID)
	}

	if in.IncludeFiles && len(ids) > 0 {
		filesByTask, err := s.repo.FilesByTaskIDs(ctx, projectID, ids)
		if err != nil {
			return nil, err
		}
		for i := range out {
			id, err := uuid.Parse(out[i].ID)
			if err != nil {
				continue
			}
			out[i].FilePaths = filesByTask[id]
		}
	}

	return out, nil
}

type GetTaskResponse struct {
	Task TaskDTO `json:"task"`
}

func (s *Service) GetTask(ctx context.Context, repoKey string, taskID uuid.UUID) (TaskDTO, bool, error) {
	projectID, err := s.projects.Resolve(ctx, repoKey)
	if err != nil {
		return TaskDTO{}, false, err
	}

	row, ok, err := s.repo.Get(ctx, projectID, taskID)
	if err != nil {
		return TaskDTO{}, false, err
	}
	if !ok {
		return TaskDTO{}, false, nil
	}

	dto := TaskDTO{
		ID:          row.ID.String(),
		Title:       row.Title,
		Description: row.Description,
		Status:      row.Status,
		Priority:    row.Priority,
	}
	if row.RequirementID != nil {
		s := row.RequirementID.String()
		dto.RequirementID = &s
	}
	if row.EpicID != nil {
		s := row.EpicID.String()
		dto.EpicID = &s
	}

	filesByTask, err := s.repo.FilesByTaskIDs(ctx, projectID, []uuid.UUID{taskID})
	if err != nil {
		return TaskDTO{}, false, err
	}
	dto.FilePaths = filesByTask[taskID]

	return dto, true, nil
}

