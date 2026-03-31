package mcpserver

import (
	"mcp-vault-bridge/internal/app"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func New() *server.MCPServer {
	return server.NewMCPServer(
		"Vault Bridge",
		"0.2.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)
}

func RegisterTools(s *server.MCPServer, a *app.App) {
	// tasks
	s.AddTool(
		mcp.NewTool(
			"get_next_task",
			mcp.WithDescription("Fetch the highest priority todo task and lock it by moving it to in_progress."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
		),
		a.TasksTools.GetNextTask,
	)
	s.AddTool(
		mcp.NewTool(
			"complete_task",
			mcp.WithDescription("Mark an in_progress task as done and append an execution report to its description."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task UUID")),
			mcp.WithString("report", mcp.Required(), mcp.Description("Execution report to append")),
		),
		a.TasksTools.CompleteTask,
	)
	s.AddTool(
		mcp.NewTool(
			"add_context_file",
			mcp.WithDescription("Link a file path to a task (idempotent)."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task UUID")),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("File path to associate with the task")),
		),
		a.TasksTools.AddContextFile,
	)
	s.AddTool(
		mcp.NewTool(
			"list_tasks",
			mcp.WithDescription("List tasks with optional filters (status/requirement_id/epic_id) and pagination."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("status", mcp.Description("Optional: todo|in_progress|done")),
			mcp.WithString("requirement_id", mcp.Description("Optional: requirement UUID")),
			mcp.WithString("epic_id", mcp.Description("Optional: epic UUID")),
			mcp.WithNumber("limit", mcp.Description("Optional: default 50, max 200")),
			mcp.WithNumber("offset", mcp.Description("Optional: default 0")),
			mcp.WithString("order", mcp.Description("Optional: priority|created_at (default priority)")),
			mcp.WithBoolean("include_files", mcp.Description("Optional: include task_files in response")),
		),
		a.TasksTools.ListTasks,
	)
	s.AddTool(
		mcp.NewTool(
			"get_task",
			mcp.WithDescription("Get a single task by id, including files and GitHub link if present."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task UUID")),
		),
		a.TasksTools.GetTask,
	)

	// epics
	s.AddTool(
		mcp.NewTool(
			"create_epic",
			mcp.WithDescription("Create an epic."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Epic title")),
			mcp.WithString("description", mcp.Description("Optional epic description")),
		),
		a.EpicsTools.CreateEpic,
	)
	s.AddTool(
		mcp.NewTool(
			"list_epics",
			mcp.WithDescription("List epics with optional status filter and pagination."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("status", mcp.Description("Optional: open|done|archived")),
			mcp.WithNumber("limit", mcp.Description("Optional: default 50, max 200")),
			mcp.WithNumber("offset", mcp.Description("Optional: default 0")),
		),
		a.EpicsTools.ListEpics,
	)
	s.AddTool(
		mcp.NewTool(
			"link_requirement_to_epic",
			mcp.WithDescription("Link a requirement to an epic by setting requirements.epic_id."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("requirement_id", mcp.Required(), mcp.Description("Requirement UUID")),
			mcp.WithString("epic_id", mcp.Required(), mcp.Description("Epic UUID")),
		),
		a.EpicsTools.LinkRequirementToEpic,
	)
	s.AddTool(
		mcp.NewTool(
			"link_task_to_epic",
			mcp.WithDescription("Link a task to an epic by setting tasks.epic_id."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task UUID")),
			mcp.WithString("epic_id", mcp.Required(), mcp.Description("Epic UUID")),
		),
		a.EpicsTools.LinkTaskToEpic,
	)

	// github issues
	s.AddTool(
		mcp.NewTool(
			"github_get_issue_link",
			mcp.WithDescription("Get a stored GitHub issue link for an entity from Postgres (no GitHub API call)."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("entity_type", mcp.Required(), mcp.Description("epic|requirement|task")),
			mcp.WithString("entity_id", mcp.Required(), mcp.Description("Entity UUID")),
		),
		a.GHTools.GetIssueLink,
	)
	s.AddTool(
		mcp.NewTool(
			"github_create_issue_for_task",
			mcp.WithDescription("Create a GitHub issue for a task using GitHub App auth, and store the link in Postgres."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task UUID")),
			mcp.WithString("repo_owner", mcp.Required(), mcp.Description("GitHub repository owner/org")),
			mcp.WithString("repo_name", mcp.Required(), mcp.Description("GitHub repository name")),
			mcp.WithString("title_override", mcp.Description("Optional: override issue title")),
			mcp.WithString("body_mode", mcp.Description("Optional: from_task_description|minimal (default from_task_description)")),
		),
		a.GHTools.CreateIssueForTask,
	)

	// knowledge base (pgvector)
	s.AddTool(
		mcp.NewTool(
			"kb_upsert_document_chunks",
			mcp.WithDescription("Upsert a document and its embedded chunks into Postgres pgvector KB."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithString("source_type", mcp.Description("Optional: markdown|decision|file (default markdown)")),
			mcp.WithString("source_path", mcp.Required(), mcp.Description("Source path or identifier (e.g. README.md)")),
			mcp.WithString("title", mcp.Description("Optional: document title")),
			mcp.WithString("full_text", mcp.Required(), mcp.Description("Full raw text used for hashing/dedup")),
			mcp.WithArray("chunks", mcp.Description("Chunks with embeddings: [{chunk_index, content, embedding, metadata}]")),
		),
		a.KBTools.UpsertDocumentChunks,
	)

	s.AddTool(
		mcp.NewTool(
			"kb_search_context",
			mcp.WithDescription("Semantic search over stored chunks using a query embedding vector."),
			mcp.WithString("repo_key", mcp.Description("Optional: owner/repo (defaults to DEFAULT_REPO_KEY env var)")),
			mcp.WithArray("query_embedding", mcp.Description("float32 embedding vector"), mcp.WithNumberItems()),
			mcp.WithNumber("top_k", mcp.Description("Optional: default 8, max 50")),
		),
		a.KBTools.SearchContext,
	)

	s.AddTool(
		mcp.NewTool(
			"kb_chunk_markdown",
			mcp.WithDescription("Chunk Markdown into retrieval-friendly sections (headings-aware), returning chunks + metadata. No embeddings generated."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Markdown text to chunk")),
			mcp.WithNumber("max_chars", mcp.Description("Optional: default 3000")),
			mcp.WithNumber("overlap_chars", mcp.Description("Optional: default 300")),
		),
		a.KBTools.ChunkMarkdown,
	)
}

