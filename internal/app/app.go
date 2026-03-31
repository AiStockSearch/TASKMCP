package app

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"time"

	"mcp-vault-bridge/internal/features/epics"
	"mcp-vault-bridge/internal/features/githubissues"
	"mcp-vault-bridge/internal/features/kb"
	"mcp-vault-bridge/internal/features/projects"
	"mcp-vault-bridge/internal/features/tasks"
	"mcp-vault-bridge/internal/storage/postgres"
)

type App struct {
	DB       *sql.DB
	DBGuard  *postgres.DBGuard
	HTTP     *http.Client
	Projects *projects.Resolver

	TasksTools *tasks.Tools
	EpicsTools *epics.Tools
	GHTools    *githubissues.Tools
	KBTools    *kb.Tools
}

func New(db *sql.DB) *App {
	guard := postgres.NewDBGuard(db)

	httpClient := &http.Client{Timeout: 15 * time.Second}

	projectResolver := projects.NewResolver(db)

	tasksRepo := tasks.NewRepo(db)
	tasksSvc := tasks.NewService(tasksRepo, projectResolver)

	epicsRepo := epics.NewRepo(db)
	epicsSvc := epics.NewService(epicsRepo, projectResolver)

	ghRepo := githubissues.NewRepo(db)
	ghApp := githubissues.NewGitHubAppFromEnv()
	ghSvc := githubissues.NewService(ghRepo, tasksSvc, projectResolver, ghApp, httpClient)

	kbRepo := kb.NewRepo(db)
	kbSvc := kb.NewService(kbRepo, projectResolver)
	if s := os.Getenv("KB_EMBEDDING_DIM"); s != "" {
		if dim, err := strconv.Atoi(s); err == nil && dim > 0 {
			kbSvc.SetEmbeddingDim(dim)
		}
	}

	return &App{
		DB:        db,
		DBGuard:   guard,
		HTTP:      httpClient,
		Projects:  projectResolver,
		TasksTools: tasks.NewTools(tasksSvc, guard),
		EpicsTools: epics.NewTools(epicsSvc, guard),
		GHTools:    githubissues.NewTools(ghSvc, guard),
		KBTools:    kb.NewTools(kbSvc, guard),
	}
}

