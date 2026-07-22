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

type MockPluginRegistry struct{}

func (r *MockPluginRegistry) Register(p models.Plugin) error           { return nil }
func (r *MockPluginRegistry) InitAll() error                           { return nil }
func (r *MockPluginRegistry) RunMigrations(ctx context.Context) error  { return nil }
func (r *MockPluginRegistry) DropMigrations(ctx context.Context) error { return nil }
func (r *MockPluginRegistry) Plugins() []models.Plugin                 { return nil }
func (r *MockPluginRegistry) GetConfig() *models.Config                { return nil }
func (r *MockPluginRegistry) CloseAll()                                {}
func (r *MockPluginRegistry) GetPlugin(pluginID string) models.Plugin  { return nil }
