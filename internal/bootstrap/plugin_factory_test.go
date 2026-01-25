package bootstrap

import (
	"testing"

	"github.com/GoBetterAuth/go-better-auth/models"
)

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	f()
}

func TestBuildPluginsFromConfig_ValidPlugins(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{
			models.PluginCSRF.String(): map[string]any{
				"enabled": true,
			},
		},
	}

	plugins := BuildPluginsFromConfig(cfg)

	if len(plugins) == 0 {
		t.Errorf("expected at least 1 plugin, got %d", len(plugins))
	}

	// Verify core plugin is present
	hasCorePlugin := false
	for _, p := range plugins {
		if p.Metadata().ID == models.PluginCSRF.String() {
			hasCorePlugin = true
			break
		}
	}
	if !hasCorePlugin {
		t.Errorf("core plugin not found in plugins list")
	}
}

func TestBuildPluginsFromConfig_UnknownPlugin(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{
			models.PluginCSRF.String(): map[string]any{
				"enabled": true,
			},
			"unknown_plugin": map[string]any{
				"enabled": true,
			},
		},
	}

	assertPanic(t, func() { BuildPluginsFromConfig(cfg) })
}

func TestBuildPluginsFromConfig_DisabledPlugins(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{
			models.PluginCSRF.String(): map[string]any{
				"enabled": true,
			},
			models.PluginCSRF.String(): map[string]any{
				"enabled": false,
			},
		},
	}

	plugins := BuildPluginsFromConfig(cfg)

	for _, p := range plugins {
		if p.Metadata().ID == models.PluginCSRF.String() {
			t.Errorf("csrf plugin should not be in plugins list when disabled")
		}
	}
}

func TestBuildPluginsFromConfig_PluginOrder(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{
			models.PluginConfigManager.String(): map[string]any{
				"enabled": true,
			},
			models.PluginCSRF.String(): map[string]any{
				"enabled": true,
			},
		},
	}

	plugins := BuildPluginsFromConfig(cfg)

	if len(plugins) == 0 {
		t.Fatalf("expected plugins to be present, got %d", len(plugins))
	}

	if plugins[0].Metadata().ID != models.PluginConfigManager.String() {
		t.Errorf("expected %s to be first, got %s", models.PluginConfigManager.String(), plugins[0].Metadata().ID)
	}
}

func TestBuildPluginsFromConfig_CoreDisabled(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{
			models.PluginCSRF.String(): map[string]any{
				"enabled": false,
			},
		},
	}

	plugins := BuildPluginsFromConfig(cfg)

	for _, p := range plugins {
		if p.Metadata().ID == models.PluginCSRF.String() {
			t.Errorf("csrf plugin should not be in plugins list when disabled")
		}
	}
}

func TestBuildPluginsFromConfig_EmptyConfig(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{},
	}

	plugins := BuildPluginsFromConfig(cfg)

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins for empty config, got %d", len(plugins))
	}
}
