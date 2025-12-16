package events

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/GoBetterAuth/go-better-auth/internal/pubsub"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// NewInMemoryPubSub creates a simple in-memory PubSub implementation
func NewInMemoryPubSub() models.PubSub {
	return pubsub.NewInMemoryPubSub()
}

// NewWatermillPubSub creates a PubSub implementation that wraps a Watermill pub/sub.
// This allows you to use any Watermill transport (Kafka, Redis, RabbitMQ, etc.).
func NewWatermillPubSub(pub message.Publisher, sub message.Subscriber) models.PubSub {
	return pubsub.NewWatermillPubSub(pub, sub)
}
