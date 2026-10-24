package bootstrap

import (
	"log/slog"
	"os"
	"time"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/models"
	"github.com/lmittmann/tint"
)

// LoggerOptions configures logger initialization
type LoggerOptions struct {
	Level  string
	Logger models.Logger
}

// InitLogger creates a configured logger instance. It always returns a
// fan-out: a user-supplied logger is wrapped rather than replaced, so plugins
// can attach sinks in either case.
func InitLogger(opts LoggerOptions) models.Logger {
	base := opts.Logger
	var slogBase *slog.Logger

	if base == nil {
		level := parseSlogLevel(opts.Level)

		if os.Getenv(env.EnvGoEnvironment) != "production" {
			slogBase = slog.New(tint.NewTextHandler(os.Stderr, &tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.Kitchen,
			}))
		} else {
			slogBase = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: level,
			}))
		}
		base = slogBase
	}

	fanout := newFanoutLogger(base)
	if slogBase != nil {
		// Compatibility only: third-party libraries calling slog.* directly reach
		// stderr, but bypass the fan-out and therefore any attached sinks.
		slog.SetDefault(slogBase)
	}

	return fanout
}

// parseSlogLevel maps a configured level name to a slog level. Unknown or empty
// values fall back to info.
func parseSlogLevel(name string) slog.Level {
	switch name {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
