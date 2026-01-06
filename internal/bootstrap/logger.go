package bootstrap

import (
	"log/slog"
	"os"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// LoggerOptions configures logger initialization
type LoggerOptions struct {
	Level string
}

// InitLogger creates a configured logger instance
func InitLogger(opts LoggerOptions) models.Logger {
	level := slog.LevelInfo
	switch opts.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	return logger
}
