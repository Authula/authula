package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/otel/types"
)

// fanoutLogger is a models.Logger that also supports models.LoggerFanout so the
// sink attachment can be asserted without touching the global slog logger.
type fanoutLogger struct {
	sinks []models.Logger
}

func (l *fanoutLogger) Debug(string, ...any) {}
func (l *fanoutLogger) Info(string, ...any)  {}
func (l *fanoutLogger) Warn(string, ...any)  {}
func (l *fanoutLogger) Error(string, ...any) {}

func (l *fanoutLogger) AddLogger(sink models.Logger) func() {
	l.sinks = append(l.sinks, sink)

	return func() {
		for i, attached := range l.sinks {
			if attached == sink {
				l.sinks = append(l.sinks[:i], l.sinks[i+1:]...)
				return
			}
		}
	}
}

func TestInitLoggerInactiveSkipsExporter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		exporter types.ExporterType
	}{
		{name: "logs disabled", enabled: false, exporter: types.ExporterOTLP},
		{name: "exporter none", enabled: true, exporter: types.ExporterNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := &OTelPlugin{pluginConfig: types.OTelPluginConfig{
				Exporter:    tt.exporter,
				LogsEnabled: tt.enabled,
			}}

			require.NoError(t, plugin.initLogger(nil))
			assert.Nil(t, plugin.loggerProvider)
			assert.Nil(t, plugin.detachLogSink)
		})
	}
}

func TestInitLoggerReturnsExporterError(t *testing.T) {
	t.Parallel()

	plugin := &OTelPlugin{pluginConfig: types.OTelPluginConfig{
		Exporter:    "jaeger",
		LogsEnabled: true,
	}}

	err := plugin.initLogger(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported log exporter")
	assert.Nil(t, plugin.loggerProvider)
	assert.Nil(t, plugin.detachLogSink)
}

func TestInitLoggerAttachesFanoutSink(t *testing.T) {
	t.Parallel()

	logger := new(fanoutLogger)
	plugin := &OTelPlugin{
		pluginConfig: types.OTelPluginConfig{
			Exporter:    types.ExporterStdout,
			LogsEnabled: true,
		},
		logger: logger,
	}

	require.NoError(t, plugin.initLogger(nil))
	require.NotNil(t, plugin.loggerProvider)
	t.Cleanup(func() { shutdownWithTimeout(t, plugin.loggerProvider.Shutdown) })

	require.Len(t, logger.sinks, 1)
	require.NotNil(t, plugin.detachLogSink)

	plugin.detachLogSink()
	assert.Empty(t, logger.sinks)
}

func TestInitLoggerWithExporterWarnsWithoutFanout(t *testing.T) {
	t.Parallel()

	logger := new(internaltests.LevelRecordingLogger)
	plugin := &OTelPlugin{logger: logger}

	plugin.initLoggerWithExporter(nil, stubLogExporter{})
	require.NotNil(t, plugin.loggerProvider)
	t.Cleanup(func() { shutdownWithTimeout(t, plugin.loggerProvider.Shutdown) })

	assert.Nil(t, plugin.detachLogSink)

	records := logger.Records()
	require.Len(t, records, 1)
	assert.Equal(t, "warn", records[0].Level)
	assert.Contains(t, records[0].Message, "log bridge not attached")
}

func TestLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pluginCtx *models.PluginContext
		want      string
	}{
		{name: "nil plugin context", pluginCtx: nil, want: ""},
		{name: "nil get config", pluginCtx: &models.PluginContext{}, want: ""},
		{
			name: "nil config",
			pluginCtx: &models.PluginContext{
				GetConfig: func() *models.Config { return nil },
			},
			want: "",
		},
		{
			name: "configured level",
			pluginCtx: &models.PluginContext{
				GetConfig: func() *models.Config {
					return &models.Config{Logger: models.LoggerConfig{Level: "debug"}}
				},
			},
			want: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := &OTelPlugin{pluginCtx: tt.pluginCtx}
			assert.Equal(t, tt.want, plugin.logLevel())
		})
	}
}
