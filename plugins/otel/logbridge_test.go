package otel

import (
	"context"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/Authula/authula/models"
	oteltests "github.com/Authula/authula/plugins/otel/tests"
)

func newTestBridge(t *testing.T, min slog.Level) (models.Logger, *oteltests.Exporter) {
	t.Helper()

	exporter := new(oteltests.Exporter)
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
	)
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown provider: %v", err)
		}
	})

	return newBridgeLogger(provider, "1.2.3", min), exporter
}

func TestBridgeLoggerEmitsRecord(t *testing.T) {
	t.Parallel()

	logger, exporter := newTestBridge(t, slog.LevelDebug)
	logger.Info("hello", "key", "value", "count", 3)

	records := exporter.Records()
	if len(records) != 1 {
		t.Fatalf("exported %d records, want 1", len(records))
	}

	record := records[0]
	if got := record.Body().AsString(); got != "hello" {
		t.Fatalf("body = %q, want %q", got, "hello")
	}
	if got := record.Severity(); got != otellog.SeverityInfo {
		t.Fatalf("severity = %v, want %v", got, otellog.SeverityInfo)
	}

	scope := record.InstrumentationScope()
	if scope.Name != scopeName {
		t.Fatalf("scope name = %q, want %q", scope.Name, scopeName)
	}
	if scope.Version != "1.2.3" {
		t.Fatalf("scope version = %q, want %q", scope.Version, "1.2.3")
	}

	attributes := map[string]string{}
	record.WalkAttributes(func(kv attribute.KeyValue) bool {
		attributes[string(kv.Key)] = kv.Value.String()
		return true
	})
	if attributes["key"] != "value" {
		t.Fatalf("key attribute = %q, want %q", attributes["key"], "value")
	}
	if attributes["count"] != "3" {
		t.Fatalf("count attribute = %q, want %q", attributes["count"], "3")
	}
}

func TestBridgeLoggerLevelFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level slog.Level
		want  bool
	}{
		{name: "debug", level: slog.LevelDebug, want: false},
		{name: "info", level: slog.LevelInfo, want: true},
		{name: "warn", level: slog.LevelWarn, want: true},
		{name: "error", level: slog.LevelError, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exporter := new(oteltests.Exporter)
			provider := sdklog.NewLoggerProvider(
				sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
			)
			t.Cleanup(func() {
				_ = provider.Shutdown(context.Background())
			})

			logger := newBridgeLogger(provider, "1.0.0", slog.LevelInfo)
			switch tc.level {
			case slog.LevelDebug:
				logger.Debug("message")
			case slog.LevelInfo:
				logger.Info("message")
			case slog.LevelWarn:
				logger.Warn("message")
			case slog.LevelError:
				logger.Error("message")
			}

			got := len(exporter.Records()) == 1
			if got != tc.want {
				t.Fatalf("exported = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
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

			if got := parseLevel(tc.name); got != tc.want {
				t.Fatalf("parseLevel(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
