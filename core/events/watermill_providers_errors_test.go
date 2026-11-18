package events

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Authula/authula/events"
	"github.com/Authula/authula/models"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nilLoggerConfig returns a gochannel config so the nil-logger path can be exercised
// without touching any broker.
func nilLoggerConfig() *models.EventBusConfig {
	return &models.EventBusConfig{
		Provider:  "gochannel",
		GoChannel: &models.GoChannelConfig{BufferSize: 10},
	}
}

func TestInitWatermillProvider_NilLogger(t *testing.T) {
	pubsub, err := InitWatermillProvider(nilLoggerConfig(), nil)
	require.NoError(t, err)
	require.NotNil(t, pubsub)

	require.NoError(t, pubsub.Close())
}

func TestInitWatermillProvider_UnknownProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{name: "empty provider", provider: ""},
		{name: "unknown provider", provider: "nats-streaming"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := InitWatermillProvider(&models.EventBusConfig{Provider: events.EventBusProvider(tt.provider)}, nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported event bus provider")
		})
	}
}

func TestInitWatermillProvider_SQLite_TempDatabase(t *testing.T) {
	tests := []struct {
		name   string
		dbPath func(t *testing.T) string
	}{
		{
			name: "database file in temp dir",
			dbPath: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "events.db")
			},
		},
		{
			name: "nested directories are created",
			dbPath: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "nested", "deeper", "events.db")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.dbPath(t)

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "sqlite",
				SQLite:   &models.SQLiteConfig{DBPath: dbPath},
			}, watermill.NewStdLogger(false, false))

			require.NoError(t, err)
			require.NotNil(t, pubsub)
			require.NoError(t, pubsub.Close())

			_, statErr := os.Stat(dbPath)
			assert.NoError(t, statErr, "sqlite database file should exist at the configured path")
		})
	}
}

func TestInitWatermillProvider_SQLite_InvalidPath(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  func(t *testing.T) string
		wantErr string
	}{
		{
			name: "existing directory used as database file",
			dbPath: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantErr: "failed to create sqlite connection pool",
		},
		{
			name: "null byte in directory name",
			dbPath: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "a\x00b", "events.db")
			},
			wantErr: "failed to create directory for sqlite database",
		},
		{
			name: "regular file blocks directory creation",
			dbPath: func(t *testing.T) string {
				t.Helper()
				blocker := filepath.Join(t.TempDir(), "blocker")
				require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))
				return filepath.Join(blocker, "nested", "events.db")
			},
			wantErr: "failed to create directory for sqlite database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.dbPath(t)

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "sqlite",
				SQLite:   &models.SQLiteConfig{DBPath: dbPath},
			}, watermill.NewStdLogger(false, false))

			require.Error(t, err)
			assert.Nil(t, pubsub)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestInitWatermillProvider_Postgres_InvalidDSN(t *testing.T) {
	tests := []struct {
		name   string
		envURL *string
		config *models.PostgreSQLConfig
	}{
		{
			name:   "malformed dsn in config",
			config: &models.PostgreSQLConfig{URL: "not a valid dsn"},
		},
		{
			name:   "malformed dsn in env overrides config",
			envURL: new("also not a valid dsn"),
			config: &models.PostgreSQLConfig{URL: "postgres://ignored"},
		},
		{
			name:   "nil config without env url",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("POSTGRES_URL", "")
			if tt.envURL != nil {
				t.Setenv("POSTGRES_URL", *tt.envURL)
			}

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider:   "postgres",
				PostgreSQL: tt.config,
			}, watermill.NewStdLogger(false, false))

			require.Error(t, err)
			assert.Nil(t, pubsub)
			if tt.envURL == nil && tt.config == nil {
				assert.Contains(t, err.Error(), "postgres config with url is required")
				return
			}
			assert.Contains(t, err.Error(), "failed to ping postgres")
		})
	}
}

func TestInitWatermillProvider_Redis_InvalidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "missing protocol scheme", url: "://bad"},
		{name: "unsupported scheme", url: "http://localhost:6379"},
		{name: "missing scheme separator", url: "not a url"},
		{name: "invalid url escape", url: "redis://user:%zz@localhost:6379"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("REDIS_URL", "")

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "redis",
				Redis:    &models.RedisConfig{URL: tt.url},
			}, watermill.NewStdLogger(false, false))

			require.Error(t, err)
			assert.Nil(t, pubsub)
			assert.Contains(t, err.Error(), "invalid redis url")
		})
	}
}

func TestInitWatermillProvider_Redis_InvalidURLFromEnv(t *testing.T) {
	t.Setenv("REDIS_URL", "http://localhost:6379")

	pubsub, err := InitWatermillProvider(&models.EventBusConfig{Provider: "redis"}, watermill.NewStdLogger(false, false))

	require.Error(t, err)
	assert.Nil(t, pubsub)
	assert.Contains(t, err.Error(), "invalid redis url")
}

func TestInitWatermillProvider_Kafka_InvalidBrokers(t *testing.T) {
	tests := []struct {
		name       string
		envBrokers *string
		config     *models.KafkaConfig
		wantErr    string
	}{
		{
			name:    "whitespace only brokers from config",
			config:  &models.KafkaConfig{Brokers: " , , "},
			wantErr: "kafka config with brokers is required",
		},
		{
			name:       "whitespace only brokers from env",
			envBrokers: new(" , "),
			wantErr:    "kafka config with brokers is required",
		},
		{
			name:    "malformed broker address fails publisher creation",
			config:  &models.KafkaConfig{Brokers: "no-port", ConsumerGroup: "custom-group"},
			wantErr: "failed to create kafka publisher",
		},
		{
			name:       "malformed broker address from env fails publisher creation",
			envBrokers: new("no-port"),
			wantErr:    "failed to create kafka publisher",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KAFKA_BROKERS", "")
			t.Setenv("EVENT_BUS_CONSUMER_GROUP", "")
			if tt.envBrokers != nil {
				t.Setenv("KAFKA_BROKERS", *tt.envBrokers)
			}

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "kafka",
				Kafka:    tt.config,
			}, watermill.NewStdLogger(false, false))

			require.Error(t, err)
			assert.Nil(t, pubsub)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestInitWatermillProvider_NATS_InvalidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "missing protocol scheme", url: "://bad"},
		{name: "invalid url escape", url: "nats://user:%zz@localhost:4222"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NATS_URL", "")

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "nats",
				NATS:     &models.NatsConfig{URL: tt.url},
			}, watermill.NewStdLogger(false, false))

			require.Error(t, err)
			assert.Nil(t, pubsub)
			assert.Contains(t, err.Error(), "failed to create nats subscriber")
		})
	}
}

func TestInitWatermillProvider_NATS_InvalidURLFromEnv(t *testing.T) {
	t.Setenv("NATS_URL", "://bad")

	pubsub, err := InitWatermillProvider(&models.EventBusConfig{Provider: "nats"}, watermill.NewStdLogger(false, false))

	require.Error(t, err)
	assert.Nil(t, pubsub)
	assert.Contains(t, err.Error(), "failed to create nats subscriber")
}

func TestInitWatermillProvider_RabbitMQ_InvalidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "missing protocol scheme", url: "://bad"},
		{name: "unsupported scheme", url: "http://localhost"},
		{name: "invalid url escape", url: "amqp://user:%zz@localhost/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RABBITMQ_URL", "")

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "rabbitmq",
				RabbitMQ: &models.RabbitMQConfig{URL: tt.url},
			}, watermill.NewStdLogger(false, false))

			require.Error(t, err)
			assert.Nil(t, pubsub)
			assert.Contains(t, err.Error(), "failed to create rabbitmq subscriber")
		})
	}
}

func TestInitWatermillProvider_RabbitMQ_InvalidURLFromEnv(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "http://localhost")

	pubsub, err := InitWatermillProvider(&models.EventBusConfig{Provider: "rabbitmq"}, watermill.NewStdLogger(false, false))

	require.Error(t, err)
	assert.Nil(t, pubsub)
	assert.Contains(t, err.Error(), "failed to create rabbitmq subscriber")
}
