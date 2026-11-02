package otel

import (
	"context"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/Authula/authula/models"
)

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
