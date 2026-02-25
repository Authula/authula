package events

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/stretchr/testify/assert"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// newTestGoChannelPubSub creates a Watermill GoChannel PubSub for testing.
// This is a test helper and not part of the public API.
func newTestGoChannelPubSub(logger watermill.LoggerAdapter, bufferSize int) models.PubSub {
	if logger == nil {
		logger = watermill.NopLogger{}
	}

	if bufferSize <= 0 {
		bufferSize = 100
	}

	goChannel := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: int64(bufferSize),
			Persistent:          false,
		},
		logger,
	)

	return NewWatermillPubSub(goChannel, goChannel)
}

func TestNewGoChannelPubSub_Default(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ps := newTestGoChannelPubSub(nil, 0)
		assert.NotNil(t, ps)
		defer func() {
			if err := ps.Close(); err != nil {
				t.Errorf("failed to close PubSub: %v", err)
			}
		}()

		ctx := context.Background()
		ch, err := ps.Subscribe(ctx, "test.topic")
		assert.NoError(t, err)

		msg := &models.Message{
			UUID:    "test-123",
			Payload: []byte("test message"),
			Metadata: map[string]string{
				"key": "value",
			},
		}

		err = ps.Publish(ctx, "test.topic", msg)
		assert.NoError(t, err)
		synctest.Wait()

		select {
		case received := <-ch:
			assert.Equal(t, "test-123", received.UUID)
			assert.Equal(t, []byte("test message"), received.Payload)
			assert.Equal(t, "value", received.Metadata["key"])
		default:
			t.Fatal("expected message to be delivered")
		}
	})
}

func TestNewGoChannelPubSub_WithOptions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logger := watermill.NewStdLogger(false, false)
		ps := newTestGoChannelPubSub(logger, 500)
		assert.NotNil(t, ps)
		defer func() {
			if err := ps.Close(); err != nil {
				t.Errorf("failed to close PubSub: %v", err)
			}
		}()

		ctx := context.Background()
		ch, err := ps.Subscribe(ctx, "test.topic")
		assert.NoError(t, err)

		msg := &models.Message{
			UUID:    "test-456",
			Payload: []byte("custom config test"),
		}

		err = ps.Publish(ctx, "test.topic", msg)
		assert.NoError(t, err)
		synctest.Wait()

		select {
		case received := <-ch:
			assert.Equal(t, "test-456", received.UUID)
		default:
			t.Fatal("expected message to be delivered")
		}
	})
}

func TestNewGoChannelPubSub_MultipleSubscribers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ps := newTestGoChannelPubSub(nil, 100)
		defer func() {
			if err := ps.Close(); err != nil {
				t.Errorf("failed to close PubSub: %v", err)
			}
		}()

		ctx := context.Background()
		ch1, err := ps.Subscribe(ctx, "broadcast.topic")
		assert.NoError(t, err)

		ch2, err := ps.Subscribe(ctx, "broadcast.topic")
		assert.NoError(t, err)

		msg := &models.Message{
			UUID:    "broadcast-789",
			Payload: []byte("broadcast message"),
		}

		err = ps.Publish(ctx, "broadcast.topic", msg)
		assert.NoError(t, err)
		synctest.Wait()

		select {
		case msg1 := <-ch1:
			assert.Equal(t, "broadcast-789", msg1.UUID)
		default:
			t.Fatal("expected first subscriber to receive message")
		}

		select {
		case msg2 := <-ch2:
			assert.Equal(t, "broadcast-789", msg2.UUID)
		default:
			t.Fatal("expected second subscriber to receive message")
		}
	})
}
