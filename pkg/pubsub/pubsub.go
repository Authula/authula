package pubsub

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/GoBetterAuth/go-better-auth/internal/pubsub"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// NewInMemoryPubSub creates a simple in-memory PubSub implementation
// with no external dependencies (not even Watermill).
// This is the most lightweight option.
func NewInMemoryPubSub() domain.PubSub {
	return pubsub.NewInMemoryPubSub()
}

// NewWatermillPubSub creates a PubSub implementation that wraps a Watermill pub/sub.
// This allows you to use any Watermill transport (Kafka, Redis, RabbitMQ, etc.).
func NewWatermillPubSub(pub message.Publisher, sub message.Subscriber) domain.PubSub {
	return pubsub.NewWatermillPubSub(pub, sub)
}
