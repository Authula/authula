package config

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/events"
	"github.com/Authula/authula/models"
)

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	config := NewDefaultConfig()

	require.Equal(t, "Authula", config.AppName)
	require.Equal(t, "http://localhost:8080", config.BaseURL)
	require.Equal(t, "/auth", config.BasePath)
	require.Equal(t, defaultSecret, config.Secret)
	require.Empty(t, config.Secrets)
	require.Empty(t, config.DisabledPaths)

	require.Equal(t, "authula.session_token", config.Session.CookieName)
	require.Empty(t, config.Session.Domain)
	require.Equal(t, 7*24*time.Hour, config.Session.ExpiresIn)
	require.Equal(t, 24*time.Hour, config.Session.UpdateAge)
	require.Equal(t, 7*24*time.Hour, config.Session.CookieMaxAge)
	require.False(t, config.Session.Secure)
	require.True(t, config.Session.HttpOnly)
	require.Equal(t, "lax", config.Session.SameSite)
	require.False(t, config.Session.AutoCleanup)
	require.Equal(t, time.Minute, config.Session.CleanupInterval)
	require.Equal(t, 5, config.Session.MaxSessionsPerUser)

	require.False(t, config.Verification.AutoCleanup)
	require.Equal(t, time.Minute, config.Verification.CleanupInterval)

	require.Empty(t, config.Security.TrustedOrigins)
	require.Empty(t, config.Security.TrustedHeaders)
	require.Empty(t, config.Security.TrustedProxies)
	require.True(t, config.Security.CORS.AllowCredentials)
	require.Equal(t, []string{"*"}, config.Security.CORS.AllowedOrigins)
	require.Equal(t, []string{"OPTIONS", "GET", "POST", "PUT", "PATCH", "DELETE"}, config.Security.CORS.AllowedMethods)
	require.Equal(t, []string{"Authorization", "Content-Type", "Cookie", "Set-Cookie"}, config.Security.CORS.AllowedHeaders)
	require.Empty(t, config.Security.CORS.ExposedHeaders)
	require.Equal(t, 24*time.Hour, config.Security.CORS.MaxAge)

	require.Equal(t, 25, config.Database.MaxOpenConns)
	require.Equal(t, 5, config.Database.MaxIdleConns)
	require.Equal(t, 10*time.Minute, config.Database.ConnMaxLifetime)

	require.Equal(t, 100, config.EventBus.MaxConcurrentHandlers)
	require.Equal(t, 5*time.Second, config.EventBus.ContextTimeout)
	require.Equal(t, events.ProviderGoChannel, config.EventBus.Provider)
	require.NotNil(t, config.EventBus.GoChannel)

	require.NotNil(t, config.PreParsedConfigs)
	require.Empty(t, config.PreParsedConfigs)
	require.Empty(t, config.RouteMappings)
	require.Nil(t, config.CoreServiceHooks)
}

func TestNewConfigOptions(t *testing.T) {
	t.Parallel()

	hooks := &models.CoreServiceHooksConfig{}

	tests := []struct {
		name    string
		options []ConfigOption
		check   func(t *testing.T, config *models.Config)
	}{
		{
			name:    "app name is applied",
			options: []ConfigOption{WithAppName("My App")},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "My App", config.AppName)
			},
		},
		{
			name:    "empty app name keeps the default",
			options: []ConfigOption{WithAppName("")},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "Authula", config.AppName)
			},
		},
		{
			name:    "base URL is applied",
			options: []ConfigOption{WithBaseURL("https://api.example.com")},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "https://api.example.com", config.BaseURL)
			},
		},
		{
			name:    "empty base URL keeps the default",
			options: []ConfigOption{WithBaseURL("")},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "http://localhost:8080", config.BaseURL)
			},
		},
		{
			name:    "base path is applied",
			options: []ConfigOption{WithBasePath("/api/auth")},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "/api/auth", config.BasePath)
			},
		},
		{
			name:    "empty base path keeps the default",
			options: []ConfigOption{WithBasePath("")},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "/auth", config.BasePath)
			},
		},
		{
			name: "database fields are applied",
			options: []ConfigOption{WithDatabase(models.DatabaseConfig{
				Provider:        "postgres",
				URL:             "postgres://localhost:5432/authula",
				MaxOpenConns:    10,
				MaxIdleConns:    2,
				ConnMaxLifetime: 3 * time.Minute,
			})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "postgres", config.Database.Provider)
				require.Equal(t, "postgres://localhost:5432/authula", config.Database.URL)
				require.Equal(t, 10, config.Database.MaxOpenConns)
				require.Equal(t, 2, config.Database.MaxIdleConns)
				require.Equal(t, 3*time.Minute, config.Database.ConnMaxLifetime)
			},
		},
		{
			name:    "empty database fields keep the defaults",
			options: []ConfigOption{WithDatabase(models.DatabaseConfig{})},
			check: func(t *testing.T, config *models.Config) {
				require.Empty(t, config.Database.Provider)
				require.Empty(t, config.Database.URL)
				require.Equal(t, 25, config.Database.MaxOpenConns)
				require.Equal(t, 5, config.Database.MaxIdleConns)
				require.Equal(t, 10*time.Minute, config.Database.ConnMaxLifetime)
			},
		},
		{
			name: "session fields are applied",
			options: []ConfigOption{WithSession(models.SessionConfig{
				CookieName:         "custom.session",
				ExpiresIn:          2 * time.Hour,
				UpdateAge:          30 * time.Minute,
				CookieMaxAge:       3 * time.Hour,
				Secure:             true,
				HttpOnly:           true,
				SameSite:           "strict",
				AutoCleanup:        true,
				CleanupInterval:    5 * time.Minute,
				MaxSessionsPerUser: 3,
			})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "custom.session", config.Session.CookieName)
				require.Equal(t, 2*time.Hour, config.Session.ExpiresIn)
				require.Equal(t, 30*time.Minute, config.Session.UpdateAge)
				require.Equal(t, 3*time.Hour, config.Session.CookieMaxAge)
				require.True(t, config.Session.Secure)
				require.True(t, config.Session.HttpOnly)
				require.Equal(t, "strict", config.Session.SameSite)
				require.True(t, config.Session.AutoCleanup)
				require.Equal(t, 5*time.Minute, config.Session.CleanupInterval)
				require.Equal(t, 3, config.Session.MaxSessionsPerUser)
			},
		},
		{
			name:    "empty session keeps the duration defaults",
			options: []ConfigOption{WithSession(models.SessionConfig{})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, "authula.session_token", config.Session.CookieName)
				require.Equal(t, 7*24*time.Hour, config.Session.ExpiresIn)
				require.Equal(t, 24*time.Hour, config.Session.UpdateAge)
				require.Equal(t, 7*24*time.Hour, config.Session.CookieMaxAge)
				require.Equal(t, "lax", config.Session.SameSite)
				require.Equal(t, time.Minute, config.Session.CleanupInterval)
			},
		},
		{
			name: "verification fields are applied",
			options: []ConfigOption{WithVerification(models.VerificationConfig{
				AutoCleanup:     true,
				CleanupInterval: 10 * time.Minute,
			})},
			check: func(t *testing.T, config *models.Config) {
				require.True(t, config.Verification.AutoCleanup)
				require.Equal(t, 10*time.Minute, config.Verification.CleanupInterval)
			},
		},
		{
			name:    "empty verification keeps the interval default",
			options: []ConfigOption{WithVerification(models.VerificationConfig{})},
			check: func(t *testing.T, config *models.Config) {
				require.False(t, config.Verification.AutoCleanup)
				require.Equal(t, time.Minute, config.Verification.CleanupInterval)
			},
		},
		{
			name: "security fields and CORS are applied",
			options: []ConfigOption{WithSecurity(models.SecurityConfig{
				TrustedOrigins: []string{"https://app.example.com"},
				TrustedHeaders: []string{"x-forwarded-for", "X-REAL-IP"},
				TrustedProxies: []string{"10.0.0.0/8"},
				CORS: models.CORSConfig{
					AllowCredentials: true,
					AllowedOrigins:   []string{"https://app.example.com"},
					AllowedMethods:   []string{"GET", "POST"},
					AllowedHeaders:   []string{"Authorization"},
					ExposedHeaders:   []string{"X-Total-Count"},
					MaxAge:           2 * time.Hour,
				},
			})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, []string{"https://app.example.com"}, config.Security.TrustedOrigins)
				require.Equal(t, []string{"X-Forwarded-For", "X-Real-Ip"}, config.Security.TrustedHeaders)
				require.Equal(t, []string{"10.0.0.0/8"}, config.Security.TrustedProxies)
				require.True(t, config.Security.CORS.AllowCredentials)
				require.Equal(t, []string{"https://app.example.com"}, config.Security.CORS.AllowedOrigins)
				require.Equal(t, []string{"GET", "POST"}, config.Security.CORS.AllowedMethods)
				require.Equal(t, []string{"Authorization"}, config.Security.CORS.AllowedHeaders)
				require.Equal(t, []string{"X-Total-Count"}, config.Security.CORS.ExposedHeaders)
				require.Equal(t, 2*time.Hour, config.Security.CORS.MaxAge)
			},
		},
		{
			name:    "empty security keeps most defaults but always sets allow credentials",
			options: []ConfigOption{WithSecurity(models.SecurityConfig{})},
			check: func(t *testing.T, config *models.Config) {
				require.Empty(t, config.Security.TrustedOrigins)
				require.Empty(t, config.Security.TrustedHeaders)
				require.Empty(t, config.Security.TrustedProxies)
				require.False(t, config.Security.CORS.AllowCredentials)
				require.Equal(t, []string{"*"}, config.Security.CORS.AllowedOrigins)
				require.Equal(t, 24*time.Hour, config.Security.CORS.MaxAge)
			},
		},
		{
			name: "event bus fields and provider configs are applied",
			options: []ConfigOption{WithEventBus(models.EventBusConfig{
				MaxConcurrentHandlers: 5,
				ContextTimeout:        2 * time.Second,
				Provider:              events.ProviderGoChannel,
				GoChannel:             &models.GoChannelConfig{BufferSize: 16},
				SQLite:                &models.SQLiteConfig{DBPath: "events.db"},
				PostgreSQL:            &models.PostgreSQLConfig{URL: "postgres://events"},
				Redis:                 &models.RedisConfig{URL: "redis://events", ConsumerGroup: "group"},
				Kafka:                 &models.KafkaConfig{Brokers: "localhost:9092", ConsumerGroup: "group"},
				NATS:                  &models.NatsConfig{URL: "nats://events"},
				RabbitMQ:              &models.RabbitMQConfig{URL: "amqp://events"},
			})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, 5, config.EventBus.MaxConcurrentHandlers)
				require.Equal(t, 2*time.Second, config.EventBus.ContextTimeout)
				require.Equal(t, events.ProviderGoChannel, config.EventBus.Provider)
				require.Equal(t, 16, config.EventBus.GoChannel.BufferSize)
				require.Equal(t, "events.db", config.EventBus.SQLite.DBPath)
				require.Equal(t, "postgres://events", config.EventBus.PostgreSQL.URL)
				require.Equal(t, "redis://events", config.EventBus.Redis.URL)
				require.Equal(t, "group", config.EventBus.Redis.ConsumerGroup)
				require.Equal(t, "localhost:9092", config.EventBus.Kafka.Brokers)
				require.Equal(t, "nats://events", config.EventBus.NATS.URL)
				require.Equal(t, "amqp://events", config.EventBus.RabbitMQ.URL)
			},
		},
		{
			name:    "empty event bus config keeps the defaults",
			options: []ConfigOption{WithEventBus(models.EventBusConfig{})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, 100, config.EventBus.MaxConcurrentHandlers)
				require.Equal(t, 5*time.Second, config.EventBus.ContextTimeout)
				require.Equal(t, events.ProviderGoChannel, config.EventBus.Provider)
				require.NotNil(t, config.EventBus.GoChannel)
			},
		},
		{
			name: "plugins are replaced",
			options: []ConfigOption{WithPlugins(models.PluginsConfig{
				models.PluginSession.String(): map[string]any{"enabled": true},
			})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, true, config.Plugins[models.PluginSession.String()].(map[string]any)["enabled"])
			},
		},
		{
			name: "route mappings are replaced",
			options: []ConfigOption{WithRouteMappings([]models.RouteMapping{
				{Paths: []string{"GET:/auth/me"}, Plugins: []string{"session.auth"}},
			})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, []models.RouteMapping{
					{Paths: []string{"GET:/auth/me"}, Plugins: []string{"session.auth"}},
				}, config.RouteMappings)
			},
		},
		{
			name:    "disabled paths are replaced",
			options: []ConfigOption{WithDisabledPaths([]string{"/health", "/metrics"})},
			check: func(t *testing.T, config *models.Config) {
				require.Equal(t, []string{"/health", "/metrics"}, config.DisabledPaths)
			},
		},
		{
			name:    "core service hooks are attached",
			options: []ConfigOption{WithCoreServiceHooks(hooks)},
			check: func(t *testing.T, config *models.Config) {
				require.Same(t, hooks, config.CoreServiceHooks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := NewConfig(tt.options...)

			tt.check(t, config)
		})
	}
}

func TestNewConfigBasePathValidation(t *testing.T) {
	t.Parallel()

	requirePanics(t, func() {
		_ = NewConfig(WithBasePath("auth"))
	})
}

func TestNewConfigRejectsAnEmptyBaseURL(t *testing.T) {
	t.Setenv(env.EnvBaseURL, "")

	requirePanics(t, func() {
		_ = NewConfig(func(config *models.Config) {
			config.BaseURL = ""
		})
	})
}

func TestWithBaseURLEnvOverride(t *testing.T) {
	t.Setenv(env.EnvBaseURL, "https://env.example.com")

	config := NewConfig(WithBaseURL("https://option.example.com"))

	require.Equal(t, "https://env.example.com", config.BaseURL)
}

func TestWithDatabaseEnvOverride(t *testing.T) {
	t.Setenv(env.EnvDatabaseURL, "postgres://env.example.com/authula")

	config := NewConfig(WithDatabase(models.DatabaseConfig{
		Provider:     "postgres",
		URL:          "postgres://option.example.com/authula",
		MaxOpenConns: 7,
	}))

	require.Equal(t, "postgres", config.Database.Provider)
	require.Equal(t, "postgres://env.example.com/authula", config.Database.URL)
	require.Equal(t, 7, config.Database.MaxOpenConns)
}

func TestValidateSecretsIsIdempotent(t *testing.T) {
	t.Setenv(env.EnvSecret, "")
	t.Setenv(env.EnvSecrets, "")

	config := NewConfig(
		WithSecret(customSecretOne),
		WithSecrets([]models.SecretConfig{
			{Version: 2, Value: customSecretTwo},
			{Version: 1, Value: customSecretOne},
		}),
	)
	secretKey := append([]byte(nil), config.SecretKey...)

	ValidateSecrets(config)

	require.Equal(t, customSecretOne, config.Secret)
	require.Equal(t, secretKey, config.SecretKey)
	require.Equal(t, []models.SecretConfig{
		{Version: 2, Value: customSecretTwo, Key: customSecretTwoRaw},
		{Version: 1, Value: customSecretOne, Key: customSecretOneRaw},
	}, config.Secrets)
}

func TestValidateSecretsRejectsInvalidVersionedSecrets(t *testing.T) {
	t.Setenv(env.EnvSecrets, "")

	tests := []struct {
		name    string
		secrets []models.SecretConfig
	}{
		{
			name:    "zero version",
			secrets: []models.SecretConfig{{Version: 0, Value: customSecretOne}},
		},
		{
			name:    "negative version",
			secrets: []models.SecretConfig{{Version: -1, Value: customSecretOne}},
		},
		{
			name: "duplicate version",
			secrets: []models.SecretConfig{
				{Version: 1, Value: customSecretOne},
				{Version: 1, Value: customSecretTwo},
			},
		},
		{
			name:    "empty value",
			secrets: []models.SecretConfig{{Version: 1, Value: "  "}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirePanics(t, func() {
				_ = NewConfig(WithSecrets(tt.secrets))
			})
		})
	}
}

func TestParseSecretsEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    []models.SecretConfig
		wantErr bool
	}{
		{
			name: "empty value yields no secrets",
			raw:  "",
			want: nil,
		},
		{
			name: "whitespace value yields no secrets",
			raw:  "   \t\n",
			want: nil,
		},
		{
			name: "valid entries are parsed and trimmed",
			raw:  " 2:" + customSecretTwo + " , 1:" + customSecretOne + " ",
			want: []models.SecretConfig{
				{Version: 2, Value: customSecretTwo},
				{Version: 1, Value: customSecretOne},
			},
		},
		{
			name:    "missing colon",
			raw:     "1",
			wantErr: true,
		},
		{
			name:    "empty entry",
			raw:     "1:" + customSecretOne + ",",
			wantErr: true,
		},
		{
			name:    "empty value",
			raw:     "1:",
			wantErr: true,
		},
		{
			name:    "non-numeric version",
			raw:     "abc:" + customSecretOne,
			wantErr: true,
		},
		{
			name:    "zero version",
			raw:     "0:" + customSecretOne,
			wantErr: true,
		},
		{
			name:    "negative version",
			raw:     "-2:" + customSecretOne,
			wantErr: true,
		},
		{
			name:    "duplicate version",
			raw:     "1:" + customSecretOne + ",1:" + customSecretTwo,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseSecretsEnv(tt.raw)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeCookieDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		domain  string
		baseURL string
		want    string
		wantErr bool
	}{
		{name: "empty domain stays host-only", domain: "", baseURL: "http://localhost:8080", want: ""},
		{name: "domain equal to the host is accepted", domain: "example.com", baseURL: "https://example.com", want: "example.com"},
		{name: "parent domain is accepted", domain: "example.com", baseURL: "https://api.example.com", want: "example.com"},
		{name: "leading dot and mixed case are normalized", domain: ".Example.COM", baseURL: "https://api.example.com:8443", want: "example.com"},
		{name: "domain with a port is rejected", domain: "example.com:8080", baseURL: "https://api.example.com", wantErr: true},
		{name: "domain with a scheme is rejected", domain: "https://example.com", baseURL: "https://api.example.com", wantErr: true},
		{name: "domain with a space is rejected", domain: "exa mple.com", baseURL: "https://api.example.com", wantErr: true},
		{name: "bare dot is rejected", domain: ".", baseURL: "https://api.example.com", wantErr: true},
		{name: "IP address is rejected", domain: "127.0.0.1", baseURL: "http://127.0.0.1:8080", wantErr: true},
		{name: "look-alike host is rejected", domain: "example.com", baseURL: "https://notexample.com", wantErr: true},
		{name: "unparsable base URL is rejected", domain: "example.com", baseURL: "://bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeCookieDomain(tt.domain, tt.baseURL)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestValidateEventBusConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  models.EventBusConfig
		envKey  string
		envVal  string
		wantErr bool
	}{
		{
			name:   "defaults to gochannel",
			config: models.EventBusConfig{GoChannel: &models.GoChannelConfig{}},
		},
		{
			name:    "empty provider without gochannel config",
			config:  models.EventBusConfig{},
			wantErr: true,
		},
		{
			name:    "gochannel without config",
			config:  models.EventBusConfig{Provider: events.ProviderGoChannel},
			wantErr: true,
		},
		{
			name:   "gochannel with config",
			config: models.EventBusConfig{Provider: events.ProviderGoChannel, GoChannel: &models.GoChannelConfig{}},
		},
		{
			name:    "sqlite without config",
			config:  models.EventBusConfig{Provider: events.ProviderSQLite},
			wantErr: true,
		},
		{
			name:   "sqlite with config",
			config: models.EventBusConfig{Provider: events.ProviderSQLite, SQLite: &models.SQLiteConfig{}},
		},
		{
			name:    "postgres without config",
			config:  models.EventBusConfig{Provider: events.ProviderPostgres},
			wantErr: true,
		},
		{
			name:    "postgres without url",
			config:  models.EventBusConfig{Provider: events.ProviderPostgres, PostgreSQL: &models.PostgreSQLConfig{}},
			wantErr: true,
		},
		{
			name:   "postgres with url",
			config: models.EventBusConfig{Provider: events.ProviderPostgres, PostgreSQL: &models.PostgreSQLConfig{URL: "postgres://events"}},
		},
		{
			name:   "postgres with url from the environment",
			config: models.EventBusConfig{Provider: events.ProviderPostgres, PostgreSQL: &models.PostgreSQLConfig{}},
			envKey: env.EnvPostgresURL,
			envVal: "postgres://env",
		},
		{
			name:    "redis without config",
			config:  models.EventBusConfig{Provider: events.ProviderRedis},
			wantErr: true,
		},
		{
			name:    "redis without url",
			config:  models.EventBusConfig{Provider: events.ProviderRedis, Redis: &models.RedisConfig{}},
			wantErr: true,
		},
		{
			name:   "redis with url from the environment",
			config: models.EventBusConfig{Provider: events.ProviderRedis, Redis: &models.RedisConfig{}},
			envKey: env.EnvRedisURL,
			envVal: "redis://env",
		},
		{
			name:    "kafka without config",
			config:  models.EventBusConfig{Provider: events.ProviderKafka},
			wantErr: true,
		},
		{
			name:    "kafka without brokers",
			config:  models.EventBusConfig{Provider: events.ProviderKafka, Kafka: &models.KafkaConfig{}},
			wantErr: true,
		},
		{
			name:   "kafka with brokers",
			config: models.EventBusConfig{Provider: events.ProviderKafka, Kafka: &models.KafkaConfig{Brokers: "localhost:9092"}},
		},
		{
			name:    "nats without config",
			config:  models.EventBusConfig{Provider: events.ProviderNATS},
			wantErr: true,
		},
		{
			name:    "nats without url",
			config:  models.EventBusConfig{Provider: events.ProviderNATS, NATS: &models.NatsConfig{}},
			wantErr: true,
		},
		{
			name:   "nats with url from the environment",
			config: models.EventBusConfig{Provider: events.ProviderNATS, NATS: &models.NatsConfig{}},
			envKey: env.EnvNatsURL,
			envVal: "nats://env",
		},
		{
			name:    "rabbitmq without config",
			config:  models.EventBusConfig{Provider: events.ProviderRabbitMQ},
			wantErr: true,
		},
		{
			name:    "rabbitmq without url",
			config:  models.EventBusConfig{Provider: events.ProviderRabbitMQ, RabbitMQ: &models.RabbitMQConfig{}},
			wantErr: true,
		},
		{
			name:   "rabbitmq with url from the environment",
			config: models.EventBusConfig{Provider: events.ProviderRabbitMQ, RabbitMQ: &models.RabbitMQConfig{}},
			envKey: env.EnvRabbitMQURL,
			envVal: "amqp://env",
		},
		{
			name:   "unknown provider is not validated",
			config: models.EventBusConfig{Provider: "bogus"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear every provider URL so ambient variables cannot change the result.
			t.Setenv(env.EnvPostgresURL, "")
			t.Setenv(env.EnvRedisURL, "")
			t.Setenv(env.EnvNatsURL, "")
			t.Setenv(env.EnvRabbitMQURL, "")
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			err := validateEventBusConfig(&tt.config)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewConfigEventBusProviderValidation(t *testing.T) {
	t.Setenv(env.EnvPostgresURL, "")
	t.Setenv(env.EnvRedisURL, "")
	t.Setenv(env.EnvNatsURL, "")
	t.Setenv(env.EnvRabbitMQURL, "")

	requirePanics(t, func() {
		_ = NewConfig(WithEventBus(models.EventBusConfig{Provider: events.ProviderSQLite}))
	})

	config := NewConfig(WithEventBus(models.EventBusConfig{
		Provider: events.ProviderSQLite,
		SQLite:   &models.SQLiteConfig{DBPath: "events.db"},
	}))

	require.Equal(t, events.ProviderSQLite, config.EventBus.Provider)
	require.Equal(t, "events.db", config.EventBus.SQLite.DBPath)
}

func TestSecurityTrustedHeadersRemainCanonical(t *testing.T) {
	t.Parallel()

	trustedHeaders := []string{"x-forwarded-for", "x-forwarded-proto"}
	headers := models.SecurityConfig{TrustedHeaders: trustedHeaders}

	config := NewConfig(WithSecurity(headers))

	require.Equal(t, []string{http.CanonicalHeaderKey("x-forwarded-for"), http.CanonicalHeaderKey("x-forwarded-proto")}, config.Security.TrustedHeaders)
}
