package epics

type EpicDTO struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type EpicTaskDTO struct {
	ID            string   `json:"id"`
	RequirementID *string  `json:"requirement_id,omitempty"`
	EpicID        *string  `json:"epic_id,omitempty"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
	Priority      int      `json:"priority"`
	FilePaths     []string `json:"file_paths,omitempty"`
}

