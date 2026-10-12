package types_test

import (
	"testing"

	"github.com/Authula/authula/plugins/otel/types"
)

func TestOTelPluginConfigApplyDefaults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		config   types.OTelPluginConfig
		expected types.OTelPluginConfig
	}{
		{
			name:   "empty config gets all defaults",
			config: types.OTelPluginConfig{},
			expected: types.OTelPluginConfig{
				Exporter:       types.ExporterOTLP,
				TracesEnabled:  new(true),
				MetricsEnabled: new(true),
				RuntimeMetrics: new(true),
			},
		},
		{
			name: "explicit exporter is preserved",
			config: types.OTelPluginConfig{
				Exporter: types.ExporterStdout,
			},
			expected: types.OTelPluginConfig{
				Exporter:       types.ExporterStdout,
				TracesEnabled:  new(true),
				MetricsEnabled: new(true),
				RuntimeMetrics: new(true),
			},
		},
		{
			name: "explicit toggles are preserved",
			config: types.OTelPluginConfig{
				TracesEnabled:  new(false),
				MetricsEnabled: new(true),
				RuntimeMetrics: new(false),
			},
			expected: types.OTelPluginConfig{
				Exporter:       types.ExporterOTLP,
				TracesEnabled:  new(false),
				MetricsEnabled: new(true),
				RuntimeMetrics: new(false),
			},
		},
		{
			name: "all other fields are preserved",
			config: types.OTelPluginConfig{
				Enabled:     true,
				ServiceName: "billing",
				Exporter:    types.ExporterOTLP,
				Protocol:    types.ProtocolGRPC,
				Endpoint:    "collector:4317",
				Insecure:    true,
				Headers:     map[string]string{"x-token": "abc"},
				SampleRatio: new(0.25),
				TrustProxy:  true,
			},
			expected: types.OTelPluginConfig{
				Enabled:        true,
				ServiceName:    "billing",
				Exporter:       types.ExporterOTLP,
				Protocol:       types.ProtocolGRPC,
				Endpoint:       "collector:4317",
				Insecure:       true,
				Headers:        map[string]string{"x-token": "abc"},
				TracesEnabled:  new(true),
				MetricsEnabled: new(true),
				RuntimeMetrics: new(true),
				SampleRatio:    new(0.25),
				TrustProxy:     true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := tc.config
			config.ApplyDefaults()

			if config.Exporter != tc.expected.Exporter {
				t.Errorf("Exporter = %q, want %q", config.Exporter, tc.expected.Exporter)
			}
			if *config.TracesEnabled != *tc.expected.TracesEnabled {
				t.Errorf("TracesEnabled = %v, want %v", *config.TracesEnabled, *tc.expected.TracesEnabled)
			}
			if *config.MetricsEnabled != *tc.expected.MetricsEnabled {
				t.Errorf("MetricsEnabled = %v, want %v", *config.MetricsEnabled, *tc.expected.MetricsEnabled)
			}
			if *config.RuntimeMetrics != *tc.expected.RuntimeMetrics {
				t.Errorf("RuntimeMetrics = %v, want %v", *config.RuntimeMetrics, *tc.expected.RuntimeMetrics)
			}
			if config.ServiceName != tc.expected.ServiceName {
				t.Errorf("ServiceName = %q, want %q", config.ServiceName, tc.expected.ServiceName)
			}
			if config.Protocol != tc.expected.Protocol {
				t.Errorf("Protocol = %q, want %q", config.Protocol, tc.expected.Protocol)
			}
			if config.Endpoint != tc.expected.Endpoint {
				t.Errorf("Endpoint = %q, want %q", config.Endpoint, tc.expected.Endpoint)
			}
			if config.Insecure != tc.expected.Insecure {
				t.Errorf("Insecure = %v, want %v", config.Insecure, tc.expected.Insecure)
			}
			if config.TrustProxy != tc.expected.TrustProxy {
				t.Errorf("TrustProxy = %v, want %v", config.TrustProxy, tc.expected.TrustProxy)
			}
		})
	}
}

func TestOTelPluginConfigSampleRatio(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		ratio  *float64
		expect bool
	}{
		{name: "nil is valid", ratio: nil, expect: true},
		{name: "zero is valid", ratio: new(0.0), expect: true},
		{name: "one is valid", ratio: new(1.0), expect: true},
		{name: "fraction is valid", ratio: new(0.5), expect: true},
		{name: "below zero is invalid", ratio: new(-0.1), expect: false},
		{name: "above one is invalid", ratio: new(1.1), expect: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := types.OTelPluginConfig{Exporter: types.ExporterOTLP, SampleRatio: tc.ratio}
			if got := config.Valid(); got != tc.expect {
				t.Fatalf("Valid() = %v, want %v", got, tc.expect)
			}
		})
	}
}

func TestOTelPluginConfigValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		config types.OTelPluginConfig
		valid  bool
	}{
		{name: "otlp http", config: types.OTelPluginConfig{Exporter: types.ExporterOTLP, Protocol: types.ProtocolHTTP}, valid: true},
		{name: "otlp grpc", config: types.OTelPluginConfig{Exporter: types.ExporterOTLP, Protocol: types.ProtocolGRPC}, valid: true},
		{name: "empty protocol", config: types.OTelPluginConfig{Exporter: types.ExporterOTLP}, valid: true},
		{name: "stdout", config: types.OTelPluginConfig{Exporter: types.ExporterStdout}, valid: true},
		{name: "none", config: types.OTelPluginConfig{Exporter: types.ExporterNone}, valid: true},
		{name: "unknown exporter", config: types.OTelPluginConfig{Exporter: "carrier-pigeon"}, valid: false},
		{name: "unknown protocol", config: types.OTelPluginConfig{Exporter: types.ExporterOTLP, Protocol: "quic"}, valid: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.config.Valid(); got != tc.valid {
				t.Fatalf("Valid() = %v, want %v", got, tc.valid)
			}
		})
	}
}

func TestOTelPluginConfigActive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		config    types.OTelPluginConfig
		tracesOn  bool
		metricsOn bool
	}{
		{
			name:      "exporter none disables both",
			config:    types.OTelPluginConfig{Exporter: types.ExporterNone, TracesEnabled: new(true), MetricsEnabled: new(true)},
			tracesOn:  false,
			metricsOn: false,
		},
		{
			name:      "otlp enables both",
			config:    types.OTelPluginConfig{Exporter: types.ExporterOTLP, TracesEnabled: new(true), MetricsEnabled: new(true)},
			tracesOn:  true,
			metricsOn: true,
		},
		{
			name:      "individual toggles respected",
			config:    types.OTelPluginConfig{Exporter: types.ExporterStdout, TracesEnabled: new(false), MetricsEnabled: new(true)},
			tracesOn:  false,
			metricsOn: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.config.TracesActive(); got != tc.tracesOn {
				t.Errorf("TracesActive() = %v, want %v", got, tc.tracesOn)
			}
			if got := tc.config.MetricsActive(); got != tc.metricsOn {
				t.Errorf("MetricsActive() = %v, want %v", got, tc.metricsOn)
			}
		})
	}
}
