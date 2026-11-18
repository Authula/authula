package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
)

var (
	_ message.Publisher  = (*MockWatermillPublisher)(nil)
	_ message.Subscriber = (*MockWatermillSubscriber)(nil)
)

func TestMockWatermillPublisher_Publish(t *testing.T) {
	t.Parallel()

	publishErr := errors.New("publish failed")
	first := message.NewMessage("1", []byte("first"))
	second := message.NewMessage("2", []byte("second"))

	tests := []struct {
		name     string
		topic    string
		messages []*message.Message
		err      error
	}{
		{
			name:  "publishes no messages",
			topic: "events",
		},
		{
			name:     "publishes messages",
			topic:    "events",
			messages: []*message.Message{first, second},
		},
		{
			name:     "returns the configured error",
			topic:    "events",
			messages: []*message.Message{first},
			err:      publishErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			publisher := new(MockWatermillPublisher)
			publisher.On("Publish", tt.topic, tt.messages).Return(tt.err).Once()

			err := publisher.Publish(tt.topic, tt.messages...)

			assert.ErrorIs(t, err, tt.err)
			publisher.AssertExpectations(t)
		})
	}
}

func TestMockWatermillPublisher_Close(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close failed")

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "returns nil when close succeeds",
		},
		{
			name: "returns the configured error",
			err:  closeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			publisher := new(MockWatermillPublisher)
			publisher.On("Close").Return(tt.err).Once()

			err := publisher.Close()

			assert.ErrorIs(t, err, tt.err)
			publisher.AssertExpectations(t)
		})
	}
}

func TestMockWatermillSubscriber_Subscribe(t *testing.T) {
	t.Parallel()

	subscribeErr := errors.New("subscribe failed")
	messages := make(chan *message.Message, 1)
	messages <- message.NewMessage("1", []byte("first"))
	ctx := context.Background()

	tests := []struct {
		name     string
		topic    string
		messages <-chan *message.Message
		err      error
	}{
		{
			name:     "returns the configured messages channel",
			topic:    "events",
			messages: messages,
		},
		{
			name:  "returns the configured error",
			topic: "events",
			err:   subscribeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			subscriber := new(MockWatermillSubscriber)
			subscriber.On("Subscribe", ctx, tt.topic).Return(tt.messages, tt.err).Once()

			got, err := subscriber.Subscribe(ctx, tt.topic)

			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.messages, got)
			subscriber.AssertExpectations(t)
		})
	}
}

func TestMockWatermillSubscriber_Close(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close failed")

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "returns nil when close succeeds",
		},
		{
			name: "returns the configured error",
			err:  closeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			subscriber := new(MockWatermillSubscriber)
			subscriber.On("Close").Return(tt.err).Once()

			err := subscriber.Close()

			assert.ErrorIs(t, err, tt.err)
			subscriber.AssertExpectations(t)
		})
	}
}
