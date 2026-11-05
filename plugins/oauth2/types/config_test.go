package types

import (
	"reflect"
	"strings"
	"testing"
)

func TestProviderConfig_PKCEEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config ProviderConfig
		want   bool
	}{
		{name: "absent defaults to enabled", config: ProviderConfig{}, want: true},
		{name: "explicit true", config: ProviderConfig{PKCE: new(true)}, want: true},
		{name: "explicit false opts out", config: ProviderConfig{PKCE: new(false)}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.config.PKCEEnabled(); got != tt.want {
				t.Fatalf("PKCEEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPKCEConfigSurface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		typ         reflect.Type
		wantPresent bool
	}{
		{name: "plugin config has no global PKCE toggle", typ: reflect.TypeFor[OAuth2PluginConfig](), wantPresent: false},
		{name: "provider config exposes per-provider PKCE", typ: reflect.TypeFor[ProviderConfig](), wantPresent: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			present := false
			for field := range tt.typ.Fields() {
				if strings.EqualFold(field.Name, "PKCE") {
					present = true
					break
				}
			}
			if present != tt.wantPresent {
				t.Fatalf("%s PKCE field present = %v, want %v", tt.typ.Name(), present, tt.wantPresent)
			}
		})
	}
}
