package otel

import (
	"testing"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/otel/constants"
	"github.com/Authula/authula/plugins/otel/types"
)

func nilConfig() *models.Config {
	return nil
}

func TestOTelPluginMetadata(t *testing.T) {
	t.Parallel()

	plugin := New(types.OTelPluginConfig{Enabled: true})

	metadata := plugin.Metadata()
	if metadata.ID != models.PluginOtel.String() {
		t.Fatalf("Metadata().ID = %q, want %q", metadata.ID, models.PluginOtel.String())
	}
	if metadata.Version != "1.0.0" {
		t.Fatalf("Metadata().Version = %q, want %q", metadata.Version, "1.0.0")
	}
	if plugin.Config() == nil {
		t.Fatal("Config() should not be nil")
	}
}

func TestOTelPluginHooks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		traces     bool
		metrics    bool
		wantOnReq  bool
		wantOnDone bool
	}{
		{name: "both enabled", traces: true, metrics: true, wantOnReq: true, wantOnDone: true},
		{name: "only traces", traces: true, metrics: false, wantOnReq: true, wantOnDone: true},
		{name: "only metrics", traces: false, metrics: true, wantOnReq: true, wantOnDone: true},
		{name: "both disabled", traces: false, metrics: false, wantOnReq: false, wantOnDone: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := &OTelPlugin{}
			plugin.pluginConfig.TracesEnabled = new(tc.traces)
			plugin.pluginConfig.MetricsEnabled = new(tc.metrics)

			hooks := plugin.Hooks()
			if !tc.wantOnReq && !tc.wantOnDone {
				if hooks != nil {
					t.Fatalf("Hooks() = %v, want nil", hooks)
				}
				return
			}

			var onReq, onDone int
			for _, hook := range hooks {
				switch hook.Stage {
				case models.HookOnRequest:
					onReq++
					if !hook.RunOnBypass {
						t.Error("HookOnRequest should opt into bypass paths")
					}
					if hook.Order != -100 {
						t.Errorf("HookOnRequest Order = %d, want -100", hook.Order)
					}
				case models.HookOnComplete:
					onDone++
				}
			}

			if (onReq > 0) != tc.wantOnReq {
				t.Errorf("HookOnRequest count = %d, want present = %v", onReq, tc.wantOnReq)
			}
			if (onDone > 0) != tc.wantOnDone {
				t.Errorf("HookOnComplete count = %d, want present = %v", onDone, tc.wantOnDone)
			}
		})
	}
}

func TestOTelPluginInitDisabledIsNoOp(t *testing.T) {
	t.Parallel()

	plugin := New(types.OTelPluginConfig{Enabled: false})

	err := plugin.Init(&models.PluginContext{GetConfig: nilConfig})
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}
	if plugin.tracerProvider != nil {
		t.Error("tracerProvider should be nil when disabled")
	}
	if plugin.meterProvider != nil {
		t.Error("meterProvider should be nil when disabled")
	}
}

func TestOTelPluginInitInvalidConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		config types.OTelPluginConfig
	}{
		{name: "unknown exporter", config: types.OTelPluginConfig{Exporter: "unknown"}},
		{name: "unknown protocol", config: types.OTelPluginConfig{Exporter: types.ExporterOTLP, Protocol: "quic"}},
		{name: "sample ratio below zero", config: types.OTelPluginConfig{Exporter: types.ExporterOTLP, SampleRatio: new(-0.5)}},
		{name: "sample ratio above one", config: types.OTelPluginConfig{Exporter: types.ExporterOTLP, SampleRatio: new(2.0)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := tc.config
			config.Enabled = true
			plugin := New(config)

			if err := plugin.Init(&models.PluginContext{GetConfig: nilConfig}); err == nil {
				t.Fatal("Init() error = nil, want non-nil")
			}
		})
	}
}

func TestOTelPluginInitAndClose(t *testing.T) {
	t.Parallel()

	plugin := New(types.OTelPluginConfig{
		Enabled:        true,
		Exporter:       types.ExporterStdout,
		ServiceName:    "test-service",
		MetricsEnabled: new(false),
		RuntimeMetrics: new(false),
	})
	if err := plugin.Init(&models.PluginContext{GetConfig: nilConfig}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if plugin.tracerProvider == nil {
		t.Error("tracerProvider should not be nil")
	}
	if plugin.tracer == nil {
		t.Error("tracer should not be nil")
	}
	if plugin.duration == nil {
		t.Error("duration instrument should not be nil")
	}

	if err := plugin.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	// Close is idempotent.
	if err := plugin.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestOTelPluginCloseNeverInitialized(t *testing.T) {
	t.Parallel()

	if err := (&OTelPlugin{}).Close(); err != nil {
		t.Fatalf("Close() on zero plugin error = %v", err)
	}
}

func TestResolveServiceName(t *testing.T) {
	cases := []struct {
		name     string
		config   types.OTelPluginConfig
		env      string
		expected string
	}{
		{name: "config wins", config: types.OTelPluginConfig{ServiceName: "from-config"}, env: "from-env", expected: "from-config"},
		{name: "env used when config empty", config: types.OTelPluginConfig{}, env: "from-env", expected: "from-env"},
		{name: "default when both empty", config: types.OTelPluginConfig{}, env: "", expected: types.DefaultServiceName},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env == "" {
				t.Setenv(constants.EnvOTelServiceName, "")
			} else {
				t.Setenv(constants.EnvOTelServiceName, tc.env)
			}

			if got := resolveServiceName(tc.config); got != tc.expected {
				t.Fatalf("resolveServiceName() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestResolveProtocol(t *testing.T) {
	cases := []struct {
		name     string
		config   types.OTelPluginConfig
		env      string
		expected types.ProtocolType
	}{
		{name: "config wins", config: types.OTelPluginConfig{Protocol: types.ProtocolGRPC}, env: "http/protobuf", expected: types.ProtocolGRPC},
		{name: "env grpc", config: types.OTelPluginConfig{}, env: "grpc", expected: types.ProtocolGRPC},
		{name: "env http", config: types.OTelPluginConfig{}, env: "http/protobuf", expected: types.ProtocolHTTP},
		{name: "default http", config: types.OTelPluginConfig{}, env: "", expected: types.ProtocolHTTP},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(constants.EnvOTelExporterOTLPProtocol, tc.env)

			if got := resolveProtocol(tc.config); got != tc.expected {
				t.Fatalf("resolveProtocol() = %q, want %q", got, tc.expected)
			}
		})
	}
}
