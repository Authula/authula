package util

import (
	"io"
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// Provides utility types and functions for tests

// ------------------------------------

// createMockRequest creates a basic mock HTTP request for testing
func CreateMockRequest(method string, path string, query map[string]string, body io.Reader, headers map[string]string) *http.Request {
	req, _ := http.NewRequest(method, path, body)
	if query != nil {
		q := req.URL.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

// ------------------------------------

type MockLogger struct {
}

func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (m *MockLogger) Debug(msg string, args ...any) {
	// Mock implementation - no-op
}

func (m *MockLogger) Info(msg string, args ...any) {
	// Mock implementation - no-op
}

func (m *MockLogger) Warn(msg string, args ...any) {
	// Mock implementation - no-op
}

func (m *MockLogger) Error(msg string, args ...any) {
	// Mock implementation - no-op
}

// ------------------------------------

type mockPlugin struct{}

func NewMockPlugin() *mockPlugin {
	return &mockPlugin{}
}

func (m *mockPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		Name:        "Mock Plugin",
		Version:     "0.0.1",
		Description: "A mock plugin.",
	}
}

func (m *mockPlugin) Config() models.PluginConfig {
	return models.PluginConfig{Enabled: true}
}

func (m *mockPlugin) Ctx() *models.PluginContext {
	return &models.PluginContext{Config: nil, EventBus: nil, Middleware: nil}
}

func (m *mockPlugin) Init(ctx *models.PluginContext) error {
	return nil
}

func (m *mockPlugin) Migrations() []any {
	return []any{}
}

func (m *mockPlugin) Routes() []models.PluginRoute {
	return []models.PluginRoute{}
}

func (m *mockPlugin) RateLimit() *models.PluginRateLimit {
	return &models.PluginRateLimit{
		Enabled: true,
		CustomRules: map[string]models.RateLimitCustomRule{
			"/plugin": {
				Window: 1 * time.Minute,
				Max:    1,
			},
		},
	}
}

func (m *mockPlugin) DatabaseHooks() any {
	return nil
}

func (m *mockPlugin) EventHooks() any {
	return nil
}

func (m *mockPlugin) Close() error {
	return nil
}

// ------------------------------------
