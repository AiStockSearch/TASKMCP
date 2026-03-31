package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mcp-vault-bridge/internal/app"
	"mcp-vault-bridge/internal/mcpserver"
	"mcp-vault-bridge/internal/platform/env"
	"mcp-vault-bridge/internal/platform/logging"
	"mcp-vault-bridge/internal/storage/postgres"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	logging.SetupDefault()
	env.LoadDotEnv()

	dsn, err := env.Required("DATABASE_URL")
	if err != nil {
		os.Exit(1)
	}

	db, err := postgres.Open(dsn)
	if err != nil {
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	// Don't hard-fail on initial DB connectivity: Cursor expects the MCP server to stay up.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = db.PingContext(ctx)
		cancel()
	}

	a := app.New(db)
	s := mcpserver.New()
	mcpserver.RegisterTools(s, a)

	// Cursor launches MCP servers as subprocesses and communicates over stdio.
	// Build: go build -o mcp-vault-bridge .
	// Cursor: Settings -> Features -> MCP -> Command -> /absolute/path/to/mcp-vault-bridge
	// Ensure DATABASE_URL is set (via .env in working dir or environment variables).

	go func() {
		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		_ = db.Close()
		os.Exit(0)
	}()

	if err := server.ServeStdio(s); err != nil {
		os.Exit(1)
	}
}

