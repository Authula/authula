package otel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/otel/types"
	"github.com/Authula/authula/util"
)

// scopeName identifies the instrumentation scope attached to every span and
// metric produced by the plugin.
const scopeName = "github.com/Authula/authula/plugins/otel"

// OTelPlugin provides distributed tracing and request/runtime metrics backed by
// the OpenTelemetry SDK. All providers are instance-local: the plugin never
// touches the process-global otel providers.
type OTelPlugin struct {
	pluginConfig types.OTelPluginConfig
	pluginCtx    *models.PluginContext
	logger       models.Logger

	tracer     trace.Tracer
	meter      metric.Meter
	propagator propagation.TextMapPropagator

	tracerProvider *sdktrace.TracerProvider // nil when traces disabled/none
	meterProvider  *sdkmetric.MeterProvider // nil when metrics disabled/none
	duration       metric.Float64Histogram

	closeOnce sync.Once
	closeErr  error
}

func New(config types.OTelPluginConfig) *OTelPlugin {
	config.ApplyDefaults()
	return &OTelPlugin{pluginConfig: config}
}

func (p *OTelPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginOtel.String(),
		Version:     "1.0.0",
		Description: "Provides OpenTelemetry distributed tracing and request/runtime metrics",
	}
}

func (p *OTelPlugin) Config() any {
	return p.pluginConfig
}

// Init builds the tracer, meter and propagator. Invalid configuration and
// provider construction failures are returned so the server refuses to start
// rather than running blind. A disabled plugin is a no-op.
func (p *OTelPlugin) Init(pluginCtx *models.PluginContext) error {
	p.pluginCtx = pluginCtx
	p.logger = pluginCtx.Logger

	if err := util.LoadPluginConfig(pluginCtx.GetConfig(), p.Metadata().ID, &p.pluginConfig); err != nil {
		return fmt.Errorf("otel: failed to load plugin config: %w", err)
	}

	p.pluginConfig.ApplyDefaults()

	if !p.pluginConfig.Enabled {
		return nil
	}

	if !p.pluginConfig.Valid() {
		return fmt.Errorf(
			"otel: invalid configuration: exporter=%q protocol=%q sample_ratio=%v",
			p.pluginConfig.Exporter,
			p.pluginConfig.Protocol,
			p.pluginConfig.SampleRatio,
		)
	}

	p.propagator = propagation.TraceContext{}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(resolveServiceName(p.pluginConfig)),
			semconv.ServiceVersion(p.Metadata().Version),
		),
	)
	if err != nil {
		return fmt.Errorf("otel: failed to build resource: %w", err)
	}

	if err := p.initTracer(res); err != nil {
		return err
	}

	if err := p.initMeter(res); err != nil {
		return err
	}

	if p.pluginConfig.RuntimeOn() && p.pluginConfig.MetricsActive() {
		if err := runtime.Start(runtime.WithMeterProvider(p.meterProvider)); err != nil {
			p.warn("otel: failed to start runtime metrics", "error", err)
		}
	}

	return nil
}

func (p *OTelPlugin) initTracer(res *resource.Resource) error {
	if !p.pluginConfig.TracesActive() {
		p.tracer = tracenoop.NewTracerProvider().Tracer(scopeName)
		return nil
	}

	exp, err := buildTraceExporter(context.Background(), p.pluginConfig)
	if err != nil {
		return err
	}

	p.tracerProvider = newTracerProvider(p.pluginConfig, res, exp)
	p.tracer = p.tracerProvider.Tracer(scopeName, trace.WithInstrumentationVersion(p.Metadata().Version))

	return nil
}

func (p *OTelPlugin) initMeter(res *resource.Resource) error {
	if !p.pluginConfig.MetricsActive() {
		return p.bindMeter(metricnoop.NewMeterProvider().Meter(scopeName))
	}

	exp, err := buildMetricExporter(context.Background(), p.pluginConfig)
	if err != nil {
		return err
	}

	var reader sdkmetric.Reader
	if exp != nil {
		reader = sdkmetric.NewPeriodicReader(exp)
	}
	p.meterProvider = newMeterProvider(p.pluginConfig, res, reader)

	return p.bindMeter(p.meterProvider.Meter(scopeName, metric.WithInstrumentationVersion(p.Metadata().Version)))
}

// bindMeter stores the meter and creates the sole instrument used by the
// plugin. It is split out so tests can inject an in-memory meter.
func (p *OTelPlugin) bindMeter(meter metric.Meter) error {
	p.meter = meter

	duration, err := meter.Float64Histogram(
		durationInstrumentName,
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(durationBuckets...),
	)
	if err != nil {
		return fmt.Errorf("otel: failed to create %s instrument: %w", durationInstrumentName, err)
	}
	p.duration = duration

	return nil
}

func (p *OTelPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *OTelPlugin) Close() error {
	p.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var errs []error
		if p.meterProvider != nil {
			if err := p.meterProvider.Shutdown(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if p.tracerProvider != nil {
			if err := p.tracerProvider.Shutdown(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		p.closeErr = errors.Join(errs...)
	})

	return p.closeErr
}

func (p *OTelPlugin) warn(msg string, args ...any) {
	if p.logger != nil {
		p.logger.Warn(msg, args...)
	}
}
