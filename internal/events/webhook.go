package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// WebhookExecutor handles HTTP webhook execution
type WebhookExecutor struct {
	logger models.Logger
	client *http.Client
}

// NewWebhookExecutor creates a new webhook executor
func NewWebhookExecutor(logger models.Logger) *WebhookExecutor {
	return &WebhookExecutor{
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteWebhook sends a webhook request with the given payload
func (w *WebhookExecutor) ExecuteWebhook(webhook *models.WebhookConfig, payload any) error {
	if webhook == nil || webhook.URL == "" {
		return nil
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	ctx := context.Background()
	if webhook.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, webhook.TimeoutSeconds)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	w.logger.Debug("Executing webhook", "url", webhook.URL)
	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Error("Webhook request failed", "url", webhook.URL, "error", err)
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.logger.Warn("Webhook returned non-success status", "url", webhook.URL, "status", resp.StatusCode)
		return fmt.Errorf("webhook returned status code %d", resp.StatusCode)
	}

	w.logger.Debug("Webhook executed successfully", "url", webhook.URL, "status", resp.StatusCode)
	return nil
}
