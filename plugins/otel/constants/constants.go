package constants

const (
	// EnvOTelServiceName is the standard OpenTelemetry environment variable that
	// sets the logical service name reported on every signal.
	EnvOTelServiceName = "OTEL_SERVICE_NAME"

	// EnvOTelExporterOTLPProtocol is the standard OpenTelemetry environment
	// variable that selects the OTLP transport ("grpc" or "http/protobuf").
	EnvOTelExporterOTLPProtocol = "OTEL_EXPORTER_OTLP_PROTOCOL"
)
