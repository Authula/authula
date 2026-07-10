package bootstrap

import (
	"testing"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/util"
)

func assertPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()

	f()
}

func pluginIDs(plugins []models.Plugin) []string {
	ids := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		ids = append(ids, plugin.Metadata().ID)
	}

	return ids
}

func TestBuildPluginsFromConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *models.Config
		wantIDs   []string
		wantPanic bool
	}{
		{
			name: "enabled plugin is included",
			cfg: &models.Config{
				Plugins: map[string]any{
					models.PluginCSRF.String(): map[string]any{
						"enabled": true,
					},
				},
			},
			wantIDs: []string{models.PluginCSRF.String()},
		},
		{
			name: "unknown plugin panics",
			cfg: &models.Config{
				Plugins: map[string]any{
					models.PluginCSRF.String(): map[string]any{
						"enabled": true,
					},
					"unknown_plugin": map[string]any{
						"enabled": true,
					},
				},
			},
			wantPanic: true,
		},
		{
			name: "disabled plugin is omitted",
			cfg: &models.Config{
				Plugins: map[string]any{
					models.PluginCSRF.String(): map[string]any{
						"enabled": false,
					},
				},
			},
			wantIDs: []string{},
		},
		{
			name: "plugin order follows registry order",
			cfg: &models.Config{
				Plugins: map[string]any{
					models.PluginAdmin.String(): map[string]any{
						"enabled": true,
					},
					models.PluginCSRF.String(): map[string]any{
						"enabled": true,
					},
				},
			},
			wantIDs: []string{models.PluginAdmin.String(), models.PluginCSRF.String()},
		},
		{
			name: "empty config returns no plugins",
			cfg: &models.Config{
				Plugins: map[string]any{},
			},
			wantIDs: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantPanic {
				assertPanic(t, func() {
					BuildPluginsFromConfig(tc.cfg)
				})

				return
			}

			plugins := BuildPluginsFromConfig(tc.cfg)
			gotIDs := pluginIDs(plugins)

			if !util.CompareStringArrays(gotIDs, tc.wantIDs) {
				t.Fatalf("unexpected plugin IDs: got %v, want %v", gotIDs, tc.wantIDs)
			}
		})
	}
}
