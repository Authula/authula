package secondarystorage

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/env"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

// recordingServiceRegistry records registrations so tests can assert that the
// plugin published its service under the expected name.
type recordingServiceRegistry struct {
	mu       sync.Mutex
	services map[string]any
}

func newRecordingServiceRegistry() *recordingServiceRegistry {
	return &recordingServiceRegistry{services: make(map[string]any)}
}

func (r *recordingServiceRegistry) Register(name string, service any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[name] = service
}

func (r *recordingServiceRegistry) Get(name string) any {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.services[name]
}

func newTestPluginContext(registry models.ServiceRegistry, logger models.Logger, db bun.IDB) *models.PluginContext {
	return &models.PluginContext{
		DB:              db,
		Logger:          logger,
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return &models.Config{} },
	}
}

func TestSecondaryStoragePluginMetadata(t *testing.T) {
	t.Parallel()

	plugin := New(SecondaryStoragePluginConfig{})

	metadata := plugin.Metadata()
	assert.Equal(t, models.PluginSecondaryStorage.String(), metadata.ID)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)
	assert.Nil(t, plugin.DependsOn())
}

func TestNewSecondaryStoragePluginConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       SecondaryStoragePluginConfig
		wantProvider SecondaryStorageProvider
		wantEnabled  bool
	}{
		{
			name:         "empty config applies memory provider default",
			config:       SecondaryStoragePluginConfig{},
			wantProvider: SecondaryStorageProviderMemory,
		},
		{
			name:         "explicit provider and enabled flag are preserved",
			config:       SecondaryStoragePluginConfig{Provider: SecondaryStorageProviderDatabase, Enabled: true},
			wantProvider: SecondaryStorageProviderDatabase,
			wantEnabled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tt.config)

			config, ok := plugin.Config().(SecondaryStoragePluginConfig)
			require.True(t, ok)
			assert.Equal(t, tt.wantProvider, config.Provider)
			assert.Equal(t, tt.wantEnabled, config.Enabled)
		})
	}
}

func TestNewWithStorage(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})

	plugin := NewWithStorage("custom", storage)

	config, ok := plugin.Config().(SecondaryStoragePluginConfig)
	require.True(t, ok)
	assert.Equal(t, SecondaryStorageProvider("custom"), config.Provider)
	assert.Same(t, storage, plugin.storage)
}

func TestSecondaryStoragePluginInitMemoryProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		config           SecondaryStoragePluginConfig
		wantInterval     time.Duration
		wantProviderName string
	}{
		{
			name:             "default provider falls back to memory",
			config:           SecondaryStoragePluginConfig{},
			wantInterval:     time.Minute,
			wantProviderName: "memory",
		},
		{
			name: "memory config controls cleanup interval",
			config: SecondaryStoragePluginConfig{
				Provider: SecondaryStorageProviderMemory,
				Memory:   &MemoryStorageConfig{CleanupInterval: 5 * time.Second},
			},
			wantInterval:     5 * time.Second,
			wantProviderName: "memory",
		},
		{
			name: "zero memory cleanup interval uses default",
			config: SecondaryStoragePluginConfig{
				Provider: SecondaryStorageProviderMemory,
				Memory:   &MemoryStorageConfig{},
			},
			wantInterval:     time.Minute,
			wantProviderName: "memory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := newRecordingServiceRegistry()
			plugin := New(tt.config)

			err := plugin.Init(newTestPluginContext(registry, &internaltests.LevelRecordingLogger{}, nil))
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, plugin.Close())
			})

			registered, ok := registry.Get(models.ServiceSecondaryStorage.String()).(*SecondaryStorageServiceImpl)
			require.True(t, ok, "plugin must register the secondary storage service")
			assert.Equal(t, tt.wantProviderName, registered.GetProviderName())

			memory, ok := registered.GetStorage().(*MemorySecondaryStorage)
			require.True(t, ok)
			assert.Equal(t, tt.wantInterval, memory.cleanupInterval)
		})
	}
}

func TestSecondaryStoragePluginInitDatabaseProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       SecondaryStoragePluginConfig
		wantInterval time.Duration
	}{
		{
			name:         "missing database config uses defaults",
			config:       SecondaryStoragePluginConfig{Provider: SecondaryStorageProviderDatabase},
			wantInterval: time.Minute,
		},
		{
			name: "database config controls cleanup interval",
			config: SecondaryStoragePluginConfig{
				Provider: SecondaryStorageProviderDatabase,
				Database: &DatabaseStorageConfig{CleanupInterval: 30 * time.Second},
			},
			wantInterval: 30 * time.Second,
		},
		{
			name: "zero database cleanup interval uses default",
			config: SecondaryStoragePluginConfig{
				Provider: SecondaryStorageProviderDatabase,
				Database: &DatabaseStorageConfig{},
			},
			wantInterval: time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := internaltests.NewSQLiteIntegrationDB(t)
			_, err := db.NewCreateTable().Model(&KeyValueStore{}).Exec(context.Background())
			require.NoError(t, err)

			registry := newRecordingServiceRegistry()
			plugin := New(tt.config)

			err = plugin.Init(newTestPluginContext(registry, &internaltests.LevelRecordingLogger{}, db))
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, plugin.Close())
			})

			registered, ok := registry.Get(models.ServiceSecondaryStorage.String()).(*SecondaryStorageServiceImpl)
			require.True(t, ok)
			assert.Equal(t, "database", registered.GetProviderName())

			storage, ok := registered.GetStorage().(*DatabaseSecondaryStorage)
			require.True(t, ok)
			assert.Equal(t, tt.wantInterval, storage.cleanupInterval)

			require.NoError(t, storage.Set(context.Background(), "plugin-key", "plugin-value", nil))
			value, err := storage.Get(context.Background(), "plugin-key")
			require.NoError(t, err)
			assert.Equal(t, "plugin-value", value)
		})
	}
}

func TestSecondaryStoragePluginInitRedisProvider(t *testing.T) {
	server := miniredis.RunT(t)

	tests := []struct {
		name   string
		envURL string
		config *RedisStorageConfig
	}{
		{
			name:   "config url is used when environment url is empty",
			envURL: "",
			config: &RedisStorageConfig{URL: "redis://" + server.Addr(), MaxRetries: -1},
		},
		{
			name:   "environment url is used when config url is empty",
			envURL: "redis://" + server.Addr(),
			config: &RedisStorageConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(env.EnvRedisURL, tt.envURL)

			registry := newRecordingServiceRegistry()
			plugin := New(SecondaryStoragePluginConfig{
				Provider: SecondaryStorageProviderRedis,
				Redis:    tt.config,
			})

			err := plugin.Init(newTestPluginContext(registry, &internaltests.LevelRecordingLogger{}, nil))
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, plugin.Close())
			})

			registered, ok := registry.Get(models.ServiceSecondaryStorage.String()).(*SecondaryStorageServiceImpl)
			require.True(t, ok)
			assert.Equal(t, "redis", registered.GetProviderName())

			storage, ok := registered.GetStorage().(*RedisSecondaryStorage)
			require.True(t, ok)

			require.NoError(t, storage.Set(context.Background(), "plugin-key", "plugin-value", nil))
			value, err := storage.Get(context.Background(), "plugin-key")
			require.NoError(t, err)
			assert.Equal(t, "plugin-value", value)
		})
	}
}

func TestSecondaryStoragePluginInitRedisFallbackToMemory(t *testing.T) {
	tests := []struct {
		name string
		cfg  *RedisStorageConfig
	}{
		{
			name: "missing config and environment url",
			cfg:  nil,
		},
		{
			name: "empty config url",
			cfg:  &RedisStorageConfig{},
		},
		{
			name: "invalid url",
			cfg:  &RedisStorageConfig{URL: "://not-a-url"},
		},
		{
			name: "unreachable server",
			cfg:  &RedisStorageConfig{URL: "redis://127.0.0.1:1", MaxRetries: -1, PoolTimeout: 100 * time.Millisecond},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(env.EnvRedisURL, "")

			logger := new(internaltests.LevelRecordingLogger)
			registry := newRecordingServiceRegistry()
			plugin := New(SecondaryStoragePluginConfig{
				Provider: SecondaryStorageProviderRedis,
				Redis:    tt.cfg,
			})

			err := plugin.Init(newTestPluginContext(registry, logger, nil))
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, plugin.Close())
			})

			assert.Contains(t, logger.Messages(), "failed to initialize Redis provider, falling back to memory")

			registered, ok := registry.Get(models.ServiceSecondaryStorage.String()).(*SecondaryStorageServiceImpl)
			require.True(t, ok)
			assert.Equal(t, "redis", registered.GetProviderName())

			memory, ok := registered.GetStorage().(*MemorySecondaryStorage)
			require.True(t, ok, "redis failures must fall back to memory storage")
			require.NoError(t, memory.Set(context.Background(), "key", "value", nil))
		})
	}
}

func TestSecondaryStoragePluginInitLoadsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       *models.Config
		wantProvider SecondaryStorageProvider
		wantInterval time.Duration
		wantMemory   bool
	}{
		{
			name: "config map is decoded",
			config: &models.Config{
				Plugins: models.PluginsConfig{
					models.PluginSecondaryStorage.String(): map[string]any{
						"provider": "database",
						"database": map[string]any{"cleanup_interval": "45s"},
					},
				},
			},
			wantProvider: SecondaryStorageProviderDatabase,
			wantInterval: 45 * time.Second,
		},
		{
			name: "pre-parsed config wins",
			config: &models.Config{
				Plugins: models.PluginsConfig{},
				PreParsedConfigs: map[string]any{
					models.PluginSecondaryStorage.String(): SecondaryStoragePluginConfig{
						Provider: SecondaryStorageProviderDatabase,
						Database: &DatabaseStorageConfig{CleanupInterval: 15 * time.Second},
					},
				},
			},
			wantProvider: SecondaryStorageProviderDatabase,
			wantInterval: 15 * time.Second,
		},
		{
			name: "missing plugin config keeps constructor defaults",
			config: &models.Config{
				Plugins: models.PluginsConfig{},
			},
			wantProvider: SecondaryStorageProviderMemory,
			wantInterval: time.Minute,
			wantMemory:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := internaltests.NewSQLiteIntegrationDB(t)
			registry := newRecordingServiceRegistry()
			plugin := New(SecondaryStoragePluginConfig{})

			ctx := newTestPluginContext(registry, &internaltests.LevelRecordingLogger{}, db)
			ctx.GetConfig = func() *models.Config { return tt.config }

			require.NoError(t, plugin.Init(ctx))
			t.Cleanup(func() {
				assert.NoError(t, plugin.Close())
			})

			assert.Equal(t, tt.wantProvider, plugin.config.Provider)

			registered, ok := registry.Get(models.ServiceSecondaryStorage.String()).(*SecondaryStorageServiceImpl)
			require.True(t, ok)
			assert.Equal(t, tt.wantProvider.String(), registered.GetProviderName())

			if tt.wantMemory {
				memory, ok := registered.GetStorage().(*MemorySecondaryStorage)
				require.True(t, ok)
				assert.Equal(t, tt.wantInterval, memory.cleanupInterval)
				return
			}

			storage, ok := registered.GetStorage().(*DatabaseSecondaryStorage)
			require.True(t, ok)
			assert.Equal(t, tt.wantInterval, storage.cleanupInterval)
		})
	}
}

func TestSecondaryStoragePluginInitConfigError(t *testing.T) {
	t.Parallel()

	registry := newRecordingServiceRegistry()
	plugin := New(SecondaryStoragePluginConfig{})

	ctx := newTestPluginContext(registry, &internaltests.LevelRecordingLogger{}, nil)
	ctx.GetConfig = func() *models.Config {
		return &models.Config{
			Plugins: models.PluginsConfig{
				models.PluginSecondaryStorage.String(): map[string]any{
					"memory": []any{"not-a-config"},
				},
			},
		}
	}

	err := plugin.Init(ctx)

	require.Error(t, err)
	assert.Nil(t, registry.Get(models.ServiceSecondaryStorage.String()))
}

func TestSecondaryStoragePluginInitWithCustomStorageIsNoOp(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})

	registry := newRecordingServiceRegistry()
	plugin := NewWithStorage("custom", storage)

	err := plugin.Init(newTestPluginContext(registry, &internaltests.LevelRecordingLogger{}, nil))
	require.NoError(t, err)

	assert.Same(t, storage, plugin.storage)
}

func TestSecondaryStoragePluginMigrations(t *testing.T) {
	t.Parallel()

	validProviders := []string{"sqlite", "postgres", "mysql"}

	tests := []struct {
		name         string
		provider     SecondaryStorageProvider
		query        string
		wantVersions bool
	}{
		{name: "database sqlite", provider: SecondaryStorageProviderDatabase, query: "sqlite", wantVersions: true},
		{name: "database postgres", provider: SecondaryStorageProviderDatabase, query: "postgres", wantVersions: true},
		{name: "database mysql", provider: SecondaryStorageProviderDatabase, query: "mysql", wantVersions: true},
		{name: "database unknown provider", provider: SecondaryStorageProviderDatabase, query: "oracle"},
		{name: "memory has no migrations", provider: SecondaryStorageProviderMemory, query: "sqlite"},
		{name: "redis has no migrations", provider: SecondaryStorageProviderRedis, query: "sqlite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(SecondaryStoragePluginConfig{Provider: tt.provider})

			got := plugin.Migrations(tt.query)

			if !tt.wantVersions {
				assert.Empty(t, got)
				return
			}
			require.Len(t, got, 1)
			assert.Equal(t, "20260129000000_secondary_storage_initial", got[0].Version)
			assert.NotNil(t, got[0].Up)
			assert.NotNil(t, got[0].Down)
		})
	}

	t.Run("every supported provider yields a migration", func(t *testing.T) {
		t.Parallel()

		plugin := New(SecondaryStoragePluginConfig{Provider: SecondaryStorageProviderDatabase})
		for _, provider := range validProviders {
			assert.Len(t, plugin.Migrations(provider), 1, "provider %s", provider)
		}
	})
}

func TestSecondaryStoragePluginClose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		newPlugin  func(t *testing.T, storage *internaltests.MockSecondaryStorage) *SecondaryStoragePlugin
		closeCalls int
		wantErr    string
	}{
		{
			name: "without storage",
			newPlugin: func(t *testing.T, _ *internaltests.MockSecondaryStorage) *SecondaryStoragePlugin {
				return New(SecondaryStoragePluginConfig{})
			},
			closeCalls: 1,
		},
		{
			name: "propagates storage close errors",
			newPlugin: func(t *testing.T, storage *internaltests.MockSecondaryStorage) *SecondaryStoragePlugin {
				storage.On("Close").Return(errors.New("close failed")).Once()
				return NewWithStorage("custom", storage)
			},
			closeCalls: 1,
			wantErr:    "close failed",
		},
		{
			name: "closes initialized storage and is idempotent",
			newPlugin: func(t *testing.T, _ *internaltests.MockSecondaryStorage) *SecondaryStoragePlugin {
				plugin := New(SecondaryStoragePluginConfig{Provider: SecondaryStorageProviderMemory})
				require.NoError(t, plugin.Init(newTestPluginContext(newRecordingServiceRegistry(), &internaltests.LevelRecordingLogger{}, nil)))
				return plugin
			},
			closeCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := new(internaltests.MockSecondaryStorage)
			plugin := tt.newPlugin(t, storage)

			var err error
			for range tt.closeCalls {
				if err = plugin.Close(); err != nil {
					break
				}
			}

			storage.AssertExpectations(t)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
