// Package domain provides the core types and interfaces for the event-driven plugin system.
//
// # Event-Driven Architecture
//
// The GoBetterAuth library uses the Watermill library to provide a robust event-driven
// system for plugins. Plugins can publish and subscribe to events that occur throughout
// the authentication lifecycle.
//
// # Using the Event Bus
//
// ## Example: Creating a Plugin that Reacts to Events
//
// ```go
// package myplugin
//
// import (
//
//	"context"
//	"encoding/json"
//	"log/slog"
//
//	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
//	"github.com/ThreeDotsLabs/watermill/message"
//
// )
//
//	type MyPlugin struct {
//	    config    domain.PluginConfig
//	    eventBus  *domain.PluginEventBus
//	}
//
//	func (p *MyPlugin) Init(ctx *domain.PluginContext) error {
//	    // Initialize the event bus (would be passed during plugin setup)
//	    // For now, this is a placeholder
//	    return nil
//	}
//
//	func (p *MyPlugin) EventHooks() *domain.PluginEventHooks {
//	    // Return the event bus reference for the framework to use
//	    return p.eventBus
//	}
//
// // Subscribe to authentication events
//
//	func (p *MyPlugin) subscribeToEvents(ctx context.Context) error {
//	    // Subscribe to user signup events
//	    signupHandler := func(ctx context.Context, msg *message.Message) error {
//	        var payload domain.EventPayload
//	        if err := json.Unmarshal(msg.Payload, &payload); err != nil {
//	            slog.Error("Failed to parse signup event", slog.Any("error", err))
//	            return err
//	        }
//
//	        slog.Info("User signed up", slog.Any("data", payload.Data))
//	        // Perform your plugin logic here
//	        return nil
//	    }
//
//	    // Subscribe to login events
//	    loginHandler := func(ctx context.Context, msg *message.Message) error {
//	        var payload domain.EventPayload
//	        if err := json.Unmarshal(msg.Payload, &payload); err != nil {
//	            return err
//	        }
//
//	        slog.Info("User logged in", slog.Any("data", payload.Data))
//	        return nil
//	    }
//
//	    // Subscribe to multiple events at once
//	    subscriptions := []domain.EventSubscription{
//	        {EventType: domain.EventUserSignedUp, Handler: signupHandler},
//	        {EventType: domain.EventUserLoggedIn, Handler: loginHandler},
//	    }
//
//	    return p.eventBus.SubscribeMultiple(ctx, subscriptions...)
//	}
//
// ```
//
// # Available Events
//
// The following events are emitted by the GoBetterAuth library:
//
//   - EventUserSignedUp: Emitted when a new user successfully signs up
//   - EventUserLoggedIn: Emitted when a user logs in
//   - EventEmailVerified: Emitted when a user verifies their email
//   - EventPasswordChanged: Emitted when a user changes their password
//   - EventEmailChanged: Emitted when a user changes their email address
//
// # Event Payload Structure
//
// Each event contains an EventPayload with the following structure:
//
//	type EventPayload struct {
//	    Data      any    // The actual event data (typically the affected entity)
//	    Timestamp int64  // Unix timestamp when the event occurred
//	    Source    string // The source of the event (e.g., "auth_service")
//	}
//
// # Publishing Custom Events
//
// Plugins can also publish their own events:
//
//	customPayload := domain.EventPayload{
//	    Data:      map[string]string{"plugin_data": "value"},
//	    Timestamp: time.Now().Unix(),
//	    Source:    "my_plugin",
//	}
//
//	if err := eventBus.Publish(ctx, "custom.event.type", customPayload); err != nil {
//	    slog.Error("Failed to publish event", slog.Any("error", err))
//	}
//
// # Event Handler Guidelines
//
//   - Event handlers should be idempotent (safe to be called multiple times with the same data)
//   - Handlers should complete quickly; for long-running tasks, consider queueing them
//   - Always handle errors gracefully and return them from the handler
//   - Handlers run concurrently, so ensure any shared state is thread-safe
//   - Subscribe to events early in your plugin's initialization
//
// # Implementation Details
//
// The event system is powered by Watermill's in-memory pubsub (gochannel) implementation.
// For single-instance deployments, this is sufficient. For distributed systems, you can
// replace the InMemoryEventBus with Watermill's other pubsub implementations like Kafka,
// RabbitMQ, or AWS SNS/SQS.
package domain
