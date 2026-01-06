package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/google/uuid"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type handlerEntry struct {
	id      models.SubscriptionID
	handler models.EventHandler
}

type topicState struct {
	handlers []handlerEntry
	cancel   context.CancelFunc
}

type eventBus struct {
	config *models.Config
	pubsub models.PubSub
	logger watermill.LoggerAdapter

	mu     sync.RWMutex
	topics map[string]*topicState

	subIDCounter atomic.Uint64

	// concurrency control
	handlerSem chan struct{}

	// lifecycle
	rootCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewEventBus(config *models.Config, logger watermill.LoggerAdapter, ps models.PubSub) models.EventBus {
	rootCtx, cancel := context.WithCancel(context.Background())

	// Default to 100 concurrent handlers if not configured
	maxHandlers := 100
	if config.EventBus.MaxConcurrentHandlers > 0 {
		maxHandlers = config.EventBus.MaxConcurrentHandlers
	}

	return &eventBus{
		config:     config,
		pubsub:     ps,
		logger:     logger,
		topics:     make(map[string]*topicState),
		handlerSem: make(chan struct{}, maxHandlers),
		rootCtx:    rootCtx,
		cancel:     cancel,
	}
}

func (bus *eventBus) topic(eventType string) string {
	prefix := bus.config.EventBus.Prefix
	if prefix == "" {
		return eventType
	}
	return prefix + "." + eventType
}

func (bus *eventBus) Publish(ctx context.Context, evt models.Event) error {
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

	return bus.pubsub.Publish(ctx, bus.topic(event.Type), msg)
}

func (bus *eventBus) Subscribe(
	eventType string,
	handler models.EventHandler,
) (models.SubscriptionID, error) {
	if handler == nil {
		return 0, fmt.Errorf("eventbus: handler must not be nil")
	}

	topic := bus.topic(eventType)
	id := models.SubscriptionID(bus.subIDCounter.Add(1))

	bus.mu.Lock()
	defer bus.mu.Unlock()

	state, exists := bus.topics[topic]

	// First subscriber → start consumer
	if !exists {
		ctx, cancel := context.WithCancel(bus.rootCtx)

		msgs, err := bus.pubsub.Subscribe(ctx, topic)
		if err != nil {
			cancel()
			return 0, err
		}

		state = &topicState{
			cancel: cancel,
		}
		bus.topics[topic] = state

		bus.wg.Add(1)
		go bus.consumeAndMultiplex(ctx, topic, msgs)
	}

	state.handlers = append(state.handlers, handlerEntry{
		id:      id,
		handler: handler,
	})

	return id, nil
}

func (bus *eventBus) Unsubscribe(eventType string, id models.SubscriptionID) {
	topic := bus.topic(eventType)

	bus.mu.Lock()
	defer bus.mu.Unlock()

	state, ok := bus.topics[topic]
	if !ok {
		return
	}

	handlers := state.handlers
	for i, entry := range handlers {
		if entry.id == id {
			state.handlers = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	// No handlers left → stop consumer
	if len(state.handlers) == 0 {
		state.cancel()
		delete(bus.topics, topic)
	}
}

func (bus *eventBus) consumeAndMultiplex(
	ctx context.Context,
	topic string,
	msgs <-chan *models.Message,
) {
	defer bus.wg.Done()

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
				bus.logger.Error(
					"failed to unmarshal event",
					err,
					watermill.LogFields{
						"topic":      topic,
						"message_id": msg.UUID,
					},
				)
				continue
			}

			bus.mu.RLock()
			state := bus.topics[topic]
			handlers := append([]handlerEntry(nil), state.handlers...)
			bus.mu.RUnlock()

			for _, entry := range handlers {
				bus.handlerSem <- struct{}{}
				bus.wg.Add(1)

				go bus.callHandler(ctx, entry.handler, event)
			}
		}
	}
}

func (bus *eventBus) callHandler(
	ctx context.Context,
	handler models.EventHandler,
	event models.Event,
) {
	defer func() {
		if r := recover(); r != nil {
			bus.logger.Error(
				"event handler panicked",
				fmt.Errorf("panic: %v", r),
				watermill.LogFields{
					"event_type": event.Type,
					"event_id":   event.ID,
				},
			)
		}
		<-bus.handlerSem
		bus.wg.Done()
	}()

	if err := handler(ctx, event); err != nil {
		bus.logger.Error(
			"event handler error",
			err,
			watermill.LogFields{
				"event_type": event.Type,
				"event_id":   event.ID,
			},
		)
	}
}

func (bus *eventBus) Close() error {
	// Stop everything
	bus.cancel()

	// Wait for consumers + handlers
	bus.wg.Wait()

	return bus.pubsub.Close()
}
