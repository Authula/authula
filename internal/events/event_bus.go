package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/GoBetterAuth/go-better-auth/internal/pubsub"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// EventBus implements domain.EventBus using a PubSub transport.
type EventBus struct {
	config *domain.Config
	pubsub domain.PubSub
	logger *slog.Logger
}

func NewEventBus(config *domain.Config, ps domain.PubSub) domain.EventBus {
	if config == nil {
		panic("eventbus: config must not be nil")
	}

	if ps == nil {
		ps = pubsub.NewInMemoryPubSub()
	}

	return &EventBus{
		config: config,
		pubsub: ps,
		logger: slog.Default(),
	}
}

func (b *EventBus) topic(eventType string) string {
	prefix := strings.TrimSuffix(b.config.EventBus.Prefix, ".")
	if prefix == "" {
		return eventType
	}
	return prefix + "." + eventType
}

func (b *EventBus) Publish(ctx context.Context, evt domain.Event) error {
	// Copy to avoid mutating caller-owned data
	event := evt
	if event.Type == "" {
		return fmt.Errorf("eventbus: event type must not be empty")
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := &domain.Message{
		UUID:    event.ID,
		Payload: payload,
		Metadata: map[string]string{
			"event_type": event.Type,
			"timestamp":  event.Timestamp.Format(time.RFC3339Nano),
		},
	}

	return b.pubsub.Publish(ctx, b.topic(event.Type), msg)
}

func (b *EventBus) Subscribe(
	ctx context.Context,
	eventType string,
	handler domain.EventHandler,
) error {
	if handler == nil {
		return fmt.Errorf("eventbus: event handler must not be nil")
	}

	msgs, err := b.pubsub.Subscribe(ctx, b.topic(eventType))
	if err != nil {
		return err
	}

	go b.consume(ctx, eventType, msgs, handler)

	return nil
}

func (b *EventBus) consume(
	ctx context.Context,
	eventType string,
	msgs <-chan *domain.Message,
	handler domain.EventHandler,
) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-msgs:
			if !ok {
				return
			}

			var event domain.Event
			if err := json.Unmarshal(msg.Payload, &event); err != nil {
				b.logger.Error(
					"failed to unmarshal event",
					"error", err,
					"topic", b.topic(eventType),
					"message_id", msg.UUID,
				)
				continue
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						b.logger.Error(
							"event handler panicked",
							"panic", r,
							"event_type", event.Type,
							"event_id", event.ID,
						)
					}
				}()

				if err := handler(ctx, event); err != nil {
					b.logger.Error(
						"event handler error",
						"error", err,
						"event_type", event.Type,
						"event_id", event.ID,
					)
				}
			}()
		}
	}
}

func (b *EventBus) Close() error {
	return b.pubsub.Close()
}
