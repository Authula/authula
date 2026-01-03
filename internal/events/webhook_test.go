package events

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GoBetterAuth/go-better-auth/models"
)

func TestNewWebhookExecutor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewWebhookExecutor(logger)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.logger)
	assert.NotNil(t, executor.client)
}

func TestExecuteWebhook_NilWebhook(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewWebhookExecutor(logger)

	err := executor.ExecuteWebhook(nil, map[string]string{"test": "data"})
	assert.NoError(t, err)
}

func TestExecuteWebhook_EmptyURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewWebhookExecutor(logger)

	webhook := &models.WebhookConfig{URL: ""}
	err := executor.ExecuteWebhook(webhook, map[string]string{"test": "data"})
	assert.NoError(t, err)
}

func TestExecuteWebhook_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-value", r.Header.Get("X-Test-Header"))

		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "data", payload["test"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewWebhookExecutor(logger)

	webhook := &models.WebhookConfig{
		URL: server.URL,
		Headers: map[string]string{
			"X-Test-Header": "test-value",
		},
		TimeoutSeconds: 5 * time.Second,
	}

	err := executor.ExecuteWebhook(webhook, map[string]string{"test": "data"})
	assert.NoError(t, err)
}

func TestExecuteWebhook_NonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewWebhookExecutor(logger)

	webhook := &models.WebhookConfig{URL: server.URL}
	err := executor.ExecuteWebhook(webhook, map[string]string{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code 400")
}

func TestExecuteWebhook_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewWebhookExecutor(logger)

	webhook := &models.WebhookConfig{
		URL:            server.URL,
		TimeoutSeconds: 1 * time.Second,
	}

	err := executor.ExecuteWebhook(webhook, map[string]string{"test": "data"})
	assert.Error(t, err)
}
