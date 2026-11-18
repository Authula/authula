package jwt

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internalplugins "github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

const testJWKSCacheKey = "jwks:public:set"

// staticSecondaryStorageService is a local fake for SecondaryStorageService so
// the shared plugins/jwt/tests package stays untouched.
type staticSecondaryStorageService struct {
	storage models.SecondaryStorage
}

func (s staticSecondaryStorageService) GetStorage() models.SecondaryStorage { return s.storage }
func (s staticSecondaryStorageService) GetProviderName() string             { return "memory" }

// failingUpdateDB delegates reads to a live database but sends updates to a
// closed one. It exercises the non-fatal key rotation error branch.
type failingUpdateDB struct {
	bun.IDB
	updateDB bun.IDB
}

func (db *failingUpdateDB) NewUpdate() *bun.UpdateQuery {
	return db.updateDB.NewUpdate()
}

func newJWTTestDB(t *testing.T) *bun.DB {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(context.Background(), []migrations.MigrationSet{
		{PluginID: models.PluginJWT.String(), Migrations: JWTMigrationsForProvider("sqlite")},
	}))

	return db
}

func newClosedJWTTestDB(t *testing.T) *bun.DB {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)
	require.NoError(t, db.Close())

	return db
}

func insertOldJWKSKey(t *testing.T, db bun.IDB, id string, age time.Duration) {
	t.Helper()

	key := &types.JWKS{
		ID:         id,
		PublicKey:  "old-public-key",
		PrivateKey: "old-private-key",
		CreatedAt:  time.Now().Add(-age),
	}
	_, err := db.NewInsert().Model(key).Exec(context.Background())
	require.NoError(t, err)
}

func countJWKSKeys(t *testing.T, db bun.IDB) int {
	t.Helper()

	count, err := db.NewSelect().Model((*types.JWKS)(nil)).Count(context.Background())
	require.NoError(t, err)

	return count
}

func newJWTTestRegistry(t *testing.T) *internalplugins.DefaultServiceRegistry {
	t.Helper()

	registry := internalplugins.NewServiceRegistry()
	registry.Register(models.ServiceSession.String(), new(jwttests.MockSessionService))
	registry.Register(models.ServiceToken.String(), jwttests.NopTokenService{})

	return registry
}

func hasLogMessage(logger *internaltests.LevelRecordingLogger, substr string) bool {
	for _, message := range logger.Messages() {
		if strings.Contains(message, substr) {
			return true
		}
	}

	return false
}

func TestJWTPlugin_Init_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  func() *models.Config
		db      func(*testing.T) *bun.DB
		reg     func(*testing.T) *internalplugins.DefaultServiceRegistry
		wantErr string
	}{
		{
			name: "load_plugin_config_error",
			config: func() *models.Config {
				return &models.Config{
					Plugins: map[string]any{
						models.PluginJWT.String(): map[string]any{"enabled": "not-a-bool"},
					},
				}
			},
			db:  newJWTTestDB,
			reg: newJWTTestRegistry,
		},
		{
			name: "blank_issuer_after_defaults",
			config: func() *models.Config {
				return &models.Config{}
			},
			db:      newJWTTestDB,
			reg:     newJWTTestRegistry,
			wantErr: "issuer must not be empty",
		},
		{
			name: "blank_audience_after_defaults",
			config: func() *models.Config {
				return &models.Config{
					BaseURL: "https://auth.example.com",
					Plugins: map[string]any{},
					PreParsedConfigs: map[string]any{
						models.PluginJWT.String(): types.JWTPluginConfig{Audience: "   "},
					},
				}
			},
			db:      newJWTTestDB,
			reg:     newJWTTestRegistry,
			wantErr: "audience must not be empty",
		},
		{
			name: "missing_session_service",
			config: func() *models.Config {
				return &models.Config{BaseURL: "https://auth.example.com"}
			},
			db:      newJWTTestDB,
			reg:     func(*testing.T) *internalplugins.DefaultServiceRegistry { return internalplugins.NewServiceRegistry() },
			wantErr: "session service not available",
		},
		{
			name: "missing_token_service",
			config: func() *models.Config {
				return &models.Config{BaseURL: "https://auth.example.com"}
			},
			db: newJWTTestDB,
			reg: func(t *testing.T) *internalplugins.DefaultServiceRegistry {
				registry := internalplugins.NewServiceRegistry()
				registry.Register(models.ServiceSession.String(), new(jwttests.MockSessionService))
				return registry
			},
			wantErr: "token service not available",
		},
		{
			name: "key_generation_failure",
			config: func() *models.Config {
				return &models.Config{BaseURL: "https://auth.example.com"}
			},
			db:      newClosedJWTTestDB,
			reg:     newJWTTestRegistry,
			wantErr: "failed to generate keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getConfig := tt.config
			plugin := New(types.JWTPluginConfig{})
			ctx := &models.PluginContext{
				DB:              tt.db(t),
				Logger:          &internaltests.MockLogger{},
				EventBus:        internaltests.NewRecordingEventBus(),
				ServiceRegistry: tt.reg(t),
				GetConfig:       func() *models.Config { return getConfig() },
			}

			err := plugin.Init(ctx)

			require.Error(t, err)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			}
			assert.Nil(t, plugin.jwtService)
		})
	}
}

func TestJWTPlugin_Init_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		baseURL              string
		basePath             string
		withSecondaryStorage bool
		wantIssuer           string
	}{
		{
			name:       "without_secondary_storage",
			baseURL:    "https://auth.example.com/",
			basePath:   "/api/auth/",
			wantIssuer: "https://auth.example.com/api/auth",
		},
		{
			name:                 "with_secondary_storage",
			baseURL:              "https://auth.example.com",
			withSecondaryStorage: true,
			wantIssuer:           "https://auth.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newJWTTestDB(t)
			logger := &internaltests.LevelRecordingLogger{}
			storage := jwttests.NewInMemoryStorage()
			registry := newJWTTestRegistry(t)
			if tt.withSecondaryStorage {
				registry.Register(models.ServiceSecondaryStorage.String(), staticSecondaryStorageService{storage: storage})
			}
			cfg := &models.Config{BaseURL: tt.baseURL, BasePath: tt.basePath}
			plugin := New(types.JWTPluginConfig{})
			ctx := &models.PluginContext{
				DB:              db,
				Logger:          logger,
				EventBus:        internaltests.NewRecordingEventBus(),
				ServiceRegistry: registry,
				GetConfig:       func() *models.Config { return cfg },
			}

			require.NoError(t, plugin.Init(ctx))

			assert.Same(t, logger, plugin.Logger)
			assert.NotNil(t, plugin.sessionService)
			assert.NotNil(t, plugin.tokenService)
			assert.NotNil(t, plugin.keyService)
			assert.NotNil(t, plugin.cacheService)
			assert.NotNil(t, plugin.jwtService)
			assert.NotNil(t, plugin.refreshService)
			assert.Equal(t, tt.wantIssuer, plugin.pluginConfig.Issuer)
			assert.Equal(t, types.DefaultAudience, plugin.pluginConfig.Audience)
			assert.Same(t, plugin.jwtService, registry.Get(models.ServiceJWT.String()))
			assert.Equal(t, 1, countJWKSKeys(t, db))

			if tt.withSecondaryStorage {
				assert.NotNil(t, plugin.secondaryStorage)
				assert.NotNil(t, plugin.blacklistService)

				cached, err := storage.Get(context.Background(), testJWKSCacheKey)
				require.NoError(t, err)
				assert.NotNil(t, cached)
			} else {
				assert.Nil(t, plugin.secondaryStorage)
				assert.Nil(t, plugin.blacklistService)
				assert.True(t, hasLogMessage(logger, "token blacklisting will be disabled"))
			}
		})
	}
}

func TestJWTPlugin_Init_KeyRotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		breakKeyUpdate bool
		wantKeyCount   int
		wantLogMessage string
	}{
		{
			name:           "rotation_due_rotates_keys",
			wantKeyCount:   2,
			wantLogMessage: "key rotation occurred on startup due to interval expiration",
		},
		{
			name:           "rotation_failure_is_a_warning",
			breakKeyUpdate: true,
			wantKeyCount:   1,
			wantLogMessage: "failed to check/rotate keys on startup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newJWTTestDB(t)
			insertOldJWKSKey(t, db, "old-key", 48*time.Hour)
			var pluginDB bun.IDB = db
			if tt.breakKeyUpdate {
				pluginDB = &failingUpdateDB{IDB: db, updateDB: newClosedJWTTestDB(t)}
			}
			logger := &internaltests.LevelRecordingLogger{}
			plugin := New(types.JWTPluginConfig{KeyRotationInterval: time.Hour})
			ctx := &models.PluginContext{
				DB:              pluginDB,
				Logger:          logger,
				EventBus:        internaltests.NewRecordingEventBus(),
				ServiceRegistry: newJWTTestRegistry(t),
				GetConfig:       func() *models.Config { return &models.Config{BaseURL: "https://auth.example.com"} },
			}

			require.NoError(t, plugin.Init(ctx))

			assert.NotNil(t, plugin.jwtService)
			assert.Equal(t, tt.wantKeyCount, countJWKSKeys(t, db))
			assert.True(t, hasLogMessage(logger, tt.wantLogMessage))
		})
	}
}

func TestJWTPlugin_Init_CacheInvalidationFailureIsAWarning(t *testing.T) {
	t.Parallel()

	db := newJWTTestDB(t)
	errCache := assert.AnError
	storage := new(internaltests.MockSecondaryStorage)
	storage.On("Delete", mock.Anything, testJWKSCacheKey).Return(errCache)

	registry := newJWTTestRegistry(t)
	registry.Register(models.ServiceSecondaryStorage.String(), staticSecondaryStorageService{storage: storage})
	logger := &internaltests.LevelRecordingLogger{}
	plugin := New(types.JWTPluginConfig{})
	ctx := &models.PluginContext{
		DB:              db,
		Logger:          logger,
		EventBus:        internaltests.NewRecordingEventBus(),
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return &models.Config{BaseURL: "https://auth.example.com"} },
	}

	require.NoError(t, plugin.Init(ctx))

	assert.NotNil(t, plugin.blacklistService)
	assert.NotNil(t, plugin.jwtService)
	assert.True(t, hasLogMessage(logger, "failed to pre-populate cache on startup"))
	storage.AssertExpectations(t)
}

func TestJWTPlugin_Migrations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		provider string
		wantLen  int
	}{
		{provider: "sqlite", wantLen: 1},
		{provider: "postgres", wantLen: 1},
		{provider: " MySQL ", wantLen: 1},
		{provider: "oracle", wantLen: 0},
		{provider: "", wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			t.Parallel()

			plugin := New(types.JWTPluginConfig{})
			got := plugin.Migrations(tt.provider)
			want := JWTMigrationsForProvider(tt.provider)

			require.Len(t, got, tt.wantLen)
			require.Len(t, want, tt.wantLen)
			for i := range want {
				assert.Equal(t, want[i].Version, got[i].Version)
				assert.NotNil(t, got[i].Up)
				assert.NotNil(t, got[i].Down)
			}
		})
	}
}

func TestJWTPlugin_DependsOn(t *testing.T) {
	t.Parallel()

	plugin := New(types.JWTPluginConfig{})

	assert.Nil(t, plugin.DependsOn())
}

func TestJWTPlugin_Hooks(t *testing.T) {
	t.Parallel()

	plugin := New(types.JWTPluginConfig{})

	hooks := plugin.Hooks()

	require.Len(t, hooks, 2)

	assert.Equal(t, models.HookAfter, hooks[0].Stage)
	assert.Equal(t, 15, hooks[0].Order)
	assert.Empty(t, hooks[0].PluginID)
	require.NotNil(t, hooks[0].Matcher)
	require.NotNil(t, hooks[0].Handler)

	assert.Equal(t, models.HookOnResponse, hooks[1].Stage)
	assert.Equal(t, HookIDJWTRespondJSON.String(), hooks[1].PluginID)
	assert.Equal(t, 10, hooks[1].Order)
	assert.NotNil(t, hooks[1].Handler)
}

func TestJWTPlugin_Close(t *testing.T) {
	t.Parallel()

	plugin := New(types.JWTPluginConfig{})

	assert.NoError(t, plugin.Close())
}
