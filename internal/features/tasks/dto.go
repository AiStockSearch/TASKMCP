package tasks

type NextTaskResponse struct {
	TaskID      string   `json:"task_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	FilePaths   []string `json:"file_paths"`
}

type TaskDTO struct {
	ID            string   `json:"id"`
	RequirementID *string  `json:"requirement_id,omitempty"`
	EpicID        *string  `json:"epic_id,omitempty"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
	Priority      int      `json:"priority"`
	FilePaths     []string `json:"file_paths,omitempty"`
}

