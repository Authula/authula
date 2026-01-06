package configmanager_test

import (
	"errors"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// MockConfigWatcher is a test implementation of PluginWithConfigWatcher
type MockConfigWatcher struct {
	lastConfig  *models.Config
	callCount   int
	lastError   error
	shouldError bool
}

func (m *MockConfigWatcher) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          "test_watcher",
		Version:     "1.0.0",
		Description: "Test watcher for config updates",
	}
}

func (m *MockConfigWatcher) Config() any {
	return map[string]any{"enabled": true}
}

func (m *MockConfigWatcher) Init(ctx *models.PluginContext) error {
	return nil
}

func (m *MockConfigWatcher) Close() error {
	return nil
}

func (m *MockConfigWatcher) OnConfigUpdate(cfg *models.Config) error {
	m.callCount++
	m.lastConfig = cfg
	if m.shouldError {
		m.lastError = errors.New("test error")
		return m.lastError
	}
	return nil
}

// TestConfigWatcherRegistration tests that plugins can be registered as config watchers
// NOTE: This test is currently disabled as it relies on PluginWithAPI which has been removed
// TODO: Re-implement once config watcher registration is restored via service registry
func TestConfigWatcherRegistration(t *testing.T) {
	t.Skip("Config watcher registration is currently disabled")
}

// TestConfigWatcherNotification tests that registered watchers are notified of config changes
// NOTE: This test is currently disabled as it relies on PluginWithAPI which has been removed
// TODO: Re-implement once config watcher registration is restored via service registry
func TestConfigWatcherNotification(t *testing.T) {
	t.Skip("Config watcher notification is currently disabled")
}

// TestConfigWatcherErrorHandling tests that an error in one watcher doesn't block others (fail-open)
// NOTE: This test is currently disabled as it relies on PluginWithAPI which has been removed
// TODO: Re-implement once config watcher registration is restored via service registry
func TestConfigWatcherErrorHandling(t *testing.T) {
	t.Skip("Config watcher error handling is currently disabled")
}

// Helper functions

type testLogger struct{}

func (l *testLogger) Debug(msg string, args ...any) {}
func (l *testLogger) Info(msg string, args ...any)  {}
func (l *testLogger) Warn(msg string, args ...any)  {}
func (l *testLogger) Error(msg string, args ...any) {}
