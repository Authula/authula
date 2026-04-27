package events

import (
	"context"
	"encoding/json"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

type recordingPubSub struct {
	topics []string
	msg    *models.Message
	ch     chan *models.Message
}

func newRecordingPubSub() *recordingPubSub {
	return &recordingPubSub{ch: make(chan *models.Message)}
}

func (p *recordingPubSub) Publish(ctx context.Context, topic string, msg *models.Message) error {
	p.topics = append(p.topics, topic)
	p.msg = msg
	return nil
}

func (p *recordingPubSub) Subscribe(ctx context.Context, topic string) (<-chan *models.Message, error) {
	return p.ch, nil
}

func (p *recordingPubSub) Close() error { return nil }

type wildcardTestEnv struct {
	bus      models.EventBus
	received chan models.Event
	pubsub   *recordingPubSub
	cleanup  func()
}

func setupWildcardDeliveryTestEnv(t *testing.T) wildcardTestEnv {
	t.Helper()

	config := &models.Config{}
	logger := watermill.NopLogger{}
	pubsub := gochannel.NewGoChannel(gochannel.Config{Persistent: false, OutputChannelBuffer: 10}, logger)
	bus := NewEventBus(config, logger, NewWatermillPubSub(pubsub, pubsub))

	return wildcardTestEnv{
		bus:      bus,
		received: make(chan models.Event, 1),
		cleanup: func() {
			require.NoError(t, bus.Close())
		},
	}
}

func setupWildcardTransportTestEnv(t *testing.T) wildcardTestEnv {
	t.Helper()

	config := &models.Config{}
	logger := watermill.NopLogger{}
	pubsub := newRecordingPubSub()
	bus := NewEventBus(config, logger, pubsub)

	return wildcardTestEnv{
		bus:      bus,
		received: make(chan models.Event, 1),
		pubsub:   pubsub,
		cleanup: func() {
			require.NoError(t, bus.Close())
		},
	}
}

func TestEventBusWildcardSubscription(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "receives all published events",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					env := setupWildcardDeliveryTestEnv(t)
					defer env.cleanup()
					t.Helper()

					_, err := env.bus.Subscribe(models.EventTypeWildcard, func(ctx context.Context, event models.Event) error {
						env.received <- event
						return nil
					})
					require.NoError(t, err)

					payload := map[string]string{"source": "test"}
					payloadBytes, err := json.Marshal(payload)
					require.NoError(t, err)

					now := time.Now().UTC()
					err = env.bus.Publish(models.Event{
						Type:      "some.event",
						Timestamp: now,
						Payload:   payloadBytes,
						Metadata: map[string]string{
							"key": "value",
						},
					})
					require.NoError(t, err)

					synctest.Wait()

					select {
					case event := <-env.received:
						assert.Equal(t, "some.event", event.Type)
						assert.Equal(t, now, event.Timestamp)
						assert.Equal(t, payloadBytes, []byte(event.Payload))
						assert.Equal(t, "value", event.Metadata["key"])
					default:
						t.Fatal("expected wildcard subscriber to receive event")
					}
				})
			},
		},
		{
			name: "does not use transport wildcard topic",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					env := setupWildcardTransportTestEnv(t)
					defer env.cleanup()
					t.Helper()

					_, err := env.bus.Subscribe(models.EventTypeWildcard, func(ctx context.Context, event models.Event) error {
						return nil
					})
					require.NoError(t, err)

					err = env.bus.Publish(models.Event{Type: "some.event"})
					require.NoError(t, err)

					assert.Equal(t, []string{"some.event"}, env.pubsub.topics)
				})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}
