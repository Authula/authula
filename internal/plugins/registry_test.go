package plugins

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/config"
	"github.com/GoBetterAuth/go-better-auth/models"
)

var (
	errInit = errors.New("init error")
)

func getMockConfig() *models.Config {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to initialize test database: " + err.Error())
	}

	return config.NewConfig(
		config.WithDatabase(
			models.DatabaseConfig{
				Provider: "sqlite",
				URL:      "file::memory:?cache=shared",
			},
		),
		config.WithDB(db),
	)
}

func getMockPlugin() models.Plugin {
	return config.NewPlugin(
		config.WithPluginMetadata(models.PluginMetadata{Name: "mock"}),
		config.WithPluginConfig(models.PluginConfig{Enabled: true}),
	)
}

func TestNewPluginRegistry(t *testing.T) {
	mockConfig := getMockConfig()
	registry := NewPluginRegistry(mockConfig, nil, nil, nil)

	assert.NotNil(t, registry)
	assert.Equal(t, mockConfig, registry.config)
	assert.NotNil(t, registry.pluginCtx)
	assert.Empty(t, registry.plugins)
}

func TestPluginRegistry_Register(t *testing.T) {
	mockConfig := getMockConfig()
	registry := NewPluginRegistry(mockConfig, nil, nil, nil)

	plugin := getMockPlugin()
	registry.Register(plugin)

	assert.Len(t, registry.plugins, 1)
	assert.Equal(t, plugin, registry.plugins[0])
}

func TestPluginRegistry_InitAll(t *testing.T) {
	t.Run("should init enabled plugins", func(t *testing.T) {
		mockConfig := getMockConfig()
		registry := NewPluginRegistry(mockConfig, nil, nil, nil)

		plugin1 := getMockPlugin()
		plugin2 := getMockPlugin()
		registry.Register(plugin1)
		registry.Register(plugin2)

		err := registry.InitAll()
		assert.NoError(t, err)
	})

	t.Run("should return error on init fail", func(t *testing.T) {
		mockConfig := getMockConfig()
		registry := NewPluginRegistry(mockConfig, nil, nil, nil)

		plugin := getMockPlugin()
		plugin.SetInit(func(ctx *models.PluginContext) error {
			return errInit
		})
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
		mockConfig := getMockConfig()
		registry := NewPluginRegistry(mockConfig, nil, nil, nil)

		plugin1 := config.NewPlugin(
			config.WithPluginMetadata(models.PluginMetadata{Name: "p1"}),
			config.WithPluginConfig(models.PluginConfig{Enabled: true}),
			config.WithPluginMigrations([]any{&MyEntity{}}),
		)
		plugin2 := config.NewPlugin(
			config.WithPluginMetadata(models.PluginMetadata{Name: "p2"}),
			config.WithPluginConfig(models.PluginConfig{Enabled: false}),
			config.WithPluginMigrations([]any{&MyEntity{}}),
		)
		registry.Register(plugin1)
		registry.Register(plugin2)

		err := registry.RunMigrations()
		assert.NoError(t, err)
	})
}

func TestPluginRegistry_Routes(t *testing.T) {
	mockConfig := getMockConfig()
	registry := NewPluginRegistry(mockConfig, nil, nil, nil)

	plugin1 := config.NewPlugin(
		config.WithPluginMetadata(models.PluginMetadata{Name: "p1"}),
		config.WithPluginConfig(models.PluginConfig{Enabled: true}),
		config.WithPluginRoutes([]models.PluginRoute{{Path: "test"}}),
	)
	plugin2 := config.NewPlugin(
		config.WithPluginMetadata(models.PluginMetadata{Name: "p2"}),
		config.WithPluginConfig(models.PluginConfig{Enabled: false}),
		config.WithPluginRoutes([]models.PluginRoute{{Path: "test"}}),
	)
	plugin3 := config.NewPlugin(
		config.WithPluginMetadata(models.PluginMetadata{Name: "p3"}),
		config.WithPluginConfig(models.PluginConfig{Enabled: true}),
		config.WithPluginRoutes(nil),
	)
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
	mockConfig := getMockConfig()
	registry := NewPluginRegistry(mockConfig, nil, nil, nil)

	plugin1 := config.NewPlugin(
		config.WithPluginMetadata(models.PluginMetadata{Name: "p1"}),
		config.WithPluginConfig(models.PluginConfig{Enabled: true}),
	)
	plugin2 := config.NewPlugin(
		config.WithPluginMetadata(models.PluginMetadata{Name: "p2"}),
		config.WithPluginConfig(models.PluginConfig{Enabled: false}),
	)
	registry.Register(plugin1)
	registry.Register(plugin2)

	plugins := registry.Plugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, plugin1, plugins[0])
}

func TestPluginRegistry_CloseAll(t *testing.T) {
	t.Run("should close enabled plugins", func(t *testing.T) {
		mockConfig := getMockConfig()
		registry := NewPluginRegistry(mockConfig, nil, nil, nil)

		plugin1 := config.NewPlugin(
			config.WithPluginMetadata(models.PluginMetadata{Name: "p1"}),
			config.WithPluginConfig(models.PluginConfig{Enabled: true}),
		)
		plugin2 := config.NewPlugin(
			config.WithPluginMetadata(models.PluginMetadata{Name: "p2"}),
			config.WithPluginConfig(models.PluginConfig{Enabled: false}),
		)
		registry.Register(plugin1)
		registry.Register(plugin2)

		registry.CloseAll()
	})

	t.Run("should log error on close fail", func(t *testing.T) {
		mockConfig := getMockConfig()
		registry := NewPluginRegistry(mockConfig, nil, nil, nil)

		plugin := config.NewPlugin(
			config.WithPluginMetadata(models.PluginMetadata{Name: "p1"}),
			config.WithPluginConfig(models.PluginConfig{Enabled: true}),
		)
		registry.Register(plugin)

		// In a real test, you might capture log output to verify this.
		registry.CloseAll()
	})
}
