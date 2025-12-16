package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// callHook safely calls a hook function if it's not nil
func (s *Service) callHook(hook func(models.User) error, user *models.User) {
	if hook != nil && user != nil {
		go hook(*user)
	}
}

// emitEvent publishes an event to the EventBus
func (s *Service) emitEvent(eventType string, data any) {
	if s.EventBus == nil {
		return
	}

	// Use a goroutine to keep the call non-blocking,
	// TODO: consider a buffered channel + worker pool for extreme high-throughput.
	go func() {
		payload, err := json.Marshal(data)
		if err != nil {
			slog.Error("failed to marshal event payload",
				"event_type", eventType,
				"error", err,
			)
			return
		}

		event := models.Event{
			Type:      eventType,
			Timestamp: time.Now().UTC(),
			Payload:   payload,
			Metadata: map[string]string{
				"source": "auth_service",
			},
		}

		// Use a context with a timeout so a hung EventBus doesn't leak goroutines
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.EventBus.Publish(ctx, event); err != nil {
			slog.Error("failed to publish event",
				"event_type", eventType,
				"error", err,
			)
		}
	}()
}
