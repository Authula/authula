package util

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// PublishEventAsync publishes an event asynchronously without blocking the caller.
// If the event bus is nil, it safely returns without error.
// The event is published in a background goroutine with a timeout context.
// Event publishing failures are logged but do not block or return errors to the caller,
// treating events as telemetry rather than critical to request success.
func PublishEventAsync(eventBus models.EventBus, logger models.Logger, event models.Event) {
	if eventBus == nil {
		return
	}

	go func(evt models.Event) {
		// Create a timeout context for the async publish operation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := eventBus.Publish(ctx, evt); err != nil {
			// Log error but don't fail the request - events are telemetry, not part of request contract
			if logger != nil {
				logger.Error("failed to publish event asynchronously",
					"event_type", evt.Type,
					"event_id", evt.ID,
					"error", err,
				)
			}
		}
	}(event)
}
