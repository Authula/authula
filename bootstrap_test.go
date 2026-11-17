package authula

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/config"
	"github.com/Authula/authula/env"
	"github.com/Authula/authula/events"
	internalplugins "github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	serviceinterfaces "github.com/Authula/authula/services"
)

// newTestConfig returns the default config with a valid test secret. It does
// not read environment variables, so tests that use it can run in parallel.
func newTestConfig() *models.Config {
	cfg := config.NewDefaultConfig()
	cfg.SecretKey = internaltests.MustDecodeKey(internaltests.EncryptionKeyDevDefault)

	return cfg
}

// Not parallel: t.Setenv changes process-wide environment variables and
// panics when used by a test that calls t.Parallel.
func TestInitLogger_DefaultLogger(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	tests := []struct {
		name  string
		level string
		goEnv string
	}{
		{name: "debug level", level: "debug"},
		{name: "info level", level: "info"},
		{name: "warn level", level: "warn"},
		{name: "error level", level: "error"},
		{name: "unknown level falls back to info", level: "verbose"},
		{name: "production uses json handler", level: "warn", goEnv: "production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(env.EnvGoEnvironment, tt.goEnv)

			cfg := &models.Config{Logger: models.LoggerConfig{Level: tt.level}}
			logger := InitLogger(cfg)
			require.NotNil(t, logger)

			fanout, ok := logger.(models.LoggerFanout)
			require.True(t, ok, "InitLogger must return a fan-out logger, got %T", logger)

			recorder := new(internaltests.LevelRecordingLogger)
			detach := fanout.AddLogger(recorder)
			t.Cleanup(detach)

			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			records := recorder.Records()
			require.Len(t, records, 4)
			assert.Equal(t, []string{"debug", "info", "warn", "error"}, []string{
				records[0].Level,
				records[1].Level,
				records[2].Level,
				records[3].Level,
			})
		})
	}
}

func TestInitLogger_CustomLoggerIsWrapped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		level   string
		message string
	}{
		{
			name:    "custom logger receives info messages",
			level:   "debug",
			message: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			original := slog.Default()
			base := new(internaltests.LevelRecordingLogger)
			cfg := &models.Config{Logger: models.LoggerConfig{Level: tt.level, Logger: base}}

			logger := InitLogger(cfg)
			require.NotNil(t, logger)
			assert.NotSame(t, base, logger, "the configured logger must be wrapped in a fan-out")

			fanout, ok := logger.(models.LoggerFanout)
			require.True(t, ok, "InitLogger must return a fan-out logger, got %T", logger)

			sink := new(internaltests.LevelRecordingLogger)
			detach := fanout.AddLogger(sink)
			t.Cleanup(detach)

			logger.Info(tt.message, "key", "value")

			assert.Equal(t, []string{tt.message}, base.Messages())
			assert.Equal(t, []string{tt.message}, sink.Messages())
			assert.Equal(t, original, slog.Default(), "a custom logger must not replace the slog default")
		})
	}
}

type closeableDB interface {
	Close() error
}

// Not parallel: the cases set AUTHULA_DATABASE_URL with t.Setenv, which
// changes process-wide environment variables.
func TestInitDatabase(t *testing.T) {
	tests := []struct {
		name        string
		database    models.DatabaseConfig
		databaseENV string
		wantErr     string
	}{
		{
			name:     "sqlite in-memory",
			database: models.DatabaseConfig{Provider: "sqlite", URL: ":memory:"},
		},
		{
			name:        "environment url is preferred over config",
			database:    models.DatabaseConfig{Provider: "sqlite", URL: "ignored.db"},
			databaseENV: ":memory:",
		},
		{
			name:     "missing provider",
			database: models.DatabaseConfig{URL: ":memory:"},
			wantErr:  "database provider must be specified",
		},
		{
			name:     "missing url",
			database: models.DatabaseConfig{Provider: "sqlite"},
			wantErr:  "database connection string must be specified",
		},
		{
			name:     "unsupported provider",
			database: models.DatabaseConfig{Provider: "oracle", URL: "oracle://localhost"},
			wantErr:  "unsupported database provider: oracle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(env.EnvDatabaseURL, tt.databaseENV)

			cfg := &models.Config{Database: tt.database}
			db, err := InitDatabase(cfg, new(internaltests.LevelRecordingLogger), "info")

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, db)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, db)

			closer, ok := db.(closeableDB)
			require.True(t, ok, "initialized database must be closeable")
			t.Cleanup(func() { require.NoError(t, closer.Close()) })
		})
	}
}

func TestInitEventBus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		bus     models.EventBusConfig
		wantErr string
	}{
		{
			name: "explicit gochannel config",
			bus: models.EventBusConfig{
				Provider:  events.ProviderGoChannel,
				GoChannel: &models.GoChannelConfig{BufferSize: 10},
			},
		},
		{
			name: "gochannel without config uses default buffer",
			bus:  models.EventBusConfig{Provider: events.ProviderGoChannel},
		},
		{
			name:    "unsupported provider",
			bus:     models.EventBusConfig{Provider: "carrier-pigeon"},
			wantErr: "unsupported event bus provider: carrier-pigeon",
		},
		{
			// InitEventBus only defaults the local provider variable, so an empty
			// provider is still rejected by the watermill provider factory.
			name:    "empty provider is rejected",
			bus:     models.EventBusConfig{},
			wantErr: "unsupported event bus provider:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &models.Config{EventBus: tt.bus}
			bus, err := InitEventBus(cfg)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, bus)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, bus)
			t.Cleanup(func() { require.NoError(t, bus.Close()) })
		})
	}
}

func TestInitCoreServices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  func() *models.Config
		wantErr string
	}{
		{
			name:   "registers every core service",
			config: newTestConfig,
		},
		{
			name: "invalid encryption key",
			config: func() *models.Config {
				return &models.Config{SecretKey: []byte("short")}
			},
			wantErr: "failed to initialize token repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.config()
			registry := internalplugins.NewServiceRegistry()

			var db *bun.DB
			if tt.wantErr == "" {
				db = internaltests.NewSQLiteIntegrationDB(t)
			}

			services, err := InitCoreServices(cfg, db, registry, new(internaltests.LevelRecordingLogger))

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, services)
				assert.Nil(t, registry.Get(models.ServiceUser.String()))
				assert.Nil(t, registry.Get(models.ServiceToken.String()))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, services)
			assert.NotNil(t, services.UserService)
			assert.NotNil(t, services.AccountService)
			assert.NotNil(t, services.SessionService)
			assert.NotNil(t, services.VerificationService)
			assert.NotNil(t, services.TokenService)
			assert.NotNil(t, services.PasswordService)

			registered := []struct {
				name  string
				value any
			}{
				{models.ServiceUser.String(), services.UserService},
				{models.ServiceAccount.String(), services.AccountService},
				{models.ServiceSession.String(), services.SessionService},
				{models.ServiceVerification.String(), services.VerificationService},
				{models.ServiceToken.String(), services.TokenService},
				{models.ServicePassword.String(), services.PasswordService},
			}
			for _, entry := range registered {
				assert.Same(t, entry.value, registry.Get(entry.name), "service %s is not registered", entry.name)
			}
		})
	}
}

func TestInitCoreSystems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		wantNames []string
	}{
		{
			name:      "returns session and verification cleanup systems in order",
			wantNames: []string{"SessionCleanupSystem", "VerificationCleanupSystem"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig()
			sessionService := new(internaltests.MockSessionService)
			verificationService := new(internaltests.MockVerificationService)
			coreServices := &serviceinterfaces.CoreServices{
				SessionService:      sessionService,
				VerificationService: verificationService,
			}

			systems := InitCoreSystems(new(internaltests.LevelRecordingLogger), cfg, coreServices)

			names := make([]string, 0, len(systems))
			for _, system := range systems {
				names = append(names, system.Name())
			}
			assert.Equal(t, tt.wantNames, names)

			for _, system := range systems {
				require.NoError(t, system.Init(context.Background()))
				require.NoError(t, system.Close())
			}

			sessionService.AssertExpectations(t)
			verificationService.AssertExpectations(t)
		})
	}
}
