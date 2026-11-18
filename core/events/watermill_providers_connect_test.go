package events

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/models"
)

// unreachableBrokerAddr has no listener, so dials are refused immediately and no broker is needed.
const unreachableBrokerAddr = "127.0.0.1:1"

func TestInitWatermillProvider_Postgres_UnreachableURL(t *testing.T) {
	testCases := []struct {
		name      string
		envURL    string
		configURL string
	}{
		{
			name:      "env url is used over the config url",
			envURL:    "postgres://user:pass@" + unreachableBrokerAddr + "/db?sslmode=disable",
			configURL: "postgres://ignored/db?sslmode=disable",
		},
		{
			name:      "config url is used when the env url is empty",
			configURL: "postgres://user:pass@" + unreachableBrokerAddr + "/db?sslmode=disable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POSTGRES_URL", tc.envURL)

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider:   "postgres",
				PostgreSQL: &models.PostgreSQLConfig{URL: tc.configURL},
			}, watermill.NopLogger{})

			require.ErrorContains(t, err, "failed to ping postgres")
			require.ErrorContains(t, err, unreachableBrokerAddr)
			assert.Nil(t, pubsub)
		})
	}
}

func TestInitWatermillProvider_Redis_URLSelection(t *testing.T) {
	testCases := []struct {
		name      string
		envURL    string
		configURL string
		wantErr   string
	}{
		{
			name:      "env url is used over an invalid config url",
			envURL:    "redis://" + unreachableBrokerAddr,
			configURL: "http://ignored",
			wantErr:   "failed to connect to redis",
		},
		{
			name:      "config url is used when the env url is empty",
			configURL: "redis://" + unreachableBrokerAddr,
			wantErr:   "failed to connect to redis",
		},
		{
			name:    "missing env and config url is rejected",
			wantErr: "redis config with url is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.EnvRedisURL, tc.envURL)

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "redis",
				Redis:    &models.RedisConfig{URL: tc.configURL},
			}, watermill.NopLogger{})

			require.ErrorContains(t, err, tc.wantErr)
			assert.Nil(t, pubsub)
		})
	}
}

func TestInitWatermillProvider_RabbitMQ_UnreachableBroker(t *testing.T) {
	testCases := []struct {
		name      string
		envURL    string
		configURL string
	}{
		{
			name:      "config url dial failure fails subscriber creation",
			configURL: "amqp://guest:guest@" + unreachableBrokerAddr + "/",
		},
		{
			name:   "env url dial failure fails subscriber creation",
			envURL: "amqp://guest:guest@" + unreachableBrokerAddr + "/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.EnvRabbitMQURL, tc.envURL)

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "rabbitmq",
				RabbitMQ: &models.RabbitMQConfig{URL: tc.configURL},
			}, watermill.NopLogger{})

			require.ErrorContains(t, err, "failed to create rabbitmq subscriber")
			require.ErrorContains(t, err, unreachableBrokerAddr)
			assert.Nil(t, pubsub)
		})
	}
}

func TestInitWatermillProvider_NATS_UnreachableBroker(t *testing.T) {
	testCases := []struct {
		name      string
		envURL    string
		configURL string
	}{
		{
			name:      "config url connection failure fails subscriber creation",
			configURL: "nats://" + unreachableBrokerAddr,
		},
		{
			name:   "env url connection failure fails subscriber creation",
			envURL: "nats://" + unreachableBrokerAddr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.EnvNatsURL, tc.envURL)

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "nats",
				NATS:     &models.NatsConfig{URL: tc.configURL},
			}, watermill.NopLogger{})

			require.ErrorContains(t, err, "failed to create nats subscriber")
			assert.Nil(t, pubsub)
		})
	}
}
