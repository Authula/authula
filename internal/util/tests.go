package util

// Provides utility types and functions for tests

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
