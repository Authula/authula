package events

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/GoBetterAuth/go-better-auth/config"
	"github.com/GoBetterAuth/go-better-auth/internal/pubsub"
	"github.com/GoBetterAuth/go-better-auth/models"
)

func getMockConfig() *models.Config {
	return config.NewConfig()
}

func TestEventBus_Publish(t *testing.T) {
	bus := NewEventBus(getMockConfig(), nil)
	defer bus.Close()

	event := models.Event{
		Type:      models.EventUserSignedUp,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"user_id": "123",
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}
	err := bus.Publish(context.Background(), event)
	assert.NoError(t, err)
}

func TestWatermillEventBus_Publish(t *testing.T) {
	bus := NewEventBus(getMockConfig(), nil)
	defer bus.Close()

	event := models.Event{
		Type:      models.EventUserSignedUp,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"user_id": "123",
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}
	err := bus.Publish(context.Background(), event)
	assert.NoError(t, err)
}

func TestWatermillEventBus_Subscribe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		bus := NewEventBus(getMockConfig(), nil)
		defer bus.Close()

		var wg sync.WaitGroup
		handlerCalled := atomic.Bool{}
		var receivedEvent models.Event

		wg.Add(1)
		err := bus.Subscribe(context.Background(), models.EventUserSignedUp, func(ctx context.Context, event models.Event) error {
			handlerCalled.Store(true)
			receivedEvent = event
			wg.Done()
			return nil
		})
		assert.NoError(t, err)

		event := models.Event{
			Type:      models.EventUserSignedUp,
			Timestamp: time.Now().UTC(),
			Payload: map[string]any{
				"user_id": "456",
			},
			Metadata: map[string]string{"source": "test"},
		}
		err = bus.Publish(context.Background(), event)
		assert.NoError(t, err)

		wg.Wait()
		assert.True(t, handlerCalled.Load())
		assert.Equal(t, models.EventUserSignedUp, receivedEvent.Type)
	})
}

func TestWatermillEventBus_MultipleEvents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		bus := NewEventBus(getMockConfig(), nil)
		defer bus.Close()

		signupCount := atomic.Int32{}
		loginCount := atomic.Int32{}
		var wg sync.WaitGroup

		wg.Add(2)

		err := bus.Subscribe(context.Background(), models.EventUserSignedUp, func(ctx context.Context, event models.Event) error {
			signupCount.Add(1)
			wg.Done()
			return nil
		})
		assert.NoError(t, err)

		err = bus.Subscribe(context.Background(), models.EventUserLoggedIn, func(ctx context.Context, event models.Event) error {
			loginCount.Add(1)
			wg.Done()
			return nil
		})
		assert.NoError(t, err)

		signupEvent := models.Event{
			Type:      models.EventUserSignedUp,
			Timestamp: time.Now().UTC(),
			Payload:   map[string]any{"user_id": "789"},
			Metadata:  map[string]string{"source": "test"},
		}
		loginEvent := models.Event{
			Type:      models.EventUserLoggedIn,
			Timestamp: time.Now().UTC(),
			Payload:   map[string]any{"user_id": "789"},
			Metadata:  map[string]string{"source": "test"},
		}
		bus.Publish(context.Background(), signupEvent)
		bus.Publish(context.Background(), loginEvent)

		wg.Wait()
		assert.Greater(t, signupCount.Load(), int32(0))
		assert.Greater(t, loginCount.Load(), int32(0))
	})
}

func TestWatermillEventBus_EventData(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		bus := NewEventBus(getMockConfig(), nil)
		defer bus.Close()

		var wg sync.WaitGroup
		var receivedPayload map[string]any

		wg.Add(1)
		err := bus.Subscribe(context.Background(), models.EventUserLoggedIn, func(ctx context.Context, event models.Event) error {
			receivedPayload = event.Payload
			wg.Done()
			return nil
		})
		assert.NoError(t, err)

		event := models.Event{
			Type:      models.EventUserLoggedIn,
			Timestamp: time.Now().UTC(),
			Payload: map[string]any{
				"user_id":  "123",
				"username": "testuser",
				"email":    "test@example.com",
			},
			Metadata: map[string]string{"source": "test"},
		}
		err = bus.Publish(context.Background(), event)
		assert.NoError(t, err)

		wg.Wait()
		assert.NotNil(t, receivedPayload)
		assert.Equal(t, "123", receivedPayload["user_id"])
	})
}

func TestEventBus_WithCustomPubSub(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test that EventBus works with a custom PubSub implementation
		config := getMockConfig()
		customPubSub := pubsub.NewInMemoryPubSub()
		bus := NewEventBus(config, customPubSub)
		defer bus.Close()

		var wg sync.WaitGroup
		var receivedEvent models.Event

		wg.Add(1)
		err := bus.Subscribe(context.Background(), models.EventUserSignedUp, func(ctx context.Context, event models.Event) error {
			receivedEvent = event
			wg.Done()
			return nil
		})
		assert.NoError(t, err)

		// Give subscription time to set up
		time.Sleep(10 * time.Millisecond)

		event := models.Event{
			Type:      models.EventUserSignedUp,
			Timestamp: time.Now().UTC(),
			Payload: map[string]any{
				"user_id": "custom-transport-test",
				"email":   "test@example.com",
			},
			Metadata: map[string]string{
				"source": "custom_pubsub_test",
			},
		}

		err = bus.Publish(context.Background(), event)
		assert.NoError(t, err)

		wg.Wait()
		assert.Equal(t, models.EventUserSignedUp, receivedEvent.Type)
		assert.Equal(t, "custom-transport-test", receivedEvent.Payload["user_id"])
	})
}
