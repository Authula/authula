package models

import (
	"context"
	"encoding/json"
	"time"
)

const EventTypeWildcard = "*"

type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata"`
}

type Message struct {
	UUID     string
	Payload  []byte
	Metadata map[string]string
}

type EventPublisher interface {
	Publish(event Event) error
	Close() error
}

type EventHandler func(ctx context.Context, event Event) error

type SubscriptionID uint64

type EventSubscriber interface {
	Subscribe(eventType string, handler EventHandler) (SubscriptionID, error)
	Unsubscribe(eventType string, id SubscriptionID)
	Close() error
}

type PubSub interface {
	Publish(ctx context.Context, topic string, msg *Message) error
	Subscribe(ctx context.Context, topic string) (<-chan *Message, error)
	Close() error
}

type EventBus interface {
	EventPublisher
	EventSubscriber
}
