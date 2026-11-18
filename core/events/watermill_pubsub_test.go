package events

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	eventstests "github.com/Authula/authula/core/events/tests"
)

func TestWatermillPubSub_Subscribe_ReturnsSubscriberErrorUnchanged(t *testing.T) {
	t.Parallel()

	errBrokerDown := errors.New("broker down")

	testCases := []struct {
		name          string
		subscriberErr error
	}{
		{
			name:          "plain subscriber error",
			subscriberErr: errBrokerDown,
		},
		{
			name:          "wrapped subscriber error",
			subscriberErr: fmt.Errorf("subscribe orders: %w", errBrokerDown),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			subscriber := new(eventstests.MockWatermillSubscriber)
			subscriber.On("Subscribe", mock.Anything, "orders").Return(nil, tc.subscriberErr).Once()
			pubsub := NewWatermillPubSub(new(eventstests.MockWatermillPublisher), subscriber)

			received, err := pubsub.Subscribe(context.Background(), "orders")

			assert.Equal(t, tc.subscriberErr, err)
			assert.Nil(t, received)
			subscriber.AssertExpectations(t)
		})
	}
}

func TestWatermillPubSub_Subscribe_ForwardsAndAcksMessage(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		watermillMsg := message.NewMessage("msg-1", []byte("payload"))
		watermillMsg.Metadata.Set("key", "value")
		messages := make(chan *message.Message, 1)
		messages <- watermillMsg

		subscriber := new(eventstests.MockWatermillSubscriber)
		subscriber.On("Subscribe", mock.Anything, "orders").
			Return((<-chan *message.Message)(messages), nil).
			Once()
		pubsub := NewWatermillPubSub(new(eventstests.MockWatermillPublisher), subscriber)

		received, err := pubsub.Subscribe(context.Background(), "orders")
		require.NoError(t, err)

		msg := <-received
		assert.Equal(t, "msg-1", msg.UUID)
		assert.Equal(t, []byte("payload"), msg.Payload)
		assert.Equal(t, "value", msg.Metadata["key"])

		close(messages)
		synctest.Wait()

		select {
		case <-watermillMsg.Acked():
		default:
			t.Fatal("expected delivered message to be acked")
		}
		subscriber.AssertExpectations(t)
	})
}

func TestWatermillPubSub_Subscribe_NacksPendingMessageWhenContextCancelled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		watermillMsg := message.NewMessage("msg-2", []byte("pending"))
		messages := make(chan *message.Message, 1)
		messages <- watermillMsg

		subscriber := new(eventstests.MockWatermillSubscriber)
		subscriber.On("Subscribe", mock.Anything, "orders").
			Return((<-chan *message.Message)(messages), nil).
			Once()
		pubsub := NewWatermillPubSub(new(eventstests.MockWatermillPublisher), subscriber)

		received, err := pubsub.Subscribe(ctx, "orders")
		require.NoError(t, err)

		// Nobody reads received, so the forwarder stays blocked until the context ends.
		synctest.Wait()
		cancel()
		synctest.Wait()

		select {
		case <-watermillMsg.Nacked():
		default:
			t.Fatal("expected pending message to be nacked when the context is cancelled")
		}

		select {
		case _, ok := <-received:
			assert.False(t, ok, "forwarding channel must be closed")
		default:
			t.Fatal("expected forwarding channel to close after cancellation")
		}
		subscriber.AssertExpectations(t)
	})
}

func TestWatermillPubSub_Close(t *testing.T) {
	t.Parallel()

	errPublisherClose := errors.New("publisher close failed")
	errSubscriberClose := errors.New("subscriber close failed")

	testCases := []struct {
		name          string
		publisherNil  bool
		subscriberNil bool
		publisherErr  error
		subscriberErr error
		wantErr       error
	}{
		{
			name: "returns nil when both closers succeed",
		},
		{
			name:         "returns the publisher close error",
			publisherErr: errPublisherClose,
			wantErr:      errPublisherClose,
		},
		{
			name:          "returns the subscriber close error",
			subscriberErr: errSubscriberClose,
			wantErr:       errSubscriberClose,
		},
		{
			name:          "prefers the publisher error when both closers fail",
			publisherErr:  errPublisherClose,
			subscriberErr: errSubscriberClose,
			wantErr:       errPublisherClose,
		},
		{
			name:          "skips a nil publisher and still closes the subscriber",
			publisherNil:  true,
			subscriberErr: errSubscriberClose,
			wantErr:       errSubscriberClose,
		},
		{
			name:          "skips a nil subscriber and still closes the publisher",
			subscriberNil: true,
			publisherErr:  errPublisherClose,
			wantErr:       errPublisherClose,
		},
		{
			name:          "returns nil when both components are nil",
			publisherNil:  true,
			subscriberNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var publisher message.Publisher
			publisherMock := new(eventstests.MockWatermillPublisher)
			if !tc.publisherNil {
				publisherMock.On("Close").Return(tc.publisherErr).Once()
				publisher = publisherMock
			}

			var subscriber message.Subscriber
			subscriberMock := new(eventstests.MockWatermillSubscriber)
			if !tc.subscriberNil {
				subscriberMock.On("Close").Return(tc.subscriberErr).Once()
				subscriber = subscriberMock
			}

			pubsub := NewWatermillPubSub(publisher, subscriber)

			err := pubsub.Close()

			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
			publisherMock.AssertExpectations(t)
			subscriberMock.AssertExpectations(t)
		})
	}
}
