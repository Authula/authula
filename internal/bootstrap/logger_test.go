package bootstrap

import (
	"log/slog"
	"testing"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func TestInitLoggerWrapsConfiguredLogger(t *testing.T) {
	// This test inspects the process-global slog default, so it must not run in
	// parallel with other tests that mutate it.
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	base := new(internaltests.LevelRecordingLogger)
	sink := new(internaltests.LevelRecordingLogger)

	logger := InitLogger(LoggerOptions{Level: "debug", Logger: base})
	if logger == base {
		t.Fatal("expected InitLogger to return a fan-out wrapper, not the supplied logger")
	}
	if slog.Default() != original {
		t.Fatal("InitLogger must not call slog.SetDefault for a custom logger")
	}

	fanout, ok := logger.(models.LoggerFanout)
	if !ok {
		t.Fatalf("InitLogger returned %T, want models.LoggerFanout", logger)
	}

	fanout.AddLogger(sink)
	logger.Info("hello", "key", "value")

	if got := base.Messages(); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("base messages = %v, want [hello]", got)
	}
	if got := sink.Messages(); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("sink messages = %v, want [hello]", got)
	}
}

func TestInitLoggerDefaultReturnsFanout(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	logger := InitLogger(LoggerOptions{Level: "debug"})

	fanout, ok := logger.(models.LoggerFanout)
	if !ok {
		t.Fatalf("InitLogger returned %T, want models.LoggerFanout", logger)
	}

	sink := new(internaltests.LevelRecordingLogger)
	fanout.AddLogger(sink)
	logger.Debug("to sink")

	if got := sink.Messages(); len(got) != 1 || got[0] != "to sink" {
		t.Fatalf("sink messages = %v, want [to sink]", got)
	}
}

func TestParseSlogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want slog.Level
	}{
		{name: "debug", want: slog.LevelDebug},
		{name: "info", want: slog.LevelInfo},
		{name: "warn", want: slog.LevelWarn},
		{name: "error", want: slog.LevelError},
		{name: "", want: slog.LevelInfo},
		{name: "trace", want: slog.LevelInfo},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := parseSlogLevel(tc.name); got != tc.want {
				t.Fatalf("parseSlogLevel(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
