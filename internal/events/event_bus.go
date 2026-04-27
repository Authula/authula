package events

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/google/uuid"

	"github.com/Authula/authula/models"
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

	mu       sync.RWMutex
	topics   map[string]*topicState
	wildcard *topicState

	subIDCounter atomic.Uint64

	// concurrency control
	handlerSem chan struct{}

	// lifecycle
	rootCtx    context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	ctxTimeout time.Duration
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
		wildcard:   &topicState{},
		handlerSem: make(chan struct{}, maxHandlers),
		rootCtx:    rootCtx,
		cancel:     cancel,
		ctxTimeout: config.EventBus.ContextTimeout,
	}
}

func (bus *eventBus) topic(eventType string) string {
	prefix := bus.config.EventBus.Prefix
	if prefix == "" {
		return eventType
	}
	return prefix + "." + eventType
}

func (bus *eventBus) Publish(evt models.Event) error {
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

	ctx, cancel := context.WithTimeout(bus.rootCtx, bus.ctxTimeout)
	defer cancel()

	msg := &models.Message{
		UUID:    event.ID,
		Payload: payload,
		Metadata: map[string]string{
			"event_type": event.Type,
			"timestamp":  event.Timestamp.Format(time.RFC3339Nano),
		},
	}
	if err := bus.pubsub.Publish(ctx, bus.topic(event.Type), msg); err != nil {
		return err
	}

	bus.dispatchForWildcard(event)

	return nil
}

func (bus *eventBus) Subscribe(
	eventType string,
	handler models.EventHandler,
) (models.SubscriptionID, error) {
	if handler == nil {
		return 0, fmt.Errorf("eventbus: handler must not be nil")
	}

	if eventType == models.EventTypeWildcard {
		id := models.SubscriptionID(bus.subIDCounter.Add(1))

		bus.mu.Lock()
		defer bus.mu.Unlock()

		bus.wildcard.handlers = append(bus.wildcard.handlers, handlerEntry{
			id:      id,
			handler: handler,
		})

		return id, nil
	}

	topic := bus.topic(eventType)
	id := models.SubscriptionID(bus.subIDCounter.Add(1))

	bus.mu.Lock()
	defer bus.mu.Unlock()

	state, exists := bus.topics[topic]

	// First subscriber start resilient consumer loop that re-subscribes on transport disconnects
	if !exists {
		ctx, cancel := context.WithCancel(bus.rootCtx)

		state = &topicState{
			cancel: cancel,
		}
		bus.topics[topic] = state

		bus.wg.Add(1)
		go bus.startConsumerLoop(ctx, topic)
	}

	state.handlers = append(state.handlers, handlerEntry{
		id:      id,
		handler: handler,
	})

	return id, nil
}

func (bus *eventBus) Unsubscribe(eventType string, id models.SubscriptionID) {
	if eventType == models.EventTypeWildcard {
		bus.mu.Lock()
		defer bus.mu.Unlock()

		handlers := bus.wildcard.handlers
		for i, entry := range handlers {
			if entry.id == id {
				bus.wildcard.handlers = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}

		return
	}

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

func (bus *eventBus) dispatchForWildcard(event models.Event) {
	bus.mu.RLock()
	handlers := append([]handlerEntry(nil), bus.wildcard.handlers...)
	bus.mu.RUnlock()

	for _, entry := range handlers {
		bus.handlerSem <- struct{}{}
		bus.wg.Add(1)

		handlerCtx, cancel := context.WithTimeout(bus.rootCtx, bus.ctxTimeout)

		go func(handler models.EventHandler, ctx context.Context) {
			defer cancel()
			bus.callHandler(ctx, handler, event)
		}(entry.handler, handlerCtx)
	}
}

func (bus *eventBus) consumeAndMultiplex(
	ctx context.Context,
	topic string,
	msgs <-chan *models.Message,
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
			if state == nil {
				bus.mu.RUnlock()
				continue
			}
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

func (bus *eventBus) startConsumerLoop(ctx context.Context, topic string) {
	defer bus.wg.Done()

	// Exponential backoff settings
	const (
		baseBackoff = 500 * time.Millisecond
		maxBackoff  = 30 * time.Second
	)
	backoff := baseBackoff

	for {
		msgs, err := bus.pubsub.Subscribe(ctx, topic)
		if err != nil {
			// Add jitter to avoid thundering herd when many consumers retry
			jitter := time.Duration(rand.Int63n(int64(250 * time.Millisecond)))
			wait := backoff + jitter

			bus.logger.Error(
				"failed to subscribe to topic, will retry",
				err,
				watermill.LogFields{"topic": topic, "retry_in_ms": wait.Milliseconds()},
			)

			select {
			case <-time.After(wait):
				// increase backoff exponentially up to cap
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			case <-ctx.Done():
				return
			}
		}

		// reset backoff on successful subscribe
		backoff = baseBackoff

		bus.logger.Debug(
			"Starting consuming",
			watermill.LogFields{"topic": topic},
		)

		bus.consumeAndMultiplex(ctx, topic, msgs)

		bus.logger.Debug(
			"Consuming done",
			watermill.LogFields{"topic": topic},
		)

		// If context is cancelled, stop retrying
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Small delay to avoid tight restart loops
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return
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
