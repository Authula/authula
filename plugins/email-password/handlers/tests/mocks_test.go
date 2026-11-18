package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Authula/authula/models"
)

var _ models.Logger = (*MockLogger)(nil)

func TestMockLogger_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
		args []any
	}{
		{
			name: "forwards the message without args",
			msg:  "background task failed",
		},
		{
			name: "forwards the message and key-value args",
			msg:  "background task failed",
			args: []any{"error", "boom"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := new(MockLogger)
			logger.On("Error", tt.msg, tt.args).Once()

			logger.Error(tt.msg, tt.args...)

			logger.AssertCalled(t, "Error", tt.msg, tt.args)
			logger.AssertExpectations(t)
		})
	}
}

func TestMockLogger_NoopLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(logger *MockLogger)
	}{
		{
			name: "Debug does not record a call",
			call: func(logger *MockLogger) {
				logger.Debug("debug message", "key", "value")
			},
		},
		{
			name: "Info does not record a call",
			call: func(logger *MockLogger) {
				logger.Info("info message", "key", "value")
			},
		},
		{
			name: "Warn does not record a call",
			call: func(logger *MockLogger) {
				logger.Warn("warn message", "key", "value")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := new(MockLogger)

			tt.call(logger)

			assert.Empty(t, logger.Calls, "noop levels should not reach the mock recorder")
			logger.AssertExpectations(t)
		})
	}
}
