package events

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventBusProviderString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider EventBusProvider
		want     string
	}{
		{name: "gochannel", provider: ProviderGoChannel, want: "gochannel"},
		{name: "sqlite", provider: ProviderSQLite, want: "sqlite"},
		{name: "postgres", provider: ProviderPostgres, want: "postgres"},
		{name: "redis", provider: ProviderRedis, want: "redis"},
		{name: "kafka", provider: ProviderKafka, want: "kafka"},
		{name: "nats", provider: ProviderNATS, want: "nats"},
		{name: "rabbitmq", provider: ProviderRabbitMQ, want: "rabbitmq"},
		{name: "empty", provider: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.provider.String())
		})
	}
}

func TestEventBusProviderValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider EventBusProvider
		want     bool
	}{
		{name: "gochannel is valid", provider: ProviderGoChannel, want: true},
		{name: "sqlite is valid", provider: ProviderSQLite, want: true},
		{name: "postgres is valid", provider: ProviderPostgres, want: true},
		{name: "redis is valid", provider: ProviderRedis, want: true},
		{name: "kafka is valid", provider: ProviderKafka, want: true},
		{name: "nats is valid", provider: ProviderNATS, want: true},
		{name: "rabbitmq is valid", provider: ProviderRabbitMQ, want: true},
		{name: "empty is invalid", provider: "", want: false},
		{name: "unknown is invalid", provider: "memory", want: false},
		{name: "provider names are case-sensitive", provider: "GoChannel", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.provider.Valid())
		})
	}
}
