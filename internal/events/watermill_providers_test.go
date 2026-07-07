package events

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

func TestInitWatermillProvider_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *models.EventBusConfig
	}{
		{
			name: "gochannel with buffer",
			config: &models.EventBusConfig{
				Provider:  "gochannel",
				GoChannel: &models.GoChannelConfig{BufferSize: 100},
			},
		},
		{
			name: "gochannel default buffer",
			config: &models.EventBusConfig{
				Provider:  "gochannel",
				GoChannel: &models.GoChannelConfig{},
			},
		},
		{
			name: "gochannel nil config",
			config: &models.EventBusConfig{
				Provider:  "gochannel",
				GoChannel: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := watermill.NewStdLogger(false, false)
			pubsub, err := InitWatermillProvider(tt.config, logger)

			require.NoError(t, err)
			require.NotNil(t, pubsub)
			require.NoError(t, pubsub.Close())
		})
	}
}

func TestInitWatermillProvider_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *models.EventBusConfig
	}{
		{
			name: "unsupported provider",
			config: &models.EventBusConfig{
				Provider: "unsupported",
			},
		},
		{
			name: "redis missing URL",
			config: &models.EventBusConfig{
				Provider: "redis",
				Redis:    &models.RedisConfig{},
			},
		},
		{
			name: "redis nil config",
			config: &models.EventBusConfig{
				Provider: "redis",
				Redis:    nil,
			},
		},
		{
			name: "kafka missing brokers",
			config: &models.EventBusConfig{
				Provider: "kafka",
				Kafka:    &models.KafkaConfig{},
			},
		},
		{
			name: "kafka nil config",
			config: &models.EventBusConfig{
				Provider: "kafka",
				Kafka:    nil,
			},
		},
		{
			name: "nats missing URL",
			config: &models.EventBusConfig{
				Provider: "nats",
				NATS:     &models.NatsConfig{},
			},
		},
		{
			name: "nats nil config",
			config: &models.EventBusConfig{
				Provider: "nats",
				NATS:     nil,
			},
		},
		{
			name: "postgres missing URL",
			config: &models.EventBusConfig{
				Provider:   "postgres",
				PostgreSQL: &models.PostgreSQLConfig{},
			},
		},
		{
			name: "postgres nil config",
			config: &models.EventBusConfig{
				Provider:   "postgres",
				PostgreSQL: nil,
			},
		},
		{
			name: "rabbitmq missing URL",
			config: &models.EventBusConfig{
				Provider: "rabbitmq",
				RabbitMQ: &models.RabbitMQConfig{},
			},
		},
		{
			name: "rabbitmq nil config",
			config: &models.EventBusConfig{
				Provider: "rabbitmq",
				RabbitMQ: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := watermill.NewStdLogger(false, false)
			_, err := InitWatermillProvider(tt.config, logger)

			assert.Error(t, err)
		})
	}
}
