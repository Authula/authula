package tests

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/mock"
)

// MockWatermillPublisher mocks a Watermill publisher. Close is optional on the
// Watermill interface, so tests wrap it to hide the method when needed.
type MockWatermillPublisher struct {
	mock.Mock
}

func (m *MockWatermillPublisher) Publish(topic string, messages ...*message.Message) error {
	args := m.Called(topic, messages)
	return args.Error(0)
}

func (m *MockWatermillPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockWatermillSubscriber mocks a Watermill subscriber. Close is optional on the
// Watermill interface, so tests wrap it to hide the method when needed.
type MockWatermillSubscriber struct {
	mock.Mock
}

func (m *MockWatermillSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	args := m.Called(ctx, topic)
	messages, _ := args.Get(0).(<-chan *message.Message)
	return messages, args.Error(1)
}

func (m *MockWatermillSubscriber) Close() error {
	args := m.Called()
	return args.Error(0)
}
