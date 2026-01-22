package bootstrap

import (
	"log/slog"
	"os"
	"time"

	"github.com/GoBetterAuth/go-better-auth/env"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/lmittmann/tint"
)

// LoggerOptions configures logger initialization
type LoggerOptions struct {
	Level string
}

// InitLogger creates a configured logger instance
func InitLogger(opts LoggerOptions) models.Logger {
	environment := os.Getenv(env.EnvGoEnvironment)
	var logger *slog.Logger

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

	if environment != "production" {
		logger = slog.New(tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		}))
	}

	slog.SetDefault(logger)

	return logger
}
