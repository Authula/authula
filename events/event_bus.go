package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type eventBus struct {
	config *models.Config
	pubsub models.PubSub
	logger *slog.Logger
}

func NewEventBus(config *models.Config, ps models.PubSub) models.EventBus {
	if config == nil {
		panic("eventbus: config must not be nil")
	}

	if ps == nil {
		ps = NewInMemoryPubSub()
	}

	return &eventBus{
		config: config,
		pubsub: ps,
		logger: slog.Default(),
	}
}

func (b *eventBus) topic(eventType string) string {
	prefix := strings.TrimSuffix(b.config.EventBus.Prefix, ".")
	if prefix == "" {
		return eventType
	}
	return prefix + "." + eventType
}

func (b *eventBus) Publish(ctx context.Context, evt models.Event) error {
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

	msg := &models.Message{
		UUID:    event.ID,
		Payload: payload,
		Metadata: map[string]string{
			"event_type": event.Type,
			"timestamp":  event.Timestamp.Format(time.RFC3339Nano),
		},
	}

	return b.pubsub.Publish(ctx, b.topic(event.Type), msg)
}

func (b *eventBus) Subscribe(
	ctx context.Context,
	eventType string,
	handler models.EventHandler,
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

func (b *eventBus) consume(
	ctx context.Context,
	eventType string,
	msgs <-chan *models.Message,
	handler models.EventHandler,
) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-msgs:
			if !ok {
				return
			}

			var event models.Event
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

func (b *eventBus) Close() error {
	return b.pubsub.Close()
}
