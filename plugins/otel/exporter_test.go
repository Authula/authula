package otel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"github.com/Authula/authula/plugins/otel/constants"
	"github.com/Authula/authula/plugins/otel/types"
)

const shutdownTimeout = 2 * time.Second

// isolateOTelEnv clears the OTel environment variables the plugin reads so a
// developer's shell cannot influence the assertions below.
func isolateOTelEnv(t *testing.T) {
	t.Helper()

	t.Setenv(constants.EnvOTelExporterOTLPProtocol, "")
	t.Setenv(constants.EnvOTelServiceName, "")
}

func shutdownWithTimeout(t *testing.T, shutdown func(context.Context) error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	require.NoError(t, shutdown(ctx))
}

func testResource() *resource.Resource {
	return resource.NewSchemaless(semconv.ServiceName("otel-test"))
}

// stubLogExporter satisfies sdklog.Exporter without writing anything anywhere.
type stubLogExporter struct{}

func (stubLogExporter) Export(context.Context, []sdklog.Record) error { return nil }
func (stubLogExporter) Shutdown(context.Context) error                { return nil }
func (stubLogExporter) ForceFlush(context.Context) error              { return nil }

// stubSpanExporter satisfies sdktrace.SpanExporter without writing anything.
type stubSpanExporter struct{}

func (stubSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }
func (stubSpanExporter) Shutdown(context.Context) error                             { return nil }

type exporterBuildCase struct {
	name    string
	config  types.OTelPluginConfig
	wantNil bool
	wantErr string
}

func otlpConfig(protocol types.ProtocolType) types.OTelPluginConfig {
	return types.OTelPluginConfig{
		Exporter: types.ExporterOTLP,
		Protocol: protocol,
		Endpoint: "127.0.0.1:4317",
		Insecure: true,
		Headers:  map[string]string{"x-test": "1"},
	}
}

func exporterBuildCases(prefix string) []exporterBuildCase {
	return []exporterBuildCase{
		{name: "none returns nil", config: types.OTelPluginConfig{Exporter: types.ExporterNone}, wantNil: true},
		{name: "stdout", config: types.OTelPluginConfig{Exporter: types.ExporterStdout}},
		{name: "otlp http", config: otlpConfig(types.ProtocolHTTP)},
		{name: "otlp grpc", config: otlpConfig(types.ProtocolGRPC)},
		{
			name:    "unsupported exporter",
			config:  types.OTelPluginConfig{Exporter: "jaeger"},
			wantErr: "unsupported " + prefix + " exporter",
		},
	}
}

// shutdownExporter shuts down whichever signal exporter the builders returned.
func shutdownExporter(t *testing.T, exp any) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var err error
	switch exporter := exp.(type) {
	case sdktrace.SpanExporter:
		err = exporter.Shutdown(ctx)
	case sdkmetric.Exporter:
		err = exporter.Shutdown(ctx)
	case sdklog.Exporter:
		err = exporter.Shutdown(ctx)
	}
	require.NoError(t, err)
}

func TestBuildTraceExporter(t *testing.T) {
	isolateOTelEnv(t)

	for _, tt := range exporterBuildCases("trace") {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := buildTraceExporter(context.Background(), tt.config)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, exp)
				return
			}

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, exp)
				return
			}

			require.NotNil(t, exp)
			t.Cleanup(func() { shutdownExporter(t, exp) })
		})
	}
}

func TestBuildMetricExporter(t *testing.T) {
	isolateOTelEnv(t)

	for _, tt := range exporterBuildCases("metric") {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := buildMetricExporter(context.Background(), tt.config)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, exp)
				return
			}

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, exp)
				return
			}

			require.NotNil(t, exp)
			t.Cleanup(func() { shutdownExporter(t, exp) })
		})
	}
}

func TestBuildLogExporter(t *testing.T) {
	isolateOTelEnv(t)

	for _, tt := range exporterBuildCases("log") {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := buildLogExporter(context.Background(), tt.config)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, exp)
				return
			}

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, exp)
				return
			}

			require.NotNil(t, exp)
			t.Cleanup(func() { shutdownExporter(t, exp) })
		})
	}
}

func TestExporterOptionBuilders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config types.OTelPluginConfig
		want   int
	}{
		{name: "empty config", config: types.OTelPluginConfig{}, want: 0},
		{name: "endpoint only", config: types.OTelPluginConfig{Endpoint: "collector:4317"}, want: 1},
		{name: "insecure only", config: types.OTelPluginConfig{Insecure: true}, want: 1},
		{
			name:   "headers only",
			config: types.OTelPluginConfig{Headers: map[string]string{"authorization": "Bearer token"}},
			want:   1,
		},
		{
			name:   "endpoint and insecure",
			config: types.OTelPluginConfig{Endpoint: "collector:4317", Insecure: true},
			want:   2,
		},
		{
			name: "all options",
			config: types.OTelPluginConfig{
				Endpoint: "collector:4317",
				Insecure: true,
				Headers:  map[string]string{"authorization": "Bearer token", "x-tenant": "acme"},
			},
			want: 3,
		},
	}

	builders := []struct {
		name  string
		count func(types.OTelPluginConfig) int
	}{
		{name: "traceHTTPOptions", count: func(c types.OTelPluginConfig) int { return len(traceHTTPOptions(c)) }},
		{name: "traceGRPCOptions", count: func(c types.OTelPluginConfig) int { return len(traceGRPCOptions(c)) }},
		{name: "metricHTTPOptions", count: func(c types.OTelPluginConfig) int { return len(metricHTTPOptions(c)) }},
		{name: "metricGRPCOptions", count: func(c types.OTelPluginConfig) int { return len(metricGRPCOptions(c)) }},
		{name: "logHTTPOptions", count: func(c types.OTelPluginConfig) int { return len(logHTTPOptions(c)) }},
		{name: "logGRPCOptions", count: func(c types.OTelPluginConfig) int { return len(logGRPCOptions(c)) }},
	}

	for _, builder := range builders {
		t.Run(builder.name, func(t *testing.T) {
			t.Parallel()

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()

					assert.Equal(t, tt.want, builder.count(tt.config))
				})
			}
		})
	}
}

func TestNewLoggerProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		withResource bool
		withExporter bool
	}{
		{name: "no options"},
		{name: "resource only", withResource: true},
		{name: "exporter only", withExporter: true},
		{name: "resource and exporter", withResource: true, withExporter: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var res *resource.Resource
			if tt.withResource {
				res = testResource()
			}

			var exp sdklog.Exporter
			if tt.withExporter {
				exp = stubLogExporter{}
			}

			provider := newLoggerProvider(res, exp)
			require.NotNil(t, provider)
			t.Cleanup(func() { shutdownWithTimeout(t, provider.Shutdown) })
		})
	}
}

func TestNewTracerProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		withResource bool
		withExporter bool
		sampleRatio  *float64
		wantSampled  bool
	}{
		{name: "no options", wantSampled: true},
		{name: "resource only", withResource: true, wantSampled: true},
		{name: "exporter only", withExporter: true, wantSampled: true},
		{
			name:         "resource, exporter and full sampling",
			withResource: true,
			withExporter: true,
			sampleRatio:  new(1.0),
			wantSampled:  true,
		},
		{
			name:         "resource, exporter and zero sampling",
			withResource: true,
			withExporter: true,
			sampleRatio:  new(0.0),
			wantSampled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var res *resource.Resource
			if tt.withResource {
				res = testResource()
			}

			var exp sdktrace.SpanExporter
			if tt.withExporter {
				exp = stubSpanExporter{}
			}

			provider := newTracerProvider(types.OTelPluginConfig{SampleRatio: tt.sampleRatio}, res, exp)
			require.NotNil(t, provider)
			t.Cleanup(func() { shutdownWithTimeout(t, provider.Shutdown) })

			_, span := provider.Tracer(scopeName).Start(context.Background(), "test-span")
			assert.Equal(t, tt.wantSampled, span.SpanContext().IsSampled())
			span.End()
		})
	}
}

func TestNewMeterProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		withResource bool
		withReader   bool
	}{
		{name: "no options"},
		{name: "resource only", withResource: true},
		{name: "reader only", withReader: true},
		{name: "resource and reader", withResource: true, withReader: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var res *resource.Resource
			if tt.withResource {
				res = testResource()
			}

			var reader sdkmetric.Reader
			if tt.withReader {
				reader = sdkmetric.NewManualReader()
			}

			provider := newMeterProvider(types.OTelPluginConfig{}, res, reader)
			require.NotNil(t, provider)
			t.Cleanup(func() { shutdownWithTimeout(t, provider.Shutdown) })
		})
	}
}
