package util

import (
	"io"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
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
		ID:          "Mock Plugin",
		Version:     "0.0.1",
		Description: "A mock plugin.",
	}
}

func (m *mockPlugin) Init(ctx *models.PluginContext) error {
	return nil
}

func (m *mockPlugin) Migrations() []any {
	return []any{}
}

func (m *mockPlugin) Routes() []models.Route {
	return []models.Route{}
}

func (m *mockPlugin) Close() error {
	return nil
}

// ------------------------------------
