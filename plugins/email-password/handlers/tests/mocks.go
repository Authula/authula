package tests

import (
	"github.com/stretchr/testify/mock"
)

// MockLogger records Error calls so tests can assert that background failures are logged.
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, args ...any) {}
func (m *MockLogger) Info(msg string, args ...any)  {}
func (m *MockLogger) Warn(msg string, args ...any)  {}

func (m *MockLogger) Error(msg string, args ...any) {
	m.Called(msg, args)
}
