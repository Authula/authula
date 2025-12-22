package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type EventEmitterImpl struct {
	config          *models.Config
	logger          models.Logger
	eventBus        models.EventBus
	webhookExecutor models.WebhookExecutor
}

func NewEventEmitter(
	config *models.Config,
	logger models.Logger,
	eventBus models.EventBus,
	webhookExecutor models.WebhookExecutor,
) models.EventEmitter {
	return &EventEmitterImpl{
		config:          config,
		logger:          logger,
		eventBus:        eventBus,
		webhookExecutor: webhookExecutor,
	}
}

// getConfig returns the current active config from the manager
func (e *EventEmitterImpl) getConfig() *models.Config {
	return e.config
}

func (e *EventEmitterImpl) callEventHook(hook func(models.User), user *models.User) {
	if user == nil {
		return
	}

	if hook != nil {
		go hook(*user)
	}
}

func (e *EventEmitterImpl) callWebhook(webhook *models.WebhookConfig, eventType string, user *models.User) {
	// Execute webhook if configured
	if webhook != nil && webhook.URL != "" {
		go func() {
			payload := map[string]any{
				"eventType": eventType,
				"user":      user,
				"timestamp": time.Now().UTC(),
			}

			if err := e.webhookExecutor.ExecuteWebhook(webhook, payload); err != nil {
				e.logger.Error(
					"failed to execute event webhook",
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
	cfg := e.getConfig()
	if cfg == nil {
		return
	}
	data, _ := json.Marshal(cfg.Webhooks)
	fmt.Println(string(data))
	e.callEventHook(cfg.EventHooks.OnUserSignedUp, &user)
	e.callWebhook(cfg.Webhooks.OnUserSignedUp, models.EventUserSignedUp, &user)
	e.emitEvent(models.EventUserSignedUp, user)
}

// OnUserLoggedIn implements the user login event logic.
func (e *EventEmitterImpl) OnUserLoggedIn(user models.User) {
	cfg := e.getConfig()
	if cfg == nil {
		return
	}
	e.callEventHook(cfg.EventHooks.OnUserLoggedIn, &user)
	e.callWebhook(cfg.Webhooks.OnUserLoggedIn, models.EventUserLoggedIn, &user)
	e.emitEvent(models.EventUserLoggedIn, user)
}

// OnEmailVerified implements the email verification event logic.
func (e *EventEmitterImpl) OnEmailVerified(user models.User) {
	cfg := e.getConfig()
	if cfg == nil {
		return
	}
	e.callEventHook(cfg.EventHooks.OnEmailVerified, &user)
	e.callWebhook(cfg.Webhooks.OnEmailVerified, models.EventEmailVerified, &user)
	e.emitEvent(models.EventEmailVerified, user)
}

// OnEmailChanged implements the email changed event logic.
func (e *EventEmitterImpl) OnEmailChanged(user models.User) {
	cfg := e.getConfig()
	if cfg == nil {
		return
	}
	e.callEventHook(cfg.EventHooks.OnEmailChanged, &user)
	e.callWebhook(cfg.Webhooks.OnEmailChanged, models.EventEmailChanged, &user)
	e.emitEvent(models.EventEmailChanged, user)
}

// OnPasswordChanged implements the password changed event logic.
func (e *EventEmitterImpl) OnPasswordChanged(user models.User) {
	cfg := e.getConfig()
	if cfg == nil {
		return
	}
	e.callEventHook(cfg.EventHooks.OnPasswordChanged, &user)
	e.callWebhook(cfg.Webhooks.OnPasswordChanged, models.EventPasswordChanged, &user)
	e.emitEvent(models.EventPasswordChanged, user)
}
