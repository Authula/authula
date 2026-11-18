package otel

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/otel/types"
)

func TestOTelPluginInitConfigLoadError(t *testing.T) {
	t.Parallel()

	plugin := New(types.OTelPluginConfig{})

	err := plugin.Init(&models.PluginContext{GetConfig: func() *models.Config {
		return &models.Config{Plugins: models.PluginsConfig{
			"otel": map[string]any{"enabled": "not-a-bool"},
		}}
	}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load plugin config")
}

func TestOTelPluginInitComponentErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		config       types.OTelPluginConfig
		wantErrorIs  string
		wantProvider string
	}{
		{
			name: "tracer exporter construction fails",
			config: types.OTelPluginConfig{
				Enabled:        true,
				Exporter:       types.ExporterOTLP,
				Protocol:       types.ProtocolGRPC,
				Endpoint:       "local%zzhost:4318",
				TracesEnabled:  new(true),
				MetricsEnabled: new(false),
				RuntimeMetrics: new(false),
			},
			wantErrorIs:  "invalid URL escape",
			wantProvider: "tracer",
		},
		{
			name: "meter exporter construction fails",
			config: types.OTelPluginConfig{
				Enabled:        true,
				Exporter:       types.ExporterOTLP,
				Endpoint:       "localhost:badport",
				TracesEnabled:  new(false),
				MetricsEnabled: new(true),
				RuntimeMetrics: new(false),
			},
			wantErrorIs:  "invalid port",
			wantProvider: "meter",
		},
		{
			name: "log exporter construction fails",
			config: types.OTelPluginConfig{
				Enabled:        true,
				Exporter:       types.ExporterOTLP,
				Endpoint:       "localhost:badport",
				TracesEnabled:  new(false),
				MetricsEnabled: new(false),
				RuntimeMetrics: new(false),
				LogsEnabled:    true,
			},
			wantErrorIs:  "invalid port",
			wantProvider: "logger",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tc.config)
			err := plugin.Init(&models.PluginContext{GetConfig: nilConfig})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErrorIs)

			// The failing component must not leave a half-initialized provider
			// behind; earlier components may already be wired.
			switch tc.wantProvider {
			case "tracer":
				assert.Nil(t, plugin.tracerProvider)
			case "meter":
				assert.Nil(t, plugin.meterProvider)
			case "logger":
				assert.Nil(t, plugin.loggerProvider)
			}

			require.NoError(t, plugin.Close())
		})
	}
}

func TestOTelPluginInitStartsRuntimeMetrics(t *testing.T) {
	silenceStdout(t)

	plugin := New(types.OTelPluginConfig{
		Enabled:        true,
		Exporter:       types.ExporterStdout,
		TracesEnabled:  new(false),
		MetricsEnabled: new(true),
		RuntimeMetrics: new(true),
	})

	require.NoError(t, plugin.Init(&models.PluginContext{GetConfig: nilConfig}))
	require.NotNil(t, plugin.meterProvider)
	require.NotNil(t, plugin.duration)

	require.NoError(t, plugin.Close())
}

func TestOTelPluginInitLoggerWithoutFanoutWarns(t *testing.T) {
	silenceStdout(t)

	logger := &internaltests.LevelRecordingLogger{}
	plugin := New(types.OTelPluginConfig{
		Enabled:        true,
		Exporter:       types.ExporterStdout,
		TracesEnabled:  new(false),
		MetricsEnabled: new(false),
		RuntimeMetrics: new(false),
		LogsEnabled:    true,
	})

	require.NoError(t, plugin.Init(&models.PluginContext{GetConfig: nilConfig, Logger: logger}))
	require.NotNil(t, plugin.loggerProvider)
	assert.Nil(t, plugin.detachLogSink)
	assert.Contains(t, logger.Messages(), "otel: log bridge not attached (logger does not support fan-out)")

	require.NoError(t, plugin.Close())
}

func TestOTelPluginInitLoggerAttachesAndDetachesFanout(t *testing.T) {
	silenceStdout(t)

	fanout := &testFanoutLogger{}
	plugin := New(types.OTelPluginConfig{
		Enabled:        true,
		Exporter:       types.ExporterStdout,
		TracesEnabled:  new(false),
		MetricsEnabled: new(false),
		RuntimeMetrics: new(false),
		LogsEnabled:    true,
	})

	require.NoError(t, plugin.Init(&models.PluginContext{GetConfig: nilConfig, Logger: fanout}))
	require.NotNil(t, plugin.loggerProvider)
	require.NotNil(t, plugin.detachLogSink)
	assert.Equal(t, 1, fanout.sinkCount())

	require.NoError(t, plugin.Close())
	assert.Equal(t, 1, fanout.detachCount())
	assert.Nil(t, plugin.detachLogSink)

	// Close is idempotent, so the sink is not detached twice.
	require.NoError(t, plugin.Close())
	assert.Equal(t, 1, fanout.detachCount())
}

func TestOTelPluginBindMeterInstrumentError(t *testing.T) {
	t.Parallel()

	plugin := &OTelPlugin{}

	err := plugin.bindMeter(failingMeter{Meter: metricnoop.NewMeterProvider().Meter(scopeName)})

	require.Error(t, err)
	assert.Contains(t, err.Error(), durationInstrumentName)
	assert.Nil(t, plugin.duration)
}

func TestOTelPluginCloseAggregatesErrorsAndDetachesOnce(t *testing.T) {
	t.Parallel()

	logErr := errors.New("log provider shutdown failed")
	spanErr := errors.New("trace provider shutdown failed")

	// Pre-shutdown the reader so the provider reports ErrReaderShutdown.
	reader := sdkmetric.NewManualReader()
	require.NoError(t, reader.Shutdown(context.Background()))

	detaches := 0
	plugin := &OTelPlugin{
		loggerProvider: newLoggerProvider(nil, &errorLogExporter{err: logErr}),
		meterProvider:  sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)),
		tracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(&errorSpanProcessor{err: spanErr})),
		detachLogSink: func() {
			detaches++
		},
	}

	err := plugin.Close()

	require.Error(t, err)
	assert.ErrorIs(t, err, logErr)
	assert.ErrorIs(t, err, sdkmetric.ErrReaderShutdown)
	assert.ErrorIs(t, err, spanErr)
	assert.Equal(t, 1, detaches)
	assert.Nil(t, plugin.detachLogSink)

	// The cached error is returned on every subsequent call and the sink is
	// detached only once.
	second := plugin.Close()
	require.Error(t, second)
	assert.Equal(t, err, second)
	assert.Equal(t, 1, detaches)
}

func TestOTelPluginWarn(t *testing.T) {
	t.Parallel()

	t.Run("nil logger is a no-op", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			(&OTelPlugin{}).warn("otel: no logger")
		})
	})

	t.Run("non-nil logger records the warning", func(t *testing.T) {
		t.Parallel()

		logger := &internaltests.LevelRecordingLogger{}
		plugin := &OTelPlugin{logger: logger}

		plugin.warn("otel: failed to start runtime metrics", "error", "boom")

		records := logger.Records()
		require.Len(t, records, 1)
		assert.Equal(t, "warn", records[0].Level)
		assert.Equal(t, "otel: failed to start runtime metrics", records[0].Message)
	})
}

// silenceStdout points the stdout exporters at a temp file so a test never
// prints telemetry to the test log. It must not be used by parallel tests
// because os.Stdout is process-global.
func silenceStdout(t *testing.T) {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "otel-stdout-*")
	require.NoError(t, err)

	previous := os.Stdout
	os.Stdout = file
	t.Cleanup(func() {
		os.Stdout = previous
		_ = file.Close()
	})
}

// testFanoutLogger implements models.Logger and models.LoggerFanout so tests
// can exercise the log-sink attach and detach path without touching the
// bootstrap package (which would create an import cycle through the plugin
// factory).
type testFanoutLogger struct {
	mu       sync.Mutex
	sinks    int
	detaches int
}

func (l *testFanoutLogger) Debug(string, ...any) {}
func (l *testFanoutLogger) Info(string, ...any)  {}
func (l *testFanoutLogger) Warn(string, ...any)  {}
func (l *testFanoutLogger) Error(string, ...any) {}

func (l *testFanoutLogger) AddLogger(sink models.Logger) func() {
	l.mu.Lock()
	l.sinks++
	l.mu.Unlock()

	return func() {
		l.mu.Lock()
		l.detaches++
		l.mu.Unlock()
	}
}

func (l *testFanoutLogger) sinkCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.sinks
}

func (l *testFanoutLogger) detachCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.detaches
}

type errorLogExporter struct {
	err error
}

func (e *errorLogExporter) Export(context.Context, []sdklog.Record) error { return e.err }
func (e *errorLogExporter) Shutdown(context.Context) error                { return e.err }
func (e *errorLogExporter) ForceFlush(context.Context) error              { return e.err }

type errorSpanProcessor struct {
	err error
}

func (p *errorSpanProcessor) OnStart(context.Context, sdktrace.ReadWriteSpan) {}
func (p *errorSpanProcessor) OnEnd(sdktrace.ReadOnlySpan)                     {}
func (p *errorSpanProcessor) Shutdown(context.Context) error                  { return p.err }
func (p *errorSpanProcessor) ForceFlush(context.Context) error                { return p.err }

type failingMeter struct {
	metric.Meter
}

func (failingMeter) Float64Histogram(string, ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return nil, errors.New("instrument creation failed")
}
