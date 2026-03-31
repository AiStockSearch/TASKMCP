package githubissues

import (
	"fmt"
	"strings"
)

func buildIssuePayload(taskTitle, taskDescription, taskID, titleOverride, bodyMode string) (title string, body string, err error) {
	title = strings.TrimSpace(taskTitle)
	if strings.TrimSpace(titleOverride) != "" {
		title = strings.TrimSpace(titleOverride)
	}
	if title == "" {
		title = "Vault task"
	}

	if bodyMode == "" {
		bodyMode = "from_task_description"
	}

	switch bodyMode {
	case "from_task_description":
		body = strings.TrimSpace(taskDescription)
		if body != "" {
			body += "\n\n"
		}
		body += fmt.Sprintf("Vault task: %s", strings.TrimSpace(taskID))
	case "minimal":
		body = fmt.Sprintf("Vault task: %s", strings.TrimSpace(taskID))
	default:
		return "", "", fmt.Errorf("invalid body_mode (expected from_task_description|minimal)")
	}

	return title, body, nil
}

