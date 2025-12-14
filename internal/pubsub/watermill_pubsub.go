package pubsub

import (
	"context"
	"maps"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// watermillPubSub adapts Watermill's PubSub to implement our domain.PubSub interface.
// This allows users to use any Watermill transport while keeping our domain independent of Watermill.
type watermillPubSub struct {
	publisher  message.Publisher
	subscriber message.Subscriber
}

// NewWatermillPubSub creates a PubSub adapter for Watermill transports.
// Users can pass any Watermill-compatible publisher and subscriber.
func NewWatermillPubSub(publisher message.Publisher, subscriber message.Subscriber) domain.PubSub {
	return &watermillPubSub{
		publisher:  publisher,
		subscriber: subscriber,
	}
}

// Publish sends a message to the specified topic using Watermill.
func (w *watermillPubSub) Publish(ctx context.Context, topic string, msg *domain.Message) error {
	watermillMsg := message.NewMessage(
		msg.UUID,
		msg.Payload,
	)

	for key, value := range msg.Metadata {
		watermillMsg.Metadata.Set(key, value)
	}

	return w.publisher.Publish(topic, watermillMsg)
}

// Subscribe returns a channel that receives messages from the specified topic.
func (w *watermillPubSub) Subscribe(ctx context.Context, topic string) (<-chan *domain.Message, error) {
	watermillCh, err := w.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return nil, err
	}

	// Create a channel to convert Watermill messages to our domain messages
	domainCh := make(chan *domain.Message)

	go func() {
		defer close(domainCh)

		for watermillMsg := range watermillCh {
			// Convert Watermill message to domain message
			metadata := make(map[string]string)
			maps.Copy(metadata, watermillMsg.Metadata)

			domainMsg := &domain.Message{
				UUID:     watermillMsg.UUID,
				Payload:  watermillMsg.Payload,
				Metadata: metadata,
			}

			select {
			case domainCh <- domainMsg:
				// Message sent successfully
				watermillMsg.Ack()
			case <-ctx.Done():
				// Context cancelled
				watermillMsg.Nack()
				return
			}
		}
	}()

	return domainCh, nil
}

// Close closes both the publisher and subscriber.
func (w *watermillPubSub) Close() error {
	var pubErr, subErr error

	if closer, ok := w.publisher.(interface{ Close() error }); ok {
		pubErr = closer.Close()
	}

	if closer, ok := w.subscriber.(interface{ Close() error }); ok {
		subErr = closer.Close()
	}

	if pubErr != nil {
		return pubErr
	}
	return subErr
}
