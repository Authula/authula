package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// callHook safely calls a hook function if it's not nil
func (s *Service) callHook(hook func(domain.User) error, user *domain.User) {
	if hook != nil && user != nil {
		go hook(*user)
	}
}

// emitEvent safely emits an event using a hook function if it's not nil
func (s *Service) emitEvent(eventType string, data any) {
	if s.EventBus != nil {
		go func() {
			// Convert data to map[string]any for payload
			payload, ok := data.(map[string]any)
			if !ok {
				payload = map[string]any{
					"data": data,
				}
			}

			event := domain.Event{
				Type:      eventType,
				Timestamp: time.Now().UTC(),
				Payload:   payload,
				Metadata: map[string]string{
					"source": "auth_service",
				},
			}

			if err := s.EventBus.Publish(context.Background(), event); err != nil {
				slog.Error("failed to publish event", "event_type", eventType, "error", err)
			}
		}()
	}
}
