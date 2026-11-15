package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/Authula/authula/models"
)

const defaultRecordingEventBusBuffer = 16

// RecordingEventBus is a test EventBus that records published events and
// exposes them through a buffered channel. It is safe for concurrent use, which
// makes it suitable for services that publish events from a goroutine.
type RecordingEventBus struct {
	mu         sync.Mutex
	published  []models.Event
	events     chan models.Event
	publishErr error
}

func NewRecordingEventBus(bufferSize ...int) *RecordingEventBus {
	size := defaultRecordingEventBusBuffer
	if len(bufferSize) > 0 && bufferSize[0] >= 0 {
		size = bufferSize[0]
	}

	return &RecordingEventBus{events: make(chan models.Event, size)}
}

func (b *RecordingEventBus) Publish(event models.Event) error {
	b.mu.Lock()
	b.published = append(b.published, event)
	err := b.publishErr
	b.mu.Unlock()

	b.events <- event
	return err
}

func (b *RecordingEventBus) Close() error { return nil }

func (b *RecordingEventBus) Subscribe(string, models.EventHandler) (models.SubscriptionID, error) {
	return 0, nil
}

func (b *RecordingEventBus) Unsubscribe(string, models.SubscriptionID) {}

// SetPublishError makes every subsequent Publish call return err.
func (b *RecordingEventBus) SetPublishError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.publishErr = err
}

// Events returns the channel that receives every published event.
func (b *RecordingEventBus) Events() <-chan models.Event {
	return b.events
}

// PublishedEvents returns a snapshot of every event published so far.
func (b *RecordingEventBus) PublishedEvents() []models.Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	events := make([]models.Event, len(b.published))
	copy(events, b.published)
	return events
}

// NextEvent waits for the next published event and fails the test on timeout.
func (b *RecordingEventBus) NextEvent(t *testing.T) models.Event {
	t.Helper()

	select {
	case event := <-b.events:
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
		return models.Event{}
	}
}
