package otel

import (
	"context"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/Authula/authula/models"
)

// initLogger builds the log exporter/provider and attaches the OTel sink to the
// core logger fan-out. It is a no-op unless logs_enabled is set.
func (p *OTelPlugin) initLogger(res *resource.Resource) error {
	if !p.pluginConfig.LogsActive() {
		return nil
	}

	exp, err := buildLogExporter(context.Background(), p.pluginConfig)
	if err != nil {
		return err
	}

	p.initLoggerWithExporter(res, exp)

	return nil
}

// initLoggerWithExporter creates the provider and attaches the sink. Exporters
// are injected so tests can supply an in-memory one.
func (p *OTelPlugin) initLoggerWithExporter(res *resource.Resource, exp sdklog.Exporter) {
	p.loggerProvider = newLoggerProvider(res, exp)

	fanout, ok := p.logger.(models.LoggerFanout)
	if !ok {
		p.warn("otel: log bridge not attached (logger does not support fan-out)")
		return
	}

	p.detachLogSink = fanout.AddLogger(newBridgeLogger(
		p.loggerProvider,
		p.Metadata().Version,
		parseLevel(p.logLevel()),
	))
}

// logLevel reads logger.level from the core config. It is nil-safe and returns
// an empty string when unavailable, which parseLevel treats as info.
func (p *OTelPlugin) logLevel() string {
	if p.pluginCtx == nil || p.pluginCtx.GetConfig == nil {
		return ""
	}

	config := p.pluginCtx.GetConfig()
	if config == nil {
		return ""
	}

	return config.Logger.Level
}
