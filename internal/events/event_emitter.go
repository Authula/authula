package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type EventEmitterImpl struct {
	config          *models.Config
	logger          models.Logger
	eventBus        models.EventBus
	webhookExecutor *WebhookExecutor
}

func NewEventEmitter(
	config *models.Config,
	logger models.Logger,
	eventBus models.EventBus,
	webhookExecutor *WebhookExecutor,
) models.EventEmitter {
	return &EventEmitterImpl{
		config:          config,
		logger:          logger,
		eventBus:        eventBus,
		webhookExecutor: webhookExecutor,
	}
}

func (e *EventEmitterImpl) callEventHook(hook func(models.User) error, user *models.User) {
	if user == nil {
		return
	}

	if hook != nil {
		go hook(*user)
	}
}

func (e *EventEmitterImpl) callEventHookWebhook(webhook *models.WebhookConfig, eventType string, user *models.User) {
	// Execute webhook if configured
	if webhook != nil && webhook.URL != "" {
		go func() {
			timeoutSeconds := 30
			if webhook.TimeoutSeconds != 0 {
				timeoutSeconds = webhook.TimeoutSeconds
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
			defer cancel()

			payload := map[string]any{
				"eventType": eventType,
				"user":      user,
				"timestamp": time.Now().UTC(),
			}

			if err := e.webhookExecutor.ExecuteWebhook(ctx, webhook, payload); err != nil {
				e.logger.Error("failed to execute event webhook",
					"event_type", eventType,
					"error", err,
				)
			}
		}()
	}
}

func (e *EventEmitterImpl) emitEvent(eventType string, data any) {
	if e.eventBus == nil {
		return
	}

	// Use a goroutine to keep the call non-blocking
	// TODO: consider a buffered channel + worker pool for extreme high-throughput.
	go func() {
		payload, err := json.Marshal(data)
		if err != nil {
			e.logger.Error("failed to marshal event payload",
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

		if err := e.eventBus.Publish(ctx, event); err != nil {
			e.logger.Error("failed to publish event",
				"event_type", eventType,
				"error", err,
			)
		}
	}()
}

// OnUserSignedUp implements the user signup event logic.
func (e *EventEmitterImpl) OnUserSignedUp(user models.User) {
	e.callEventHook(e.config.EventHooks.OnUserSignedUp, &user)
	e.callEventHookWebhook(e.config.Webhooks.OnUserSignedUp, models.EventUserSignedUp, &user)
}

// OnUserLoggedIn implements the user login event logic.
func (e *EventEmitterImpl) OnUserLoggedIn(user models.User) {
	e.callEventHook(e.config.EventHooks.OnUserLoggedIn, &user)
	e.callEventHookWebhook(e.config.Webhooks.OnUserLoggedIn, models.EventUserLoggedIn, &user)
	e.emitEvent(models.EventUserLoggedIn, user)
}

// OnEmailVerified implements the email verification event logic.
func (e *EventEmitterImpl) OnEmailVerified(user models.User) {
	e.callEventHook(e.config.EventHooks.OnEmailVerified, &user)
	e.callEventHookWebhook(e.config.Webhooks.OnEmailVerified, models.EventEmailVerified, &user)
	e.emitEvent(models.EventEmailVerified, user)
}

// OnEmailChanged implements the email changed event logic.
func (e *EventEmitterImpl) OnEmailChanged(user models.User) {
	e.callEventHook(e.config.EventHooks.OnEmailChanged, &user)
	e.callEventHookWebhook(e.config.Webhooks.OnEmailChanged, models.EventEmailChanged, &user)
	e.emitEvent(models.EventEmailChanged, user)
}

// OnPasswordChanged implements the password changed event logic.
func (e *EventEmitterImpl) OnPasswordChanged(user models.User) {
	e.callEventHook(e.config.EventHooks.OnPasswordChanged, &user)
	e.callEventHookWebhook(e.config.Webhooks.OnPasswordChanged, models.EventPasswordChanged, &user)
	e.emitEvent(models.EventPasswordChanged, user)
}
