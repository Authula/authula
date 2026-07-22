package tests

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/models"
)

type MockLogger struct{}

func (m *MockLogger) Debug(msg string, args ...any) {}
func (m *MockLogger) Info(msg string, args ...any)  {}
func (m *MockLogger) Warn(msg string, args ...any)  {}
func (m *MockLogger) Error(msg string, args ...any) {}

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
