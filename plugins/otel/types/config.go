package types

const DefaultServiceName = "authula"

type ExporterType string

const (
	ExporterOTLP   ExporterType = "otlp"
	ExporterStdout ExporterType = "stdout"
	ExporterNone   ExporterType = "none"
)

type ProtocolType string

const (
	ProtocolHTTP ProtocolType = "http"
	ProtocolGRPC ProtocolType = "grpc"
)

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
	LogsEnabled    bool              `json:"logs_enabled,omitempty" toml:"logs_enabled"`
	RuntimeMetrics *bool             `json:"runtime_metrics,omitempty" toml:"runtime_metrics"`
	SampleRatio    *float64          `json:"sample_ratio,omitempty" toml:"sample_ratio"`
	TrustProxy     bool              `json:"trust_proxy,omitempty" toml:"trust_proxy"`
}

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

func (c OTelPluginConfig) TracesOn() bool {
	return boolValue(c.TracesEnabled, true)
}

func (c OTelPluginConfig) MetricsOn() bool {
	return boolValue(c.MetricsEnabled, true)
}

// RuntimeOn reports whether Go runtime metrics should be started. It defaults
// to true when unset.
func (c OTelPluginConfig) RuntimeOn() bool {
	return boolValue(c.RuntimeMetrics, true)
}

func (c OTelPluginConfig) TracesActive() bool {
	return c.TracesOn() && c.Exporter != ExporterNone
}

func (c OTelPluginConfig) MetricsActive() bool {
	return c.MetricsOn() && c.Exporter != ExporterNone
}

func (c OTelPluginConfig) LogsActive() bool {
	return c.LogsEnabled && c.Exporter != ExporterNone
}

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
