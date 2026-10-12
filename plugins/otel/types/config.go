package types

// DefaultServiceName is used when neither the plugin config nor the
// OTEL_SERVICE_NAME environment variable provides a service name.
const DefaultServiceName = "authula"

// ExporterType selects where telemetry is exported.
type ExporterType string

const (
	// ExporterOTLP exports over the OpenTelemetry Protocol.
	ExporterOTLP ExporterType = "otlp"
	// ExporterStdout writes telemetry to standard output (useful for debugging).
	ExporterStdout ExporterType = "stdout"
	// ExporterNone disables exporting entirely.
	ExporterNone ExporterType = "none"
)

// ProtocolType selects the OTLP transport.
type ProtocolType string

const (
	// ProtocolHTTP uses OTLP over HTTP/protobuf.
	ProtocolHTTP ProtocolType = "http"
	// ProtocolGRPC uses OTLP over gRPC.
	ProtocolGRPC ProtocolType = "grpc"
)

// OTelPluginConfig is the OpenTelemetry plugin configuration.
//
// It carries both json tags (used by util.LoadPluginConfig/mapstructure) and
// matching toml tags for consistency with the other plugin configs.
//
// TracesEnabled, MetricsEnabled and RuntimeMetrics are pointers so an explicit
// `false` is distinguishable from "unset" (which defaults to true).
type OTelPluginConfig struct {
	Enabled        bool              `json:"enabled" toml:"enabled"`
	ServiceName    string            `json:"service_name,omitempty" toml:"service_name"`
	Exporter       ExporterType      `json:"exporter,omitempty" toml:"exporter"`
	Protocol       ProtocolType      `json:"protocol,omitempty" toml:"protocol"`
	Endpoint       string            `json:"endpoint,omitempty" toml:"endpoint"`
	Insecure       bool              `json:"insecure,omitempty" toml:"insecure"`
	Headers        map[string]string `json:"headers,omitempty" toml:"headers"`
	TracesEnabled  *bool             `json:"traces_enabled,omitempty" toml:"traces_enabled"`
	MetricsEnabled *bool             `json:"metrics_enabled,omitempty" toml:"metrics_enabled"`
	RuntimeMetrics *bool             `json:"runtime_metrics,omitempty" toml:"runtime_metrics"`
	SampleRatio    *float64          `json:"sample_ratio,omitempty" toml:"sample_ratio"`
	TrustProxy     bool              `json:"trust_proxy,omitempty" toml:"trust_proxy"`
}

// ApplyDefaults fills unset fields with their default values. It does not touch
// Enabled, which is resolved by the plugin factory and the Init guard.
func (c *OTelPluginConfig) ApplyDefaults() {
	if c.Exporter == "" {
		c.Exporter = ExporterOTLP
	}
	if c.TracesEnabled == nil {
		c.TracesEnabled = new(true)
	}
	if c.MetricsEnabled == nil {
		c.MetricsEnabled = new(true)
	}
	if c.RuntimeMetrics == nil {
		c.RuntimeMetrics = new(true)
	}
}

// TracesOn reports whether tracing is enabled, defaulting to true when unset.
func (c OTelPluginConfig) TracesOn() bool {
	return boolValue(c.TracesEnabled, true)
}

// MetricsOn reports whether metrics are enabled, defaulting to true when unset.
func (c OTelPluginConfig) MetricsOn() bool {
	return boolValue(c.MetricsEnabled, true)
}

// RuntimeOn reports whether runtime metrics are enabled, defaulting to true
// when unset.
func (c OTelPluginConfig) RuntimeOn() bool {
	return boolValue(c.RuntimeMetrics, true)
}

// TracesActive reports whether tracing should be initialized. It additionally
// treats the "none" exporter as a full disable.
func (c OTelPluginConfig) TracesActive() bool {
	return c.TracesOn() && c.Exporter != ExporterNone
}

// MetricsActive reports whether metrics should be initialized. It additionally
// treats the "none" exporter as a full disable.
func (c OTelPluginConfig) MetricsActive() bool {
	return c.MetricsOn() && c.Exporter != ExporterNone
}

// Valid reports whether the configuration values are within their supported
// ranges.
func (c OTelPluginConfig) Valid() bool {
	switch c.Exporter {
	case ExporterOTLP, ExporterStdout, ExporterNone:
	default:
		return false
	}

	switch c.Protocol {
	case "", ProtocolHTTP, ProtocolGRPC:
	default:
		return false
	}

	if c.SampleRatio != nil && (*c.SampleRatio < 0 || *c.SampleRatio > 1) {
		return false
	}

	return true
}

func boolValue(value *bool, fallback bool) bool {
	if value != nil {
		return *value
	}
	return fallback
}
