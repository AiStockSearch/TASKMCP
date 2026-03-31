package logging

import (
	"log/slog"
	"os"
)

func SetupDefault() *slog.Logger {
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(l)
	return l
}

