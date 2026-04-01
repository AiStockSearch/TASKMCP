package tasks

import "encoding/json"

type NextTaskResponse struct {
	TaskID             string          `json:"task_id"`
	Title              string          `json:"title"`
	Description        string          `json:"description"`
	FilePaths          []string        `json:"file_paths"`
	RequirementID      *string         `json:"requirement_id,omitempty"`
	RequirementTitle   *string         `json:"requirement_title,omitempty"`
	SpecJSON           json.RawMessage `json:"spec_json,omitempty"`
}

type TaskDTO struct {
	ID                string          `json:"id"`
	RequirementID     *string         `json:"requirement_id,omitempty"`
	RequirementTitle  *string         `json:"requirement_title,omitempty"`
	SpecJSON          json.RawMessage `json:"spec_json,omitempty"`
	EpicID            *string         `json:"epic_id,omitempty"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	Status            string          `json:"status"`
	Priority          int             `json:"priority"`
	FilePaths         []string        `json:"file_paths,omitempty"`
}

