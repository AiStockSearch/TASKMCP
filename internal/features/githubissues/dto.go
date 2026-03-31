package githubissues

type LinkDTO struct {
	EntityType  string `json:"entity_type"`
	EntityID    string `json:"entity_id"`
	RepoOwner   string `json:"repo_owner"`
	RepoName    string `json:"repo_name"`
	IssueNumber int    `json:"issue_number"`
	IssueURL    string `json:"issue_url"`
}

