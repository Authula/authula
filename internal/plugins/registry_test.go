package plugins

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoBetterAuth/go-better-auth/config"
	"github.com/GoBetterAuth/go-better-auth/models"
)

var (
	errInit  = errors.New("init error")
	errClose = errors.New("close error")
)

type mockPlugin struct {
	name       string
	enabled    bool
	initFails  bool
	closeFails bool
	hasRoutes  bool
	migrations []any
}

func (m *mockPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{Name: m.name}
}

func (m *mockPlugin) Config() models.PluginConfig {
	return models.PluginConfig{Enabled: m.enabled}
}

func (m *mockPlugin) Ctx() *models.PluginContext {
	return &models.PluginContext{Config: nil, EventBus: nil, Middleware: nil}
}

func (m *mockPlugin) Init(_ *models.PluginContext) error {
	if m.initFails {
		return errInit
	}
	return nil
}

func (m *mockPlugin) Migrations() []any {
	return m.migrations
}

func (m *mockPlugin) Routes() []models.PluginRoute {
	if m.hasRoutes {
		return []models.PluginRoute{
			{
				Path:   "test",
				Method: http.MethodGet,
				Handler: func() http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
				},
			},
		}
	}
	return nil
}

func (m *mockPlugin) RateLimit() *models.PluginRateLimit {
	return nil
}

func (m *mockPlugin) DatabaseHooks() *models.PluginDatabaseHooks {
	return nil
}

func (m *mockPlugin) EventHooks() *models.PluginEventHooks {
	return nil
}

func (m *mockPlugin) Close() error {
	if m.closeFails {
		return errClose
	}
	return nil
}

func getMockConfig() *models.Config {
	return config.NewConfig(
		config.WithDatabase(
			models.DatabaseConfig{
				Provider:         "sqlite",
				ConnectionString: "file::memory:?cache=shared",
			},
		),
	)
}

func TestNewPluginRegistry(t *testing.T) {
	config := getMockConfig()
	registry := NewPluginRegistry(config, nil, nil, nil)

	assert.NotNil(t, registry)
	assert.Equal(t, config, registry.config)
	assert.NotNil(t, registry.pluginCtx)
	assert.Empty(t, registry.plugins)
}

func TestPluginRegistry_Register(t *testing.T) {
	config := getMockConfig()
	registry := NewPluginRegistry(config, nil, nil, nil)

	plugin := &mockPlugin{name: "test"}
	registry.Register(plugin)

	assert.Len(t, registry.plugins, 1)
	assert.Equal(t, plugin, registry.plugins[0])
}

func TestPluginRegistry_InitAll(t *testing.T) {
	t.Run("should init enabled plugins", func(t *testing.T) {
		config := getMockConfig()
		registry := NewPluginRegistry(config, nil, nil, nil)

		plugin1 := &mockPlugin{name: "p1", enabled: true}
		plugin2 := &mockPlugin{name: "p2", enabled: false}
		registry.Register(plugin1)
		registry.Register(plugin2)

		err := registry.InitAll()
		assert.NoError(t, err)
	})

	t.Run("should return error on init fail", func(t *testing.T) {
		config := getMockConfig()
		registry := NewPluginRegistry(config, nil, nil, nil)

		plugin := &mockPlugin{name: "p1", enabled: true, initFails: true}
		registry.Register(plugin)

		err := registry.InitAll()
		assert.ErrorIs(t, err, errInit)
	})
}

func TestPluginRegistry_RunMigrations(t *testing.T) {
	type MyEntity struct {
		ID uint
	}

	t.Run("should run migrations for enabled plugins", func(t *testing.T) {
		config := getMockConfig()
		registry := NewPluginRegistry(config, nil, nil, nil)

		plugin1 := &mockPlugin{name: "p1", enabled: true, migrations: []any{&MyEntity{}}}
		plugin2 := &mockPlugin{name: "p2", enabled: false, migrations: []any{&MyEntity{}}}
		registry.Register(plugin1)
		registry.Register(plugin2)

		err := registry.RunMigrations()
		assert.NoError(t, err)
	})
}

func TestPluginRegistry_Routes(t *testing.T) {
	config := getMockConfig()
	registry := NewPluginRegistry(config, nil, nil, nil)

	plugin1 := &mockPlugin{name: "p1", enabled: true, hasRoutes: true}
	plugin2 := &mockPlugin{name: "p2", enabled: false, hasRoutes: true}
	plugin3 := &mockPlugin{name: "p3", enabled: true, hasRoutes: false}
	registry.Register(plugin1)
	registry.Register(plugin2)
	registry.Register(plugin3)

	plugins := registry.Plugins()
	routes := make([]models.PluginRoute, 0)
	for _, p := range plugins {
		if p.Config().Enabled && p.Routes() != nil {
			routes = append(routes, p.Routes()...)
		}
	}
	assert.Len(t, routes, 1)
	assert.Equal(t, "test", routes[0].Path)
}

func TestPluginRegistry_Plugins(t *testing.T) {
	config := getMockConfig()
	registry := NewPluginRegistry(config, nil, nil, nil)

	plugin1 := &mockPlugin{name: "p1", enabled: true}
	plugin2 := &mockPlugin{name: "p2", enabled: false}
	registry.Register(plugin1)
	registry.Register(plugin2)

	plugins := registry.Plugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, plugin1, plugins[0])
}

func TestPluginRegistry_CloseAll(t *testing.T) {
	t.Run("should close enabled plugins", func(t *testing.T) {
		config := getMockConfig()
		registry := NewPluginRegistry(config, nil, nil, nil)

		plugin1 := &mockPlugin{name: "p1", enabled: true}
		plugin2 := &mockPlugin{name: "p2", enabled: false}
		registry.Register(plugin1)
		registry.Register(plugin2)

		registry.CloseAll()
	})

	t.Run("should log error on close fail", func(t *testing.T) {
		config := getMockConfig()
		registry := NewPluginRegistry(config, nil, nil, nil)

		plugin := &mockPlugin{name: "p1", enabled: true, closeFails: true}
		registry.Register(plugin)

		// In a real test, you might capture log output to verify this.
		registry.CloseAll()
	})
}
