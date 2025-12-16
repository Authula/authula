package events

import (
	"context"
	"sync"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/stretchr/testify/assert"
)

func TestSimplePubSub_Publish(t *testing.T) {
	pubsub := NewInMemoryPubSub()
	defer pubsub.Close()

	msg := &models.Message{
		UUID:    "test-123",
		Payload: []byte("test payload"),
		Metadata: map[string]string{
			"key": "value",
		},
	}

	err := pubsub.Publish(context.Background(), "test.topic", msg)
	assert.NoError(t, err)
}

func TestSimplePubSub_Subscribe(t *testing.T) {
	pubsub := NewInMemoryPubSub()
	defer pubsub.Close()

	// Subscribe first
	ch, err := pubsub.Subscribe(context.Background(), "test.topic")
	assert.NoError(t, err)

	// Publish message
	msg := &models.Message{
		UUID:    "test-456",
		Payload: []byte("hello world"),
		Metadata: map[string]string{
			"source": "test",
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var received *models.Message
	go func() {
		received = <-ch
		wg.Done()
	}()

	err = pubsub.Publish(context.Background(), "test.topic", msg)
	assert.NoError(t, err)

	wg.Wait()
	assert.NotNil(t, received)
	assert.Equal(t, "test-456", received.UUID)
	assert.Equal(t, []byte("hello world"), received.Payload)
	assert.Equal(t, "test", received.Metadata["source"])
}

func TestSimplePubSub_MultipleSubscribers(t *testing.T) {
	pubsub := NewInMemoryPubSub()
	defer pubsub.Close()

	// Create two subscribers
	ch1, err := pubsub.Subscribe(context.Background(), "test.topic")
	assert.NoError(t, err)

	ch2, err := pubsub.Subscribe(context.Background(), "test.topic")
	assert.NoError(t, err)

	msg := &models.Message{
		UUID:    "broadcast-789",
		Payload: []byte("broadcast message"),
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var received1, received2 *models.Message

	go func() {
		received1 = <-ch1
		wg.Done()
	}()

	go func() {
		received2 = <-ch2
		wg.Done()
	}()

	// Publish should reach both subscribers
	err = pubsub.Publish(context.Background(), "test.topic", msg)
	assert.NoError(t, err)

	wg.Wait()
	assert.NotNil(t, received1)
	assert.NotNil(t, received2)
	assert.Equal(t, "broadcast-789", received1.UUID)
	assert.Equal(t, "broadcast-789", received2.UUID)
}
