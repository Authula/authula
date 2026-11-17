package util

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

type eventsTestBus struct {
	published chan models.Event
	err       error
}

func (b *eventsTestBus) Publish(event models.Event) error {
	b.published <- event
	return b.err
}

func (b *eventsTestBus) Close() error { return nil }

func (b *eventsTestBus) Subscribe(string, models.EventHandler) (models.SubscriptionID, error) {
	return 0, nil
}

func (b *eventsTestBus) Unsubscribe(string, models.SubscriptionID) {}

func TestPublishEventAsync(t *testing.T) {
	t.Parallel()

	event := models.Event{ID: "evt-1", Type: "user.created", Payload: map[string]any{"id": "user-1"}}

	t.Run("nil event bus does nothing", func(t *testing.T) {
		t.Parallel()

		logger := &recordingLogger{}

		PublishEventAsync(nil, logger, event)

		assert.Empty(t, logger.Records())
	})

	t.Run("publishes event asynchronously", func(t *testing.T) {
		t.Parallel()

		bus := &eventsTestBus{published: make(chan models.Event, 1)}

		PublishEventAsync(bus, nil, event)

		select {
		case got := <-bus.published:
			assert.Equal(t, event, got)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for event to be published")
		}
	})

	t.Run("logs publish failure", func(t *testing.T) {
		t.Parallel()

		publishErr := errors.New("broker unavailable")
		bus := &eventsTestBus{published: make(chan models.Event, 1), err: publishErr}
		logger := &recordingLogger{}

		PublishEventAsync(bus, logger, event)

		require.Eventually(t, func() bool {
			return len(logger.Records()) == 1
		}, 2*time.Second, 5*time.Millisecond)

		record := logger.Records()[0]
		assert.Equal(t, "error", record.Level)
		assert.Equal(t, "failed to publish event asynchronously", record.Message)
		assert.Contains(t, record.Args, "event_type")
		assert.Contains(t, record.Args, event.Type)
		assert.Contains(t, record.Args, event.ID)
		assert.Contains(t, record.Args, publishErr)
	})

	t.Run("publish failure without logger does not panic", func(t *testing.T) {
		t.Parallel()

		bus := &eventsTestBus{published: make(chan models.Event, 1), err: errors.New("broker unavailable")}

		PublishEventAsync(bus, nil, event)

		select {
		case got := <-bus.published:
			assert.Equal(t, event, got)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for event to be published")
		}
	})
}
