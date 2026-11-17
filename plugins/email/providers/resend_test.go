package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/resend/resend-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/email/constants"
	emailtypes "github.com/Authula/authula/plugins/email/types"
)

func TestNewResendProvider(t *testing.T) {
	// t.Setenv forbids t.Parallel in this test.
	tests := []struct {
		name        string
		envAPIKey   string
		config      *emailtypes.EmailPluginConfig
		wantAPIKey  string
		wantErr     bool
		wantLogCall bool
	}{
		{
			name:       "uses API key from environment",
			envAPIKey:  "re_env_key",
			config:     new(emailtypes.EmailPluginConfig),
			wantAPIKey: "re_env_key",
		},
		{
			name:       "trims whitespace from environment API key",
			envAPIKey:  "  re_env_key  ",
			config:     new(emailtypes.EmailPluginConfig),
			wantAPIKey: "re_env_key",
		},
		{
			name:       "falls back to config API key",
			config:     &emailtypes.EmailPluginConfig{Resend: &emailtypes.ResendConfig{ApiKey: "re_config_key"}},
			wantAPIKey: "re_config_key",
		},
		{
			name:       "environment API key takes precedence over config",
			envAPIKey:  "re_env_key",
			config:     &emailtypes.EmailPluginConfig{Resend: &emailtypes.ResendConfig{ApiKey: "re_config_key"}},
			wantAPIKey: "re_env_key",
		},
		{
			name:        "errors when neither environment nor config provide a key",
			config:      new(emailtypes.EmailPluginConfig),
			wantErr:     true,
			wantLogCall: true,
		},
		{
			name:        "errors when config has an empty API key",
			config:      &emailtypes.EmailPluginConfig{Resend: &emailtypes.ResendConfig{ApiKey: ""}},
			wantErr:     true,
			wantLogCall: true,
		},
		{
			name:        "ignores blank environment key and empty config key",
			envAPIKey:   "   ",
			config:      &emailtypes.EmailPluginConfig{Resend: &emailtypes.ResendConfig{}},
			wantErr:     true,
			wantLogCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(constants.EnvResendApiKey, tt.envAPIKey)
			logger := new(internaltests.RecordingLogger)

			provider, err := NewResendProvider(tt.config, logger)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)
				assert.Contains(t, err.Error(), constants.EnvResendApiKey)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				require.NotNil(t, provider.client)
				assert.Equal(t, tt.wantAPIKey, provider.client.ApiKey)
				assert.Equal(t, tt.config, provider.config)
				assert.Equal(t, logger, provider.logger)
			}

			if tt.wantLogCall {
				assert.Len(t, logger.ErrorCalls, 1)
				assert.Contains(t, logger.ErrorCalls[0], "Resend API key is not set")
			} else {
				assert.Empty(t, logger.ErrorCalls)
			}
		})
	}
}

func newResendTestProvider(t *testing.T, serverURL string, logger *internaltests.RecordingLogger) *ResendProvider {
	t.Helper()

	baseURL, err := url.Parse(serverURL + "/")
	require.NoError(t, err)

	client := resend.NewClient("re_test_key")
	client.BaseURL = baseURL

	return &ResendProvider{
		config: &emailtypes.EmailPluginConfig{FromAddress: "noreply@example.com"},
		logger: logger,
		client: client,
	}
}

func TestResendProvider_SendEmail_RequestsEmailFromAPI(t *testing.T) {
	t.Parallel()

	var (
		gotMethod        string
		gotPath          string
		gotAuthorization string
		gotBody          map[string]any
		decodeErr        error
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")

		decodeErr = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "email-123"})
	}))
	t.Cleanup(server.Close)

	provider := newResendTestProvider(t, server.URL, new(internaltests.RecordingLogger))

	err := provider.SendEmail(context.Background(), "user@example.com", "Welcome", "plain body", "")

	require.NoError(t, err)
	require.NoError(t, decodeErr)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/emails", gotPath)
	assert.Equal(t, "Bearer re_test_key", gotAuthorization)
	assert.Equal(t, map[string]any{
		"from":    "noreply@example.com",
		"to":      []any{"user@example.com"},
		"subject": "Welcome",
		"text":    "plain body",
	}, gotBody)
}

func TestResendProvider_SendEmail_SendsHTMLBody(t *testing.T) {
	t.Parallel()

	var (
		gotBody   map[string]any
		decodeErr error
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeErr = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "email-456"})
	}))
	t.Cleanup(server.Close)

	provider := newResendTestProvider(t, server.URL, new(internaltests.RecordingLogger))

	err := provider.SendEmail(context.Background(), "user@example.com", "Welcome", "", "<p>hi</p>")

	require.NoError(t, err)
	require.NoError(t, decodeErr)
	assert.Equal(t, "<p>hi</p>", gotBody["html"])
	assert.NotContains(t, gotBody, "text")
}

func TestResendProvider_SendEmail_ValidatesBodyBeforeRequest(t *testing.T) {
	t.Parallel()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	provider := newResendTestProvider(t, server.URL, new(internaltests.RecordingLogger))

	err := provider.SendEmail(context.Background(), "user@example.com", "Welcome", "", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least a text or html body")
	assert.Zero(t, requests, "no HTTP request should be made for an empty body")
}

func TestResendProvider_SendEmail_ReturnsAPIError(t *testing.T) {
	t.Parallel()

	logger := new(internaltests.RecordingLogger)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid from address"}`))
	}))
	t.Cleanup(server.Close)

	provider := newResendTestProvider(t, server.URL, logger)

	err := provider.SendEmail(context.Background(), "user@example.com", "Welcome", "plain body", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resend send failed")
	assert.Contains(t, err.Error(), "invalid from address")
	require.Len(t, logger.ErrorCalls, 1)
	assert.Contains(t, logger.ErrorCalls[0], "failed to send email via Resend")
}

func TestResendProvider_SendEmail_RejectsEmptyResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "missing id in json response", statusCode: http.StatusOK, body: `{}`},
		{name: "no content response", statusCode: http.StatusNoContent, body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			}))
			t.Cleanup(server.Close)

			provider := newResendTestProvider(t, server.URL, new(internaltests.RecordingLogger))

			err := provider.SendEmail(context.Background(), "user@example.com", "Welcome", "plain body", "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "empty response")
		})
	}
}

func TestResendProvider_SendEmail_PropagatesTransportError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	provider := newResendTestProvider(t, serverURL, new(internaltests.RecordingLogger))

	err := provider.SendEmail(context.Background(), "user@example.com", "Welcome", "plain body", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resend send failed")
}
