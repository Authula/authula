package models

import (
	"context"
	"encoding/json"
	"time"
)

// Event represents data to be published or received via the EventBus
type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata"`
}

// Message represents a message in the pub/sub system.
type Message struct {
	UUID     string
	Payload  []byte // Message payload (serialized data)
	Metadata map[string]string
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
	Close() error
}

// EventHandler processes events
type EventHandler func(ctx context.Context, event Event) error

// SubscriptionID identifies a specific event handler subscription for removal
type SubscriptionID uint64

// EventSubscriber defines the interface for subscribing to events
type EventSubscriber interface {
	Subscribe(eventType string, handler EventHandler) (SubscriptionID, error)
	Unsubscribe(eventType string, id SubscriptionID)
	Close() error
}

// PubSub is a generic publish-subscribe interface.
type PubSub interface {
	// Publish sends a message to the specified topic
	Publish(ctx context.Context, topic string, msg *Message) error

	// Subscribe returns a channel that receives messages from the specified topic.
	// The channel should be closed when the subscription is cancelled or closed.
	Subscribe(ctx context.Context, topic string) (<-chan *Message, error)

	// Close closes the pub/sub and cleans up resources
	Close() error
}

// EventBus combines publisher and subscriber functionality
type EventBus interface {
	EventPublisher
	EventSubscriber
}
