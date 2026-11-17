package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

type configTestPluginConfig struct {
	Enabled bool          `json:"enabled"`
	Timeout time.Duration `json:"timeout"`
	Roles   []string      `json:"roles"`
}

type configTestEnabledOnly struct {
	Enabled bool `json:"enabled"`
}

func TestParsePluginConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source any
		want   configTestPluginConfig
	}{
		{
			name:   "decodes duration and comma separated values",
			source: map[string]any{"enabled": true, "timeout": "5m", "roles": "admin,user"},
			want: configTestPluginConfig{
				Enabled: true,
				Timeout: 5 * time.Minute,
				Roles:   []string{"admin", "user"},
			},
		},
		{
			name:   "decodes slice values",
			source: map[string]any{"timeout": 30 * time.Second, "roles": []any{"admin", "user"}},
			want: configTestPluginConfig{
				Timeout: 30 * time.Second,
				Roles:   []string{"admin", "user"},
			},
		},
		{
			name:   "nil source leaves zero value",
			source: nil,
			want:   configTestPluginConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got configTestPluginConfig
			err := ParsePluginConfig(tt.source, &got)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParsePluginConfig_InvalidTarget(t *testing.T) {
	t.Parallel()

	var got configTestPluginConfig

	err := ParsePluginConfig(map[string]any{"enabled": true}, got)

	require.Error(t, err)
}

func TestLoadPluginConfig(t *testing.T) {
	t.Parallel()

	preParsed := configTestPluginConfig{Enabled: true, Timeout: time.Minute, Roles: []string{"admin"}}

	tests := []struct {
		name   string
		config *models.Config
		want   configTestPluginConfig
	}{
		{
			name:   "nil config leaves target untouched",
			config: nil,
			want:   configTestPluginConfig{},
		},
		{
			name:   "nil plugins leaves target untouched",
			config: &models.Config{},
			want:   configTestPluginConfig{},
		},
		{
			name: "pre-parsed typed config is assigned directly",
			config: &models.Config{
				Plugins:          models.PluginsConfig{"p": map[string]any{"enabled": false}},
				PreParsedConfigs: map[string]any{"p": preParsed},
			},
			want: preParsed,
		},
		{
			name: "pre-parsed wrong type falls back to raw config",
			config: &models.Config{
				Plugins:          models.PluginsConfig{"p": map[string]any{"enabled": true, "timeout": "2m"}},
				PreParsedConfigs: map[string]any{"p": "not-a-config"},
			},
			want: configTestPluginConfig{Enabled: true, Timeout: 2 * time.Minute},
		},
		{
			name: "nil pre-parsed value falls back to raw config",
			config: &models.Config{
				Plugins:          models.PluginsConfig{"p": map[string]any{"roles": "user"}},
				PreParsedConfigs: map[string]any{"p": nil},
			},
			want: configTestPluginConfig{Roles: []string{"user"}},
		},
		{
			name:   "missing plugin leaves target untouched",
			config: &models.Config{Plugins: models.PluginsConfig{"other": map[string]any{"enabled": true}}},
			want:   configTestPluginConfig{},
		},
		{
			name:   "nil raw config leaves target untouched",
			config: &models.Config{Plugins: models.PluginsConfig{"p": nil}},
			want:   configTestPluginConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got configTestPluginConfig
			err := LoadPluginConfig(tt.config, "p", &got)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadPluginConfig_InvalidRawConfig(t *testing.T) {
	t.Parallel()

	config := &models.Config{
		Plugins: models.PluginsConfig{"p": map[string]any{"timeout": "not-a-duration"}},
	}

	var got configTestPluginConfig
	err := LoadPluginConfig(config, "p", &got)

	require.Error(t, err)
}

func TestIsPluginEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *models.Config
		want   bool
	}{
		{name: "nil config", config: nil, want: false},
		{name: "empty config", config: &models.Config{}, want: false},
		{
			name: "pre-parsed config enabled",
			config: &models.Config{
				PreParsedConfigs: map[string]any{"p": configTestEnabledOnly{Enabled: true}},
			},
			want: true,
		},
		{
			name: "pre-parsed config disabled",
			config: &models.Config{
				PreParsedConfigs: map[string]any{"p": configTestEnabledOnly{Enabled: false}},
			},
			want: false,
		},
		{
			name: "pre-parsed config without enabled field falls through",
			config: &models.Config{
				PreParsedConfigs: map[string]any{"p": map[string]any{"timeout": "1m"}},
				Plugins:          models.PluginsConfig{"p": map[string]any{"enabled": false}},
			},
			want: false,
		},
		{
			name: "pre-parsed config wrong type falls through to raw config",
			config: &models.Config{
				PreParsedConfigs: map[string]any{"p": "unexpected"},
				Plugins:          models.PluginsConfig{"p": map[string]any{"enabled": false}},
			},
			want: false,
		},
		{
			name:   "raw config enabled true",
			config: &models.Config{Plugins: models.PluginsConfig{"p": map[string]any{"enabled": true}}},
			want:   true,
		},
		{
			name:   "raw config enabled false",
			config: &models.Config{Plugins: models.PluginsConfig{"p": map[string]any{"enabled": false}}},
			want:   false,
		},
		{
			name:   "raw config non-bool enabled defaults to enabled",
			config: &models.Config{Plugins: models.PluginsConfig{"p": map[string]any{"enabled": "yes"}}},
			want:   true,
		},
		{
			name:   "raw config without enabled defaults to enabled",
			config: &models.Config{Plugins: models.PluginsConfig{"p": map[string]any{"timeout": "1m"}}},
			want:   true,
		},
		{
			name:   "raw struct config enabled true",
			config: &models.Config{Plugins: models.PluginsConfig{"p": configTestEnabledOnly{Enabled: true}}},
			want:   true,
		},
		{
			name:   "raw struct config enabled false",
			config: &models.Config{Plugins: models.PluginsConfig{"p": configTestEnabledOnly{Enabled: false}}},
			want:   false,
		},
		{
			name:   "raw config that cannot be marshalled defaults to enabled",
			config: &models.Config{Plugins: models.PluginsConfig{"p": make(chan int)}},
			want:   true,
		},
		{
			name:   "raw config that cannot be unmarshalled defaults to enabled",
			config: &models.Config{Plugins: models.PluginsConfig{"p": "just-a-string"}},
			want:   true,
		},
		{
			name:   "raw config nil value",
			config: &models.Config{Plugins: models.PluginsConfig{"p": nil}},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsPluginEnabled(tt.config, "p")

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertRouteMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		routes []models.RouteMapping
		want   map[string]map[string]any
	}{
		{
			name:   "no routes returns empty map",
			routes: nil,
			want:   map[string]map[string]any{},
		},
		{
			name:   "bare path applies to all supported methods",
			routes: []models.RouteMapping{{Paths: []string{"/me"}, Plugins: []string{"jwt"}, Permissions: []string{"read"}}},
			want: map[string]map[string]any{
				"GET:/me":    {"plugins": []string{"jwt"}, "permissions": []string{"read"}},
				"POST:/me":   {"plugins": []string{"jwt"}, "permissions": []string{"read"}},
				"PUT:/me":    {"plugins": []string{"jwt"}, "permissions": []string{"read"}},
				"PATCH:/me":  {"plugins": []string{"jwt"}, "permissions": []string{"read"}},
				"DELETE:/me": {"plugins": []string{"jwt"}, "permissions": []string{"read"}},
			},
		},
		{
			name:   "method prefixed path",
			routes: []models.RouteMapping{{Paths: []string{" get:/users/ "}, Plugins: []string{"jwt"}}},
			want: map[string]map[string]any{
				"GET:/users": {"plugins": []string{"jwt"}, "permissions": []string{}},
			},
		},
		{
			name: "duplicate keys merge plugins and permissions",
			routes: []models.RouteMapping{
				{Paths: []string{"GET:/me"}, Plugins: []string{"jwt", "orgs"}, Permissions: []string{"read"}},
				{Paths: []string{"GET:/me"}, Plugins: []string{"orgs", "totp"}, Permissions: []string{"read", "write"}},
			},
			want: map[string]map[string]any{
				"GET:/me": {
					"plugins":     []string{"jwt", "orgs", "totp"},
					"permissions": []string{"read", "write"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ConvertRouteMetadata(tt.routes)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertRouteMetadata_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		routes []models.RouteMapping
	}{
		{
			name:   "empty paths",
			routes: []models.RouteMapping{{Plugins: []string{"jwt"}}},
		},
		{
			name:   "blank path",
			routes: []models.RouteMapping{{Paths: []string{"   "}}},
		},
		{
			name:   "missing method separator",
			routes: []models.RouteMapping{{Paths: []string{"GET"}}},
		},
		{
			name:   "empty method",
			routes: []models.RouteMapping{{Paths: []string{" :/me"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ConvertRouteMetadata(tt.routes)

			require.Error(t, err)
			assert.Nil(t, got)
		})
	}
}

func TestApplyBasePathToMetadataKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		basePath string
		want     string
	}{
		{name: "empty base path keeps key", key: "GET:/me", basePath: "", want: "GET:/me"},
		{name: "prefixes path", key: "GET:/me", basePath: "/api/auth", want: "GET:/api/auth/me"},
		{name: "trims trailing slash from base path", key: "GET:/me", basePath: "/api/auth/", want: "GET:/api/auth/me"},
		{name: "path without leading slash", key: "GET:me", basePath: "/api", want: "GET:/api/me"},
		{name: "root path is normalized", key: "GET:/", basePath: "/api", want: "GET:/api"},
		{name: "root base path and root route", key: "GET:/", basePath: "/", want: "GET:"},
		{name: "key without separator is unchanged", key: "invalid", basePath: "/api", want: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ApplyBasePathToMetadataKey(tt.key, tt.basePath)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPathDisabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		method        string
		path          string
		disabledPaths []string
		basePath      string
		want          bool
	}{
		{
			name:          "exact method and path match",
			method:        "get",
			path:          "/me",
			disabledPaths: []string{"GET:/me"},
			want:          true,
		},
		{
			name:          "method mismatch does not match",
			method:        "POST",
			path:          "/me",
			disabledPaths: []string{"GET:/me"},
			want:          false,
		},
		{
			name:          "path only entry matches any method",
			method:        "DELETE",
			path:          "/me",
			disabledPaths: []string{"/me"},
			want:          true,
		},
		{
			name:          "wildcard prefix matches nested path",
			method:        "GET",
			path:          "/organizations/123",
			disabledPaths: []string{"/organizations/*"},
			want:          true,
		},
		{
			name:          "wildcard prefix matches the prefix itself",
			method:        "GET",
			path:          "/organizations",
			disabledPaths: []string{"/organizations/*"},
			want:          true,
		},
		{
			name:          "wildcard prefix does not match sibling",
			method:        "GET",
			path:          "/organization",
			disabledPaths: []string{"/organizations/*"},
			want:          false,
		},
		{
			name:          "base path is stripped for exact match",
			method:        "GET",
			path:          "/api/auth/me",
			disabledPaths: []string{"/me"},
			basePath:      "/api/auth",
			want:          true,
		},
		{
			name:          "base path with trailing slash is normalized",
			method:        "GET",
			path:          "/api/auth/me",
			disabledPaths: []string{"/me"},
			basePath:      "/api/auth/",
			want:          true,
		},
		{
			name:          "base path is stripped for wildcard match",
			method:        "GET",
			path:          "/api/auth/orgs/1",
			disabledPaths: []string{"GET:/orgs/*"},
			basePath:      "/api/auth",
			want:          true,
		},
		{
			name:          "root wildcard matches everything",
			method:        "GET",
			path:          "/anything",
			disabledPaths: []string{"/*"},
			want:          true,
		},
		{
			name:          "blank entries are skipped",
			method:        "GET",
			path:          "/me",
			disabledPaths: []string{"", "   "},
			want:          false,
		},
		{
			name:          "entry without pattern is skipped",
			method:        "GET",
			path:          "/me",
			disabledPaths: []string{"GET:"},
			want:          false,
		},
		{
			name:          "unrelated path does not match",
			method:        "GET",
			path:          "/other",
			disabledPaths: []string{"/me", "/organizations/*"},
			want:          false,
		},
		{
			name:          "unknown entry method does not match",
			method:        "GET",
			path:          "/me",
			disabledPaths: []string{"FOO:/me"},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsPathDisabled(tt.method, tt.path, tt.disabledPaths, tt.basePath)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetEnabledFromConfig(t *testing.T) {
	t.Parallel()

	type noEnabledConfig struct {
		Timeout string `json:"timeout"`
	}

	tests := []struct {
		name      string
		config    any
		wantValue bool
		wantFound bool
	}{
		{name: "nil config", config: nil, wantValue: false, wantFound: false},
		{name: "map enabled true", config: map[string]any{"enabled": true}, wantValue: true, wantFound: true},
		{name: "map enabled false", config: map[string]any{"enabled": false}, wantValue: false, wantFound: true},
		{name: "map non-bool enabled", config: map[string]any{"enabled": "yes"}, wantValue: false, wantFound: false},
		{name: "map without enabled", config: map[string]any{"timeout": "1m"}, wantValue: false, wantFound: false},
		{name: "struct enabled true", config: configTestEnabledOnly{Enabled: true}, wantValue: true, wantFound: true},
		{name: "struct without enabled", config: noEnabledConfig{Timeout: "1m"}, wantValue: false, wantFound: false},
		{name: "struct with non-bool enabled", config: struct {
			Enabled string `json:"enabled"`
		}{Enabled: "yes"}, wantValue: false, wantFound: false},
		{name: "unmarshallable config", config: make(chan int), wantValue: false, wantFound: false},
		{name: "non-object json", config: "just-a-string", wantValue: false, wantFound: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value, found := getEnabledFromConfig(tt.config)

			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}

func TestDisabledPathMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		requestPath string
		basePath    string
		pattern     string
		want        bool
	}{
		{name: "empty pattern", requestPath: "/me", basePath: "", pattern: "", want: false},
		{name: "root wildcard matches", requestPath: "/me", basePath: "/", pattern: "/*", want: true},
		{name: "wildcard prefix match", requestPath: "/orgs/1", basePath: "", pattern: "/orgs/*", want: true},
		{name: "wildcard prefix itself", requestPath: "/orgs", basePath: "", pattern: "/orgs/*", want: true},
		{name: "wildcard match after base path strip", requestPath: "/api/orgs/1", basePath: "/api", pattern: "/orgs/*", want: true},
		{name: "wildcard base path mismatch", requestPath: "/other/orgs/1", basePath: "/api", pattern: "/orgs/*", want: false},
		{name: "wildcard base path not stripped for root", requestPath: "/orgs/1", basePath: "/", pattern: "/me/*", want: false},
		{name: "exact match", requestPath: "/me", basePath: "", pattern: "/me", want: true},
		{name: "exact match after base path strip", requestPath: "/api/me", basePath: "/api", pattern: "/me", want: true},
		{name: "exact base path mismatch", requestPath: "/other", basePath: "/api", pattern: "/me", want: false},
		{name: "exact root base path is not stripped", requestPath: "/me", basePath: "/", pattern: "/other", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, disabledPathMatches(tt.requestPath, tt.basePath, tt.pattern))
		})
	}
}
