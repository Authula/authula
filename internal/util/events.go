package util

import (
	"github.com/Authula/authula/models"
)

// PublishEventAsync publishes an event asynchronously via a goroutine.
func PublishEventAsync(eventBus models.EventBus, logger models.Logger, event models.Event) {
	if eventBus == nil {
		return
	}

	go func(evt models.Event) {
		if err := eventBus.Publish(evt); err != nil {
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
