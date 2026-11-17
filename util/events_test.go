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
	publishErr := errors.New("broker unavailable")

	tests := []struct {
		name       string
		withBus    bool
		withLogger bool
		publishErr error
		wantLog    bool
	}{
		{
			name:       "nil event bus does nothing",
			withLogger: true,
		},
		{
			name:       "publishes event asynchronously",
			withBus:    true,
			withLogger: true,
		},
		{
			name:       "logs publish failure",
			withBus:    true,
			withLogger: true,
			publishErr: publishErr,
			wantLog:    true,
		},
		{
			name:       "publish failure without logger does not panic",
			withBus:    true,
			publishErr: publishErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := &recordingLogger{}
			published := make(chan models.Event, 1)

			var bus models.EventBus
			if tt.withBus {
				bus = &eventsTestBus{published: published, err: tt.publishErr}
			}

			var logger models.Logger
			if tt.withLogger {
				logger = recorder
			}

			PublishEventAsync(bus, logger, event)

			if tt.withBus {
				select {
				case got := <-published:
					assert.Equal(t, event, got)
				case <-time.After(2 * time.Second):
					t.Fatal("timed out waiting for event to be published")
				}
			}

			if !tt.wantLog {
				assert.Empty(t, recorder.Records())
				return
			}

			require.Eventually(t, func() bool {
				return len(recorder.Records()) == 1
			}, 2*time.Second, 5*time.Millisecond)

			record := recorder.Records()[0]
			assert.Equal(t, "error", record.Level)
			assert.Equal(t, "failed to publish event asynchronously", record.Message)
			assert.Contains(t, record.Args, "event_type")
			assert.Contains(t, record.Args, event.Type)
			assert.Contains(t, record.Args, event.ID)
			assert.Contains(t, record.Args, publishErr)
		})
	}
}
