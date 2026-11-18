package events

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

const fakeChannelBuffer = 8

type publishedMessage struct {
	topic string
	msg   *models.Message
}

// fakePubSub is a models.PubSub implementation that never touches a broker.
// The first Subscribe call for a topic hands back a channel owned by the test,
// so the test can deliver messages deterministically.
type fakePubSub struct {
	mu sync.Mutex

	published  []publishedMessage
	publishErr error

	subscribeErr      error
	subscribeFailures int
	subscribeCalls    int
	subscribeTopicLog []string
	firstChannels     map[string]chan *models.Message
	subscribedOnce    map[string]bool

	// subscribed receives the topic of every Subscribe call, allowing tests to
	// wait for the consumer loop without polling or sleeping.
	subscribed chan string

	closeCalls int
	closeErr   error
}

func newFakePubSub() *fakePubSub {
	return &fakePubSub{
		firstChannels:  make(map[string]chan *models.Message),
		subscribedOnce: make(map[string]bool),
		subscribed:     make(chan string, 32),
	}
}

func (p *fakePubSub) Publish(_ context.Context, topic string, msg *models.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.publishErr != nil {
		return p.publishErr
	}

	p.published = append(p.published, publishedMessage{topic: topic, msg: msg})
	return nil
}

func (p *fakePubSub) Subscribe(_ context.Context, topic string) (<-chan *models.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.subscribeCalls++
	p.subscribeTopicLog = append(p.subscribeTopicLog, topic)

	if p.subscribeFailures > 0 {
		p.subscribeFailures--
		p.subscribed <- topic
		return nil, p.subscribeErr
	}

	if !p.subscribedOnce[topic] {
		p.subscribedOnce[topic] = true
		ch := p.firstChannel(topic)
		p.subscribed <- topic
		return ch, nil
	}

	// Resubscriptions get a fresh channel so a closed channel does not cause a
	// tight restart loop.
	ch := make(chan *models.Message, fakeChannelBuffer)
	p.subscribed <- topic
	return ch, nil
}

func (p *fakePubSub) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closeCalls++
	return p.closeErr
}

// Channel returns the channel handed to the first Subscribe call for topic.
func (p *fakePubSub) Channel(topic string) chan *models.Message {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.firstChannel(topic)
}

func (p *fakePubSub) firstChannel(topic string) chan *models.Message {
	ch, ok := p.firstChannels[topic]
	if !ok {
		ch = make(chan *models.Message, fakeChannelBuffer)
		p.firstChannels[topic] = ch
	}

	return ch
}

func (p *fakePubSub) publishedMessages() []publishedMessage {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]publishedMessage(nil), p.published...)
}

func (p *fakePubSub) subscribeCallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.subscribeCalls
}

func (p *fakePubSub) subscribeTopics() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]string(nil), p.subscribeTopicLog...)
}

func (p *fakePubSub) closeCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.closeCalls
}

// nextTopic blocks until the consumer loop calls Subscribe. Under synctest this
// also advances the fake clock when the loop is waiting out a backoff.
func (p *fakePubSub) nextTopic(t *testing.T) string {
	t.Helper()

	return <-p.subscribed
}

type logRecord struct {
	level   string
	message string
	err     error
	fields  watermill.LogFields
}

// fakeLogger records log entries so tests can assert on error paths without
// printing noise.
type fakeLogger struct {
	mu      sync.Mutex
	records []logRecord
}

func (l *fakeLogger) record(level, message string, err error, fields watermill.LogFields) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.records = append(l.records, logRecord{
		level:   level,
		message: message,
		err:     err,
		fields:  fields,
	})
}

func (l *fakeLogger) Error(message string, err error, fields watermill.LogFields) {
	l.record("error", message, err, fields)
}

func (l *fakeLogger) Info(string, watermill.LogFields) {}

func (l *fakeLogger) Debug(message string, fields watermill.LogFields) {
	l.record("debug", message, nil, fields)
}

func (l *fakeLogger) Trace(string, watermill.LogFields) {}

func (l *fakeLogger) With(watermill.LogFields) watermill.LoggerAdapter { return l }

func (l *fakeLogger) hasError(substring string) bool {
	return l.hasRecord("error", substring)
}

func (l *fakeLogger) hasDebug(substring string) bool {
	return l.hasRecord("debug", substring)
}

func (l *fakeLogger) hasRecord(level, substring string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, entry := range l.records {
		if entry.level == level && strings.Contains(entry.message, substring) {
			return true
		}
	}

	return false
}

func (l *fakeLogger) errorCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	count := 0
	for _, entry := range l.records {
		if entry.level == "error" {
			count++
		}
	}

	return count
}

func mustEventBus(
	t *testing.T,
	config *models.Config,
	logger watermill.LoggerAdapter,
	pubsub *fakePubSub,
) (models.EventBus, *eventBus) {
	t.Helper()

	bus := NewEventBus(config, logger, pubsub)

	impl, ok := bus.(*eventBus)
	require.True(t, ok, "NewEventBus must return *eventBus")

	return bus, impl
}

func closeEventBus(t *testing.T, bus models.EventBus) {
	t.Helper()

	require.NoError(t, bus.Close())
}

// eventMessage builds the transport message the bus expects for eventType.
func eventMessage(t *testing.T, eventType string) *models.Message {
	t.Helper()

	payload, err := json.Marshal(models.Event{
		ID:        "evt-" + eventType,
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   map[string]any{"key": "payload"},
		Metadata:  map[string]any{"source": "test"},
	})
	require.NoError(t, err)

	return &models.Message{
		UUID:     "evt-" + eventType,
		Payload:  payload,
		Metadata: map[string]string{"event_type": eventType},
	}
}

func noopHandler(context.Context, models.Event) error { return nil }

func TestNewEventBus(t *testing.T) {
	testCases := []struct {
		name        string
		config      *models.Config
		wantMax     int
		wantTimeout time.Duration
	}{
		{
			name:    "empty config defaults to one hundred concurrent handlers",
			config:  &models.Config{},
			wantMax: 100,
		},
		{
			name: "zero concurrency falls back to the default",
			config: &models.Config{
				EventBus: models.EventBusConfig{MaxConcurrentHandlers: 0},
			},
			wantMax: 100,
		},
		{
			name: "negative concurrency falls back to the default",
			config: &models.Config{
				EventBus: models.EventBusConfig{MaxConcurrentHandlers: -5},
			},
			wantMax: 100,
		},
		{
			name: "custom concurrency is honoured",
			config: &models.Config{
				EventBus: models.EventBusConfig{
					MaxConcurrentHandlers: 3,
					ContextTimeout:        2 * time.Second,
				},
			},
			wantMax:     3,
			wantTimeout: 2 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pubsub := newFakePubSub()
			bus, impl := mustEventBus(t, tc.config, watermill.NopLogger{}, pubsub)

			assert.Equal(t, tc.wantMax, cap(impl.handlerSem))
			assert.Zero(t, len(impl.handlerSem))
			assert.Equal(t, tc.wantTimeout, impl.ctxTimeout)
			assert.Same(t, tc.config, impl.config)
			assert.Same(t, pubsub, impl.pubsub)
			assert.Empty(t, impl.topics)
			require.NotNil(t, impl.wildcard)
			assert.Empty(t, impl.wildcard.handlers)
			require.NotNil(t, impl.rootCtx)
			assert.NoError(t, impl.rootCtx.Err())

			closeEventBus(t, bus)
			assert.Equal(t, 1, pubsub.closeCount())
		})
	}
}

func TestEventBus_Topic(t *testing.T) {
	testCases := []struct {
		name      string
		prefix    string
		eventType string
		want      string
	}{
		{
			name:      "empty prefix keeps the event type",
			eventType: "user.created",
			want:      "user.created",
		},
		{
			name:      "prefix is joined with a dot",
			prefix:    "authula",
			eventType: "user.created",
			want:      "authula.user.created",
		},
		{
			name:      "multi segment prefix is joined with a dot",
			prefix:    "authula.dev",
			eventType: "user.created",
			want:      "authula.dev.user.created",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pubsub := newFakePubSub()
			bus, impl := mustEventBus(t, &models.Config{
				EventBus: models.EventBusConfig{Prefix: tc.prefix},
			}, watermill.NopLogger{}, pubsub)
			t.Cleanup(func() { closeEventBus(t, bus) })

			assert.Equal(t, tc.want, impl.topic(tc.eventType))
		})
	}
}

func TestEventBus_Publish(t *testing.T) {
	errPublish := errors.New("transport is down")

	testCases := []struct {
		name          string
		prefix        string
		event         models.Event
		publishErr    error
		wantErr       string
		wantErrIs     error
		wantPublished bool
		check         func(t *testing.T, published publishedMessage)
	}{
		{
			name:          "fills in defaults and publishes the encoded event",
			event:         models.Event{Type: "user.created"},
			wantPublished: true,
			check: func(t *testing.T, published publishedMessage) {
				t.Helper()

				assert.Equal(t, "user.created", published.topic)
				assert.NotEmpty(t, published.msg.UUID)
				assert.Equal(t, "user.created", published.msg.Metadata["event_type"])

				_, err := time.Parse(time.RFC3339Nano, published.msg.Metadata["timestamp"])
				require.NoError(t, err)

				var decoded models.Event
				require.NoError(t, json.Unmarshal(published.msg.Payload, &decoded))
				assert.Equal(t, published.msg.UUID, decoded.ID)
				assert.Equal(t, "user.created", decoded.Type)
				assert.False(t, decoded.Timestamp.IsZero())
				assert.NotNil(t, decoded.Metadata)
			},
		},
		{
			name:          "applies the configured topic prefix",
			prefix:        "authula",
			event:         models.Event{Type: "user.created"},
			wantPublished: true,
			check: func(t *testing.T, published publishedMessage) {
				t.Helper()

				assert.Equal(t, "authula.user.created", published.topic)
			},
		},
		{
			name: "keeps caller supplied fields",
			event: models.Event{
				ID:        "custom-id",
				Type:      "user.created",
				Timestamp: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
				Payload:   map[string]any{"user_id": "user-1"},
				Metadata:  map[string]any{"correlation_id": "abc"},
			},
			wantPublished: true,
			check: func(t *testing.T, published publishedMessage) {
				t.Helper()

				assert.Equal(t, "custom-id", published.msg.UUID)

				var decoded models.Event
				require.NoError(t, json.Unmarshal(published.msg.Payload, &decoded))
				assert.Equal(t, "custom-id", decoded.ID)
				assert.Equal(t, "user-1", decoded.Payload["user_id"])
				assert.Equal(t, "abc", decoded.Metadata["correlation_id"])
				assert.True(t, decoded.Timestamp.Equal(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)))
			},
		},
		{
			name:    "rejects an empty event type",
			event:   models.Event{},
			wantErr: "event type must not be empty",
		},
		{
			name: "returns encoding errors",
			event: models.Event{
				Type:    "user.created",
				Payload: map[string]any{"bad": func() {}},
			},
			wantErr: "json: unsupported type",
		},
		{
			name:       "returns transport errors",
			event:      models.Event{Type: "user.created"},
			publishErr: errPublish,
			wantErrIs:  errPublish,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pubsub := newFakePubSub()
			pubsub.publishErr = tc.publishErr
			logger := &fakeLogger{}
			bus, _ := mustEventBus(t, &models.Config{
				EventBus: models.EventBusConfig{Prefix: tc.prefix},
			}, logger, pubsub)
			t.Cleanup(func() { closeEventBus(t, bus) })

			err := bus.Publish(tc.event)
			switch {
			case tc.wantErrIs != nil:
				require.ErrorIs(t, err, tc.wantErrIs)
			case tc.wantErr != "":
				require.ErrorContains(t, err, tc.wantErr)
			default:
				require.NoError(t, err)
			}

			published := pubsub.publishedMessages()
			if !tc.wantPublished {
				assert.Empty(t, published)
				return
			}

			require.Len(t, published, 1)
			tc.check(t, published[0])
		})
	}
}

func TestEventBus_Subscribe(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "rejects a nil handler",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				id, err := bus.Subscribe("user.created", nil)
				require.ErrorContains(t, err, "handler must not be nil")
				assert.Zero(t, id)
				assert.Empty(t, impl.topics)
				assert.Zero(t, pubsub.subscribeCallCount())
			},
		},
		{
			name: "starts one consumer for a specific topic",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				id, err := bus.Subscribe("user.created", noopHandler)
				require.NoError(t, err)
				assert.Equal(t, models.SubscriptionID(1), id)

				assert.Equal(t, "user.created", pubsub.nextTopic(t))

				state, ok := impl.topics["user.created"]
				require.True(t, ok)
				require.Len(t, state.handlers, 1)
				assert.Equal(t, id, state.handlers[0].id)
				require.NotNil(t, state.cancel)
			},
		},
		{
			name: "registers wildcard handlers without a transport consumer",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				first, err := bus.Subscribe(models.EventTypeWildcard, noopHandler)
				require.NoError(t, err)
				assert.Equal(t, models.SubscriptionID(1), first)

				second, err := bus.Subscribe(models.EventTypeWildcard, noopHandler)
				require.NoError(t, err)
				assert.Equal(t, models.SubscriptionID(2), second)

				assert.Empty(t, impl.topics)
				assert.Zero(t, pubsub.subscribeCallCount())
				require.Len(t, impl.wildcard.handlers, 2)
				assert.Equal(t, first, impl.wildcard.handlers[0].id)
				assert.Equal(t, second, impl.wildcard.handlers[1].id)
			},
		},
		{
			name: "reuses one consumer for every handler on a topic",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				first, err := bus.Subscribe("user.created", noopHandler)
				require.NoError(t, err)
				second, err := bus.Subscribe("user.created", noopHandler)
				require.NoError(t, err)

				assert.Equal(t, models.SubscriptionID(1), first)
				assert.Equal(t, models.SubscriptionID(2), second)

				assert.Equal(t, "user.created", pubsub.nextTopic(t))
				assert.Equal(t, 1, pubsub.subscribeCallCount())

				state, ok := impl.topics["user.created"]
				require.True(t, ok)
				require.Len(t, state.handlers, 2)
			},
		},
		{
			name: "creates a separate consumer per topic",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				_, err := bus.Subscribe("user.created", noopHandler)
				require.NoError(t, err)
				_, err = bus.Subscribe("user.deleted", noopHandler)
				require.NoError(t, err)

				pubsub.nextTopic(t)
				pubsub.nextTopic(t)

				assert.Equal(t, 2, pubsub.subscribeCallCount())
				assert.ElementsMatch(t, []string{"user.created", "user.deleted"}, pubsub.subscribeTopics())
				assert.Len(t, impl.topics, 2)
			},
		},
		{
			name: "applies the configured prefix to the consumer topic",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{
					EventBus: models.EventBusConfig{Prefix: "authula"},
				}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				_, err := bus.Subscribe("user.created", noopHandler)
				require.NoError(t, err)

				assert.Equal(t, "authula.user.created", pubsub.nextTopic(t))
				assert.Contains(t, impl.topics, "authula.user.created")
				assert.NotContains(t, impl.topics, "user.created")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "ignores an unknown topic",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				bus.Unsubscribe("missing.topic", models.SubscriptionID(42))
				assert.Empty(t, impl.topics)
			},
		},
		{
			name: "ignores an unknown subscription id",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
					defer closeEventBus(t, bus)

					id, err := bus.Subscribe("user.created", noopHandler)
					require.NoError(t, err)
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					bus.Unsubscribe("user.created", id+1)

					state, ok := impl.topics["user.created"]
					require.True(t, ok)
					require.Len(t, state.handlers, 1)
					assert.Equal(t, id, state.handlers[0].id)
				})
			},
		},
		{
			name: "stops delivering to the removed handler only",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
					defer closeEventBus(t, bus)

					var mu sync.Mutex
					var removedEvents, keptEvents []models.Event

					removedID, err := bus.Subscribe("user.created", func(_ context.Context, event models.Event) error {
						mu.Lock()
						defer mu.Unlock()
						removedEvents = append(removedEvents, event)
						return nil
					})
					require.NoError(t, err)

					keptID, err := bus.Subscribe("user.created", func(_ context.Context, event models.Event) error {
						mu.Lock()
						defer mu.Unlock()
						keptEvents = append(keptEvents, event)
						return nil
					})
					require.NoError(t, err)
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					bus.Unsubscribe("user.created", removedID)

					state, ok := impl.topics["user.created"]
					require.True(t, ok)
					require.Len(t, state.handlers, 1)
					assert.Equal(t, keptID, state.handlers[0].id)

					pubsub.Channel("user.created") <- eventMessage(t, "user.created")
					synctest.Wait()

					mu.Lock()
					defer mu.Unlock()
					assert.Empty(t, removedEvents)
					require.Len(t, keptEvents, 1)
					assert.Equal(t, "user.created", keptEvents[0].Type)
				})
			},
		},
		{
			name: "removes the last handler and stops the consumer",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
					defer closeEventBus(t, bus)

					id, err := bus.Subscribe("user.created", noopHandler)
					require.NoError(t, err)
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					bus.Unsubscribe("user.created", id)

					assert.NotContains(t, impl.topics, "user.created")
					assert.Empty(t, impl.topics)

					// Closing the bus must not wait for the removed consumer.
					closeEventBus(t, bus)
					assert.Equal(t, 1, pubsub.closeCount())
				})
			},
		},
		{
			name: "removes wildcard handlers by id",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				first, err := bus.Subscribe(models.EventTypeWildcard, noopHandler)
				require.NoError(t, err)
				second, err := bus.Subscribe(models.EventTypeWildcard, noopHandler)
				require.NoError(t, err)

				bus.Unsubscribe(models.EventTypeWildcard, first)

				require.Len(t, impl.wildcard.handlers, 1)
				assert.Equal(t, second, impl.wildcard.handlers[0].id)

				bus.Unsubscribe(models.EventTypeWildcard, models.SubscriptionID(999))
				require.Len(t, impl.wildcard.handlers, 1)
			},
		},
		{
			name: "ignores wildcard ids when no handlers are registered",
			run: func(t *testing.T) {
				pubsub := newFakePubSub()
				bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
				t.Cleanup(func() { closeEventBus(t, bus) })

				bus.Unsubscribe(models.EventTypeWildcard, models.SubscriptionID(1))
				assert.Empty(t, impl.wildcard.handlers)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}

func TestEventBus_ConsumerLoop(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "delivers a message to every handler on the topic",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					bus, _ := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
					defer closeEventBus(t, bus)

					received := make(chan models.Event, 4)
					for range 2 {
						_, err := bus.Subscribe("user.created", func(_ context.Context, event models.Event) error {
							received <- event
							return nil
						})
						require.NoError(t, err)
					}
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					pubsub.Channel("user.created") <- eventMessage(t, "user.created")
					synctest.Wait()

					for range 2 {
						select {
						case event := <-received:
							assert.Equal(t, "user.created", event.Type)
							assert.Equal(t, "payload", event.Payload["key"])
							assert.Equal(t, "test", event.Metadata["source"])
						default:
							t.Fatal("expected every handler to receive the event")
						}
					}
				})
			},
		},
		{
			name: "logs invalid payloads and keeps consuming",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					logger := &fakeLogger{}
					bus, _ := mustEventBus(t, &models.Config{}, logger, pubsub)
					defer closeEventBus(t, bus)

					received := make(chan models.Event, 1)
					_, err := bus.Subscribe("user.created", func(_ context.Context, event models.Event) error {
						received <- event
						return nil
					})
					require.NoError(t, err)
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					channel := pubsub.Channel("user.created")
					channel <- &models.Message{UUID: "bad-message", Payload: []byte("not json")}
					synctest.Wait()

					assert.True(t, logger.hasError("failed to unmarshal event"))
					select {
					case <-received:
						t.Fatal("invalid payload must not reach handlers")
					default:
					}

					channel <- eventMessage(t, "user.created")
					synctest.Wait()

					select {
					case event := <-received:
						assert.Equal(t, "user.created", event.Type)
					default:
						t.Fatal("consumer must keep processing after an invalid payload")
					}
				})
			},
		},
		{
			name: "resubscribes after the transport channel closes",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					bus, _ := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
					defer closeEventBus(t, bus)

					_, err := bus.Subscribe("user.created", noopHandler)
					require.NoError(t, err)
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					close(pubsub.Channel("user.created"))

					// The consumer waits out its restart delay, so this receive also
					// advances the fake clock to the next timer.
					assert.Equal(t, "user.created", pubsub.nextTopic(t))
					assert.Equal(t, 2, pubsub.subscribeCallCount())
					assert.Equal(t, []string{"user.created", "user.created"}, pubsub.subscribeTopics())
				})
			},
		},
		{
			name: "retries after a subscribe failure",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					errSubscribe := errors.New("broker unavailable")
					pubsub := newFakePubSub()
					pubsub.subscribeErr = errSubscribe
					pubsub.subscribeFailures = 1

					logger := &fakeLogger{}
					bus, _ := mustEventBus(t, &models.Config{}, logger, pubsub)
					defer closeEventBus(t, bus)

					received := make(chan models.Event, 1)
					_, err := bus.Subscribe("user.created", func(_ context.Context, event models.Event) error {
						received <- event
						return nil
					})
					require.NoError(t, err)

					require.Equal(t, "user.created", pubsub.nextTopic(t))
					synctest.Wait()
					assert.True(t, logger.hasError("failed to subscribe to topic"))

					// The retry only happens after the backoff timer fires.
					require.Equal(t, "user.created", pubsub.nextTopic(t))
					assert.Equal(t, 2, pubsub.subscribeCallCount())

					pubsub.Channel("user.created") <- eventMessage(t, "user.created")
					synctest.Wait()

					select {
					case event := <-received:
						assert.Equal(t, "user.created", event.Type)
					default:
						t.Fatal("expected delivery after the consumer recovered")
					}
				})
			},
		},
		{
			name: "caps the retry backoff and stops waiting on close",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					const failures = 7

					pubsub := newFakePubSub()
					pubsub.subscribeErr = errors.New("broker unavailable")
					pubsub.subscribeFailures = failures

					logger := &fakeLogger{}
					bus, _ := mustEventBus(t, &models.Config{}, logger, pubsub)

					_, err := bus.Subscribe("user.created", noopHandler)
					require.NoError(t, err)

					// Each receive advances the fake clock through one backoff
					// window, including the capped 30s window.
					for attempt := 1; attempt <= failures; attempt++ {
						require.Equal(t, "user.created", pubsub.nextTopic(t))
					}
					assert.Equal(t, failures, pubsub.subscribeCallCount())
					assert.True(t, logger.hasError("failed to subscribe to topic"))

					// The consumer is waiting out the capped backoff; closing the
					// bus cancels the wait instead of retrying forever.
					closeEventBus(t, bus)
					assert.Equal(t, failures, pubsub.subscribeCallCount())
					assert.Equal(t, 1, pubsub.closeCount())
				})
			},
		},
		{
			name: "cancels the restart delay when the bus closes",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					logger := &fakeLogger{}
					bus, _ := mustEventBus(t, &models.Config{}, logger, pubsub)

					_, err := bus.Subscribe("user.created", noopHandler)
					require.NoError(t, err)
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					close(pubsub.Channel("user.created"))
					synctest.Wait()
					assert.True(t, logger.hasDebug("Consuming done"))

					closeEventBus(t, bus)
					assert.Equal(t, 1, pubsub.subscribeCallCount())
					assert.Equal(t, 1, pubsub.closeCount())
				})
			},
		},
		{
			name: "limits concurrent handlers",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					bus, _ := mustEventBus(t, &models.Config{
						EventBus: models.EventBusConfig{MaxConcurrentHandlers: 1},
					}, watermill.NopLogger{}, pubsub)
					defer closeEventBus(t, bus)

					var inFlight atomic.Int32
					started := make(chan struct{}, 4)
					release := make(chan struct{}, 4)

					_, err := bus.Subscribe("user.created", func(_ context.Context, _ models.Event) error {
						inFlight.Add(1)
						started <- struct{}{}
						<-release
						inFlight.Add(-1)
						return nil
					})
					require.NoError(t, err)
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					channel := pubsub.Channel("user.created")
					channel <- eventMessage(t, "user.created")
					<-started

					channel <- eventMessage(t, "user.created")
					synctest.Wait()

					assert.Equal(t, int32(1), inFlight.Load())
					select {
					case <-started:
						t.Fatal("a second handler started while the concurrency limit was reached")
					default:
					}

					release <- struct{}{}
					<-started
					assert.Equal(t, int32(1), inFlight.Load())

					release <- struct{}{}
					synctest.Wait()
					assert.Zero(t, inFlight.Load())
				})
			},
		},
		{
			name: "cancels running handlers when the bus closes",
			run: func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					pubsub := newFakePubSub()
					bus, _ := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)

					started := make(chan struct{}, 1)
					handlerResult := make(chan error, 1)
					_, err := bus.Subscribe("user.created", func(ctx context.Context, _ models.Event) error {
						started <- struct{}{}
						<-ctx.Done()
						handlerResult <- ctx.Err()
						return ctx.Err()
					})
					require.NoError(t, err)
					require.Equal(t, "user.created", pubsub.nextTopic(t))

					pubsub.Channel("user.created") <- eventMessage(t, "user.created")
					<-started

					closeEventBus(t, bus)

					select {
					case handlerErr := <-handlerResult:
						require.ErrorIs(t, handlerErr, context.Canceled)
					default:
						t.Fatal("handler must observe cancellation when the bus closes")
					}
					assert.Equal(t, 1, pubsub.closeCount())
				})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}

func TestEventBus_WildcardDispatchContext(t *testing.T) {
	type deadlineResult struct {
		deadline time.Time
		ok       bool
	}

	synctest.Test(t, func(t *testing.T) {
		pubsub := newFakePubSub()
		bus, _ := mustEventBus(t, &models.Config{
			EventBus: models.EventBusConfig{ContextTimeout: time.Minute},
		}, watermill.NopLogger{}, pubsub)
		defer closeEventBus(t, bus)

		results := make(chan deadlineResult, 1)
		_, err := bus.Subscribe(models.EventTypeWildcard, func(ctx context.Context, _ models.Event) error {
			deadline, ok := ctx.Deadline()
			results <- deadlineResult{deadline: deadline, ok: ok}
			return nil
		})
		require.NoError(t, err)

		require.NoError(t, bus.Publish(models.Event{Type: "user.created"}))

		result := <-results
		require.True(t, result.ok, "wildcard handlers must receive a deadline")
		assert.WithinDuration(t, time.Now().Add(time.Minute), result.deadline, time.Second)
	})
}

func TestEventBus_ConsumeAndMultiplexWithUnregisteredTopic(t *testing.T) {
	pubsub := newFakePubSub()
	bus, impl := mustEventBus(t, &models.Config{}, watermill.NopLogger{}, pubsub)
	t.Cleanup(func() { closeEventBus(t, bus) })

	// A message can be in flight for a topic whose last handler was removed,
	// which deletes the topic state.
	messages := make(chan *models.Message, 1)
	messages <- eventMessage(t, "user.created")
	close(messages)

	assert.NotPanics(t, func() {
		impl.consumeAndMultiplex(context.Background(), "user.created", messages)
	})
	assert.Empty(t, impl.topics)
}

func TestEventBus_CallHandler(t *testing.T) {
	errHandler := errors.New("handler failed")

	testCases := []struct {
		name         string
		handler      models.EventHandler
		wantCalled   bool
		wantErrorLog string
	}{
		{
			name: "invokes the handler",
			handler: func(context.Context, models.Event) error {
				return nil
			},
			wantCalled: true,
		},
		{
			name: "logs handler errors without propagating them",
			handler: func(context.Context, models.Event) error {
				return errHandler
			},
			wantErrorLog: "event handler error",
		},
		{
			name: "recovers handler panics",
			handler: func(context.Context, models.Event) error {
				panic("boom")
			},
			wantErrorLog: "event handler panicked",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pubsub := newFakePubSub()
			logger := &fakeLogger{}
			bus, impl := mustEventBus(t, &models.Config{}, logger, pubsub)
			t.Cleanup(func() { closeEventBus(t, bus) })

			called := false
			handler := tc.handler
			if tc.wantCalled {
				handler = func(ctx context.Context, event models.Event) error {
					called = true
					return tc.handler(ctx, event)
				}
			}

			event := models.Event{ID: "event-1", Type: "user.created"}

			impl.handlerSem <- struct{}{}
			impl.wg.Add(1)
			assert.NotPanics(t, func() {
				impl.callHandler(context.Background(), handler, event)
			})

			assert.Equal(t, tc.wantCalled, called)
			assert.Zero(t, len(impl.handlerSem), "handler semaphore must be released")

			if tc.wantErrorLog == "" {
				assert.Zero(t, logger.errorCount())
				return
			}

			assert.True(t, logger.hasError(tc.wantErrorLog))
		})
	}
}
