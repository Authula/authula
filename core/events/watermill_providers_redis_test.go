package events

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/models"
)

// redisConsumerGroupNames returns the names of the consumer groups on the stream,
// read straight from the server so the test does not depend on subscriber internals.
func redisConsumerGroupNames(t *testing.T, addr, stream string) []string {
	t.Helper()

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	groups, err := client.XInfoGroups(t.Context(), stream).Result()
	require.NoError(t, err)

	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Name)
	}

	return names
}

// closeRedisPubSub closes a Redis-backed pubsub during cleanup. The publisher and subscriber
// share one client, so the second Close reports "redis: client is closed". Assert on the
// error again once the provider closes the shared client only once.
func closeRedisPubSub(t *testing.T, pubsub models.PubSub) {
	t.Helper()

	t.Cleanup(func() {
		_ = pubsub.Close()
	})
}

func TestInitWatermillProvider_Redis_ConnectsToServer(t *testing.T) {
	testCases := []struct {
		name      string
		envURL    func(addr string) string
		configURL func(addr string) string
	}{
		{
			name:      "config url connects when the env url is empty",
			configURL: func(addr string) string { return "redis://" + addr },
		},
		{
			name:      "env url connects over an invalid config url",
			envURL:    func(addr string) string { return "redis://" + addr },
			configURL: func(string) string { return "http://ignored" },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := miniredis.RunT(t)
			envURL := ""
			if tc.envURL != nil {
				envURL = tc.envURL(server.Addr())
			}
			t.Setenv(env.EnvRedisURL, envURL)

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "redis",
				Redis:    &models.RedisConfig{URL: tc.configURL(server.Addr())},
			}, watermill.NopLogger{})

			require.NoError(t, err)
			require.NotNil(t, pubsub)
			closeRedisPubSub(t, pubsub)
		})
	}
}

func TestInitWatermillProvider_Redis_ConsumerGroupSelection(t *testing.T) {
	const topic = "authula.events"

	testCases := []struct {
		name        string
		envGroup    string
		configGroup string
		wantGroup   string
	}{
		{
			name:        "env consumer group takes precedence over config",
			envGroup:    "env-group",
			configGroup: "config-group",
			wantGroup:   "env-group",
		},
		{
			name:        "config consumer group is used when env is empty",
			configGroup: "config-group",
			wantGroup:   "config-group",
		},
		{
			name:      "default consumer group is used when env and config are empty",
			wantGroup: "authula_consumer_group",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := miniredis.RunT(t)
			t.Setenv(env.EnvRedisURL, "")
			t.Setenv(env.EnvEventBusConsumerGroup, tc.envGroup)

			pubsub, err := InitWatermillProvider(&models.EventBusConfig{
				Provider: "redis",
				Redis: &models.RedisConfig{
					URL:           "redis://" + server.Addr(),
					ConsumerGroup: tc.configGroup,
				},
			}, watermill.NopLogger{})
			require.NoError(t, err)
			closeRedisPubSub(t, pubsub)

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			// Subscribing creates the consumer group on the stream.
			_, err = pubsub.Subscribe(ctx, topic)
			require.NoError(t, err)

			assert.Equal(t, []string{tc.wantGroup}, redisConsumerGroupNames(t, server.Addr(), topic))
		})
	}
}
