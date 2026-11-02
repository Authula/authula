package tests

import (
	"context"
	"sync"

	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/models"
)

type MockLogger struct{}

func (m *MockLogger) Debug(msg string, args ...any) {}
func (m *MockLogger) Info(msg string, args ...any)  {}
func (m *MockLogger) Warn(msg string, args ...any)  {}
func (m *MockLogger) Error(msg string, args ...any) {}

// RecordingLogger records error messages so tests can assert that errors
// were logged, e.g. after-hooks that must not fail the operation.
type RecordingLogger struct {
	ErrorCalls []string
}

func (l *RecordingLogger) Debug(msg string, args ...any) {}
func (l *RecordingLogger) Info(msg string, args ...any)  {}
func (l *RecordingLogger) Warn(msg string, args ...any)  {}
func (l *RecordingLogger) Error(msg string, args ...any) {
	l.ErrorCalls = append(l.ErrorCalls, msg)
}

// LogRecord is one captured call made to a LevelRecordingLogger.
type LogRecord struct {
	Level   string
	Message string
	Args    []any
}

// LevelRecordingLogger records every call with its severity. It is safe for
// concurrent use, which makes it suitable for fan-out tests.
type LevelRecordingLogger struct {
	mu      sync.Mutex
	records []LogRecord
}

func (l *LevelRecordingLogger) Debug(msg string, args ...any) { l.record("debug", msg, args) }
func (l *LevelRecordingLogger) Info(msg string, args ...any)  { l.record("info", msg, args) }
func (l *LevelRecordingLogger) Warn(msg string, args ...any)  { l.record("warn", msg, args) }
func (l *LevelRecordingLogger) Error(msg string, args ...any) { l.record("error", msg, args) }

func (l *LevelRecordingLogger) record(level, msg string, args []any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.records = append(l.records, LogRecord{Level: level, Message: msg, Args: args})
}

// Records returns a copy of every captured call, in order.
func (l *LevelRecordingLogger) Records() []LogRecord {
	l.mu.Lock()
	defer l.mu.Unlock()

	records := make([]LogRecord, len(l.records))
	copy(records, l.records)
	return records
}

// Messages returns the message of every captured call, in order.
func (l *LevelRecordingLogger) Messages() []string {
	records := l.Records()
	messages := make([]string, len(records))
	for i, record := range records {
		messages[i] = record.Message
	}
	return messages
}

// PanicLogger panics on every call. It is used to prove that a broken sink
// cannot propagate into the caller.
type PanicLogger struct{}

func (l *PanicLogger) Debug(msg string, args ...any) { panic("panic logger: debug") }
func (l *PanicLogger) Info(msg string, args ...any)  { panic("panic logger: info") }
func (l *PanicLogger) Warn(msg string, args ...any)  { panic("panic logger: warn") }
func (l *PanicLogger) Error(msg string, args ...any) { panic("panic logger: error") }

type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(event models.Event) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *MockEventBus) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(topic string, handler models.EventHandler) (models.SubscriptionID, error) {
	args := m.Called(topic, handler)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(models.SubscriptionID), args.Error(1)
}

func (m *MockEventBus) Unsubscribe(topic string, subscriptionID models.SubscriptionID) {
	m.Called(topic, subscriptionID)
}

type TestPluginRegistry struct{}

func (r *TestPluginRegistry) Register(p models.Plugin) error           { return nil }
func (r *TestPluginRegistry) InitAll() error                           { return nil }
func (r *TestPluginRegistry) RunMigrations(ctx context.Context) error  { return nil }
func (r *TestPluginRegistry) DropMigrations(ctx context.Context) error { return nil }
func (r *TestPluginRegistry) Plugins() []models.Plugin                 { return nil }
func (r *TestPluginRegistry) GetConfig() *models.Config                { return nil }
func (r *TestPluginRegistry) CloseAll()                                {}
func (r *TestPluginRegistry) GetPlugin(pluginID string) models.Plugin  { return nil }

type TestServiceRegistry struct {
	Services map[string]any
}

func (r *TestServiceRegistry) Register(name string, service any) {}
func (r *TestServiceRegistry) Get(name string) any {
	if r == nil || r.Services == nil {
		return nil
	}
	return r.Services[name]
}
