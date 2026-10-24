package otel

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/Authula/authula/plugins/otel/constants"
	"github.com/Authula/authula/plugins/otel/types"
)

// buildTraceExporter constructs the configured span exporter. It returns a nil
// exporter when exporting is disabled ("none").
func buildTraceExporter(ctx context.Context, cfg types.OTelPluginConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case types.ExporterNone:
		return nil, nil
	case types.ExporterStdout:
		return stdouttrace.New()
	case types.ExporterOTLP:
		if resolveProtocol(cfg) == types.ProtocolGRPC {
			return otlptracegrpc.New(ctx, traceGRPCOptions(cfg)...)
		}
		return otlptracehttp.New(ctx, traceHTTPOptions(cfg)...)
	default:
		return nil, fmt.Errorf("otel: unsupported trace exporter %q", cfg.Exporter)
	}
}

// buildMetricExporter constructs the configured metric exporter. It returns a
// nil exporter when exporting is disabled ("none").
func buildMetricExporter(ctx context.Context, cfg types.OTelPluginConfig) (sdkmetric.Exporter, error) {
	switch cfg.Exporter {
	case types.ExporterNone:
		return nil, nil
	case types.ExporterStdout:
		return stdoutmetric.New()
	case types.ExporterOTLP:
		if resolveProtocol(cfg) == types.ProtocolGRPC {
			return otlpmetricgrpc.New(ctx, metricGRPCOptions(cfg)...)
		}
		return otlpmetrichttp.New(ctx, metricHTTPOptions(cfg)...)
	default:
		return nil, fmt.Errorf("otel: unsupported metric exporter %q", cfg.Exporter)
	}
}

// buildLogExporter constructs the configured log exporter. It returns a nil
// exporter when exporting is disabled ("none").
func buildLogExporter(ctx context.Context, cfg types.OTelPluginConfig) (sdklog.Exporter, error) {
	switch cfg.Exporter {
	case types.ExporterNone:
		return nil, nil
	case types.ExporterStdout:
		return stdoutlog.New()
	case types.ExporterOTLP:
		if resolveProtocol(cfg) == types.ProtocolGRPC {
			return otlploggrpc.New(ctx, logGRPCOptions(cfg)...)
		}
		return otlploghttp.New(ctx, logHTTPOptions(cfg)...)
	default:
		return nil, fmt.Errorf("otel: unsupported log exporter %q", cfg.Exporter)
	}
}

// newLoggerProvider builds a LoggerProvider that batches records to the
// supplied exporter.
func newLoggerProvider(res *resource.Resource, exp sdklog.Exporter) *sdklog.LoggerProvider {
	opts := make([]sdklog.LoggerProviderOption, 0, 2)
	if res != nil {
		opts = append(opts, sdklog.WithResource(res))
	}
	if exp != nil {
		opts = append(opts, sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)))
	}
	return sdklog.NewLoggerProvider(opts...)
}

// newTracerProvider builds a TracerProvider that batches spans to the supplied
// exporter and only overrides the sampler when a sample ratio is configured.
// Exported so tests can inject their own exporter.
func newTracerProvider(cfg types.OTelPluginConfig, res *resource.Resource, exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	opts := make([]sdktrace.TracerProviderOption, 0, 3)
	if res != nil {
		opts = append(opts, sdktrace.WithResource(res))
	}
	if exp != nil {
		opts = append(opts, sdktrace.WithBatcher(exp))
	}
	if cfg.SampleRatio != nil {
		opts = append(opts, sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(*cfg.SampleRatio))))
	}
	return sdktrace.NewTracerProvider(opts...)
}

// newMeterProvider builds a MeterProvider reading through the supplied reader.
func newMeterProvider(_ types.OTelPluginConfig, res *resource.Resource, reader sdkmetric.Reader) *sdkmetric.MeterProvider {
	opts := make([]sdkmetric.Option, 0, 2)
	if res != nil {
		opts = append(opts, sdkmetric.WithResource(res))
	}
	if reader != nil {
		opts = append(opts, sdkmetric.WithReader(reader))
	}
	return sdkmetric.NewMeterProvider(opts...)
}

// resolveProtocol applies the precedence rule config > OTEL_EXPORTER_OTLP_PROTOCOL > "http".
func resolveProtocol(cfg types.OTelPluginConfig) types.ProtocolType {
	if cfg.Protocol != "" {
		return cfg.Protocol
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv(constants.EnvOTelExporterOTLPProtocol)))
	if strings.Contains(env, "grpc") {
		return types.ProtocolGRPC
	}
	return types.ProtocolHTTP
}

// resolveServiceName applies the precedence rule config > OTEL_SERVICE_NAME > "authula".
func resolveServiceName(cfg types.OTelPluginConfig) string {
	if cfg.ServiceName != "" {
		return cfg.ServiceName
	}
	if name := strings.TrimSpace(os.Getenv(constants.EnvOTelServiceName)); name != "" {
		return name
	}
	return types.DefaultServiceName
}

func traceHTTPOptions(cfg types.OTelPluginConfig) []otlptracehttp.Option {
	opts := make([]otlptracehttp.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}
	return opts
}

func traceGRPCOptions(cfg types.OTelPluginConfig) []otlptracegrpc.Option {
	opts := make([]otlptracegrpc.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlptracegrpc.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
	}
	return opts
}

func metricHTTPOptions(cfg types.OTelPluginConfig) []otlpmetrichttp.Option {
	opts := make([]otlpmetrichttp.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlpmetrichttp.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}
	return opts
}

func metricGRPCOptions(cfg types.OTelPluginConfig) []otlpmetricgrpc.Option {
	opts := make([]otlpmetricgrpc.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlpmetricgrpc.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.Headers))
	}
	return opts
}

func logHTTPOptions(cfg types.OTelPluginConfig) []otlploghttp.Option {
	opts := make([]otlploghttp.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlploghttp.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(cfg.Headers))
	}
	return opts
}

func logGRPCOptions(cfg types.OTelPluginConfig) []otlploggrpc.Option {
	opts := make([]otlploggrpc.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlploggrpc.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(cfg.Headers))
	}
	return opts
}
