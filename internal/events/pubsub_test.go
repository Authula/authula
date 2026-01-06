package events

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/stretchr/testify/assert"

	"github.com/GoBetterAuth/go-better-auth/models"
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
	ps := newTestGoChannelPubSub(nil, 0)
	assert.NotNil(t, ps)
	defer ps.Close()

	// Test basic publish/subscribe
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

	// Receive message
	select {
	case received := <-ch:
		assert.Equal(t, "test-123", received.UUID)
		assert.Equal(t, []byte("test message"), received.Payload)
		assert.Equal(t, "value", received.Metadata["key"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestNewGoChannelPubSub_WithOptions(t *testing.T) {
	// Test with custom buffer size and logger
	logger := watermill.NewStdLogger(false, false)
	ps := newTestGoChannelPubSub(logger, 500)
	assert.NotNil(t, ps)
	defer ps.Close()

	// Test that it still works
	ctx := context.Background()
	ch, err := ps.Subscribe(ctx, "test.topic")
	assert.NoError(t, err)

	msg := &models.Message{
		UUID:    "test-456",
		Payload: []byte("custom config test"),
	}

	err = ps.Publish(ctx, "test.topic", msg)
	assert.NoError(t, err)

	// Receive message
	select {
	case received := <-ch:
		assert.Equal(t, "test-456", received.UUID)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestNewGoChannelPubSub_MultipleSubscribers(t *testing.T) {
	ps := newTestGoChannelPubSub(nil, 100)
	defer ps.Close()

	ctx := context.Background()

	// Create two subscribers
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

	// Both subscribers should receive the message
	received := 0
	timeout := time.After(1 * time.Second)

	for received < 2 {
		select {
		case msg1 := <-ch1:
			assert.Equal(t, "broadcast-789", msg1.UUID)
			received++
		case msg2 := <-ch2:
			assert.Equal(t, "broadcast-789", msg2.UUID)
			received++
		case <-timeout:
			t.Fatalf("timeout: only received %d/2 messages", received)
		}
	}
}
