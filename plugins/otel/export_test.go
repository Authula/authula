package otel

import (
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/Authula/authula/models"
)

// AttachLogsForTest wires the log sink with an in-memory exporter, bypassing
// exporter construction so tests never touch the network.
func AttachLogsForTest(p *OTelPlugin, logger models.Logger, exp sdklog.Exporter) {
	p.logger = logger
	p.initLoggerWithExporter(nil, exp)
}

// LogProviderForTest reports the provider created by the log wiring.
func LogProviderForTest(p *OTelPlugin) *sdklog.LoggerProvider {
	return p.loggerProvider
}
