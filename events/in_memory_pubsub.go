package events

import (
	"context"
	"sync"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type InMemoryPubSub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *models.Message
	closed      bool
	closedChans []chan *models.Message
}

// NewInMemoryPubSub creates an in-memory PubSub implementation.
func NewInMemoryPubSub() models.PubSub {
	return &InMemoryPubSub{
		subscribers: make(map[string][]chan *models.Message),
		closedChans: make([]chan *models.Message, 0),
	}
}

func (s *InMemoryPubSub) Publish(ctx context.Context, topic string, msg *models.Message) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil
	}

	subscribers, exists := s.subscribers[topic]
	if !exists {
		return nil
	}

	for _, ch := range subscribers {
		select {
		case ch <- msg:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full, skip
		}
	}

	return nil
}

func (s *InMemoryPubSub) Subscribe(ctx context.Context, topic string) (<-chan *models.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		ch := make(chan *models.Message)
		close(ch)
		return ch, nil
	}

	// Create buffered channel for this subscriber
	ch := make(chan *models.Message, 100)

	s.subscribers[topic] = append(s.subscribers[topic], ch)
	s.closedChans = append(s.closedChans, ch)

	return ch, nil
}

func (s *InMemoryPubSub) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	for _, ch := range s.closedChans {
		close(ch)
	}

	s.subscribers = nil
	s.closedChans = nil

	return nil
}
