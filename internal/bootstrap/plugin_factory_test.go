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
			models.PluginCore.String(): map[string]any{
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
		if p.Metadata().ID == models.PluginCore.String() {
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
			models.PluginCore.String(): map[string]any{
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
			models.PluginCore.String(): map[string]any{
				"enabled": true,
			},
			models.PluginCORS.String(): map[string]any{
				"enabled": false,
			},
		},
	}

	plugins := BuildPluginsFromConfig(cfg)

	// Verify cors plugin is not present (disabled)
	for _, p := range plugins {
		if p.Metadata().ID == models.PluginCORS.String() {
			t.Errorf("cors plugin should not be in plugins list when disabled")
		}
	}
}

func TestBuildPluginsFromConfig_PluginOrder(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{
			models.PluginCore.String(): map[string]any{
				"enabled": true,
			},
			models.PluginConfigManager.String(): map[string]any{
				"enabled": true,
			},
			models.PluginCORS.String(): map[string]any{
				"enabled": true,
			},
		},
	}

	plugins := BuildPluginsFromConfig(cfg)

	// Verify core is first
	if len(plugins) > 0 && plugins[0].Metadata().ID != models.PluginCore.String() {
		t.Errorf("expected core plugin to be first, got %s", plugins[0].Metadata().ID)
	}
}

func TestBuildPluginsFromConfig_CoreDisabled(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{
			models.PluginCore.String(): map[string]any{
				"enabled": false,
			},
		},
	}

	assertPanic(t, func() { BuildPluginsFromConfig(cfg) })
}

func TestBuildPluginsFromConfig_EmptyConfig(t *testing.T) {
	cfg := &models.Config{
		Plugins: map[string]any{},
	}

	plugins := BuildPluginsFromConfig(cfg)

	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin for empty config, got %d", len(plugins))
	}

	if len(plugins) > 0 && plugins[0].Metadata().ID != models.PluginCore.String() {
		t.Errorf("expected core plugin, got %s", plugins[0].Metadata().ID)
	}
}
