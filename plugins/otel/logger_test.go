package otel_test

import (
	"log/slog"
	"testing"

	"github.com/Authula/authula/internal/bootstrap"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	otel "github.com/Authula/authula/plugins/otel"
	oteltests "github.com/Authula/authula/plugins/otel/tests"
	"github.com/Authula/authula/plugins/otel/types"
)

func newPluginContext(logger models.Logger) *models.PluginContext {
	return &models.PluginContext{
		Logger: logger,
		GetConfig: func() *models.Config {
			return &models.Config{Logger: models.LoggerConfig{Level: "info"}}
		},
	}
}

func newLoggingPlugin() *otel.OTelPlugin {
	return otel.New(types.OTelPluginConfig{
		Enabled:        true,
		Exporter:       types.ExporterOTLP,
		LogsEnabled:    true,
		TracesEnabled:  new(false),
		MetricsEnabled: new(false),
	})
}

func TestLogsDisabledAttachesNothing(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	logger := bootstrap.InitLogger(bootstrap.LoggerOptions{Level: "info"})
	plugin := otel.New(types.OTelPluginConfig{
		Enabled:        true,
		Exporter:       types.ExporterOTLP,
		LogsEnabled:    false,
		TracesEnabled:  new(false),
		MetricsEnabled: new(false),
	})

	if err := plugin.Init(newPluginContext(logger)); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	t.Cleanup(func() { _ = plugin.Close() })

	if provider := otel.LogProviderForTest(plugin); provider != nil {
		t.Fatal("log provider created while logs_enabled=false")
	}
}

func TestLogsBridgeReceivesRecords(t *testing.T) {
	tests := []struct {
		name string
		// build returns the logger to attach and an optional base logger whose
		// records prove the wrapper still forwards.
		build func(t *testing.T) (models.Logger, *internaltests.LevelRecordingLogger, func())
	}{
		{
			name: "default logger",
			build: func(t *testing.T) (models.Logger, *internaltests.LevelRecordingLogger, func()) {
				original := slog.Default()
				return bootstrap.InitLogger(bootstrap.LoggerOptions{Level: "info"}),
					nil,
					func() { slog.SetDefault(original) }
			},
		},
		{
			name: "custom logger wrapped by InitLogger",
			build: func(t *testing.T) (models.Logger, *internaltests.LevelRecordingLogger, func()) {
				original := slog.Default()
				base := new(internaltests.LevelRecordingLogger)
				return bootstrap.InitLogger(bootstrap.LoggerOptions{Level: "info", Logger: base}),
					base,
					func() { slog.SetDefault(original) }
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger, base, cleanup := tc.build(t)
			t.Cleanup(cleanup)

			exporter := new(oteltests.Exporter)
			plugin := newLoggingPlugin()
			otel.AttachLogsForTest(plugin, logger, exporter)

			if provider := otel.LogProviderForTest(plugin); provider == nil {
				t.Fatal("log provider was not created")
			}

			logger.Info("hello", "key", "value")

			if err := plugin.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}

			records := exporter.Records()
			if len(records) != 1 {
				t.Fatalf("exported %d records, want 1", len(records))
			}
			if got := records[0].Body().AsString(); got != "hello" {
				t.Fatalf("exported body = %q, want %q", got, "hello")
			}

			if base != nil {
				if got := base.Messages(); len(got) != 1 || got[0] != "hello" {
					t.Fatalf("base messages = %v, want [hello]", got)
				}
			}

			if err := plugin.Close(); err != nil {
				t.Fatalf("second Close() error = %v", err)
			}
		})
	}
}
