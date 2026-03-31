package githubissues

import "testing"

func TestBuildIssuePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		taskTitle     string
		taskDesc      string
		taskID        string
		titleOverride string
		bodyMode      string
		wantTitle     string
		wantBody      string
		wantErr       bool
	}{
		{
			name:      "from_description_default",
			taskTitle: "Fix auth",
			taskDesc:  "Do X then Y",
			taskID:    "00000000-0000-0000-0000-000000000001",
			bodyMode:  "",
			wantTitle: "Fix auth",
			wantBody:  "Do X then Y\n\nVault task: 00000000-0000-0000-0000-000000000001",
		},
		{
			name:      "minimal",
			taskTitle: "Refactor",
			taskDesc:  "Long text",
			taskID:    "t-123",
			bodyMode:  "minimal",
			wantTitle: "Refactor",
			wantBody:  "Vault task: t-123",
		},
		{
			name:          "title_override",
			taskTitle:     "Old",
			taskDesc:      "",
			taskID:        "x",
			titleOverride: "New",
			bodyMode:      "minimal",
			wantTitle:     "New",
			wantBody:      "Vault task: x",
		},
		{
			name:      "invalid_body_mode",
			taskTitle: "A",
			taskID:    "x",
			bodyMode:  "wat",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotTitle, gotBody, err := buildIssuePayload(tt.taskTitle, tt.taskDesc, tt.taskID, tt.titleOverride, tt.bodyMode)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotTitle != tt.wantTitle {
				t.Fatalf("title mismatch: got %q want %q", gotTitle, tt.wantTitle)
			}
			if gotBody != tt.wantBody {
				t.Fatalf("body mismatch: got %q want %q", gotBody, tt.wantBody)
			}
		})
	}
}

