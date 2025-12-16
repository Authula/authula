package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/auth/storage"
	"github.com/GoBetterAuth/go-better-auth/internal/plugins"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// ------------------------------------

type mockPlugin struct{}

func (m *mockPlugin) Metadata() domain.PluginMetadata {
	return domain.PluginMetadata{
		Name:        "Mock Plugin",
		Version:     "0.0.1",
		Description: "A mock plugin.",
	}
}

func (m *mockPlugin) Config() domain.PluginConfig {
	return domain.PluginConfig{Enabled: true}
}

func (m *mockPlugin) Ctx() *domain.PluginContext {
	return &domain.PluginContext{Config: nil, EventBus: nil, Middleware: nil}
}

func (m *mockPlugin) Init(ctx *domain.PluginContext) error {
	return nil
}

func (m *mockPlugin) Migrations() []any {
	return []any{}
}

func (m *mockPlugin) Routes() []domain.PluginRoute {
	return []domain.PluginRoute{}
}

func (m *mockPlugin) RateLimit() *domain.PluginRateLimit {
	return &domain.PluginRateLimit{
		CustomRules: map[string]domain.RateLimitCustomRuleFunc{
			"/plugin": func(req *http.Request) domain.RateLimitCustomRule {
				return domain.RateLimitCustomRule{
					Window: 1 * time.Minute,
					Max:    1,
				}
			},
		},
	}
}

func (m *mockPlugin) DatabaseHooks() *domain.PluginDatabaseHooks {
	return nil
}

func (m *mockPlugin) EventHooks() *domain.PluginEventHooks {
	return nil
}

func (m *mockPlugin) Close() error {
	return nil
}

// ------------------------------------

// createMockRequest creates a basic mock HTTP request for testing
func createMockRequest() *http.Request {
	req, _ := http.NewRequest("POST", "/test", nil)
	return req
}

func TestRateLimitService_Allow(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		key       string
		max       int
		window    time.Duration
		requests  int
		wantAllow []bool
		wantErr   bool
	}{
		{
			name:      "rate limiting disabled",
			enabled:   false,
			key:       "test-key",
			max:       2,
			window:    1 * time.Minute,
			requests:  5,
			wantAllow: []bool{true, true, true, true, true},
			wantErr:   false,
		},
		{
			name:      "rate limiting enabled - allow under limit",
			enabled:   true,
			key:       "test-key",
			max:       3,
			window:    1 * time.Minute,
			requests:  2,
			wantAllow: []bool{true, true},
			wantErr:   false,
		},
		{
			name:      "rate limiting enabled - exceed limit",
			enabled:   true,
			key:       "test-key",
			max:       2,
			window:    1 * time.Minute,
			requests:  4,
			wantAllow: []bool{true, true, false, false},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := domain.NewConfig(
				domain.WithSecondaryStorage(
					domain.SecondaryStorageConfig{
						Storage: storage.NewMemorySecondaryStorage(nil),
					},
				),
				domain.WithRateLimit(
					domain.RateLimitConfig{
						Enabled:     tt.enabled,
						Window:      tt.window,
						Max:         tt.max,
						Algorithm:   domain.RateLimitAlgorithmFixedWindow,
						Prefix:      "test:",
						CustomRules: map[string]domain.RateLimitCustomRuleFunc{},
						IP: domain.IPConfig{
							Headers: []string{"X-Forwarded-For", "X-Real-IP"},
						},
					},
				),
			)

			service := NewRateLimitService(config, plugins.NewPluginRegistry(config, nil, nil))
			ctx := context.Background()
			req := createMockRequest()

			for i := 0; i < tt.requests; i++ {
				allowed, err := service.Allow(ctx, tt.key, req)

				if (err != nil) != tt.wantErr {
					t.Errorf("Allow() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if allowed != tt.wantAllow[i] {
					t.Errorf("Allow() request %d = %v, want %v", i+1, allowed, tt.wantAllow[i])
				}
			}
		})
	}
}

func TestRateLimitService_CustomRule(t *testing.T) {
	config := &domain.Config{
		RateLimit: domain.RateLimitConfig{
			Enabled:   true,
			Window:    1 * time.Minute,
			Max:       10,
			Algorithm: domain.RateLimitAlgorithmFixedWindow,
			Prefix:    "test:",
			CustomRules: map[string]domain.RateLimitCustomRuleFunc{
				"/strict": func(req *http.Request) domain.RateLimitCustomRule {
					return domain.RateLimitCustomRule{
						Window: 1 * time.Minute,
						Max:    2,
					}
				},
				"/disabled": func(req *http.Request) domain.RateLimitCustomRule {
					return domain.RateLimitCustomRule{
						Disabled: true,
					}
				},
			},
			IP: domain.IPConfig{
				Headers: []string{"X-Forwarded-For", "X-Real-IP"},
			},
		},
		SecondaryStorage: domain.SecondaryStorageConfig{
			Storage: storage.NewMemorySecondaryStorage(nil),
		},
	}

	service := NewRateLimitService(config, plugins.NewPluginRegistry(config, nil, nil))
	ctx := context.Background()

	// Test strict custom rule
	reqStrict, _ := http.NewRequest("GET", "/strict", nil)
	allowed1, _ := service.Allow(ctx, "strict-key", reqStrict)
	allowed2, _ := service.Allow(ctx, "strict-key", reqStrict)
	allowed3, _ := service.Allow(ctx, "strict-key", reqStrict)

	if !allowed1 || !allowed2 || allowed3 {
		t.Errorf("Custom rule not working: %v, %v, %v (expected true, true, false)", allowed1, allowed2, allowed3)
	}

	// Test disabled rule (infinite)
	reqDisabled, _ := http.NewRequest("GET", "/disabled", nil)
	for i := range 100 {
		allowed, err := service.Allow(ctx, "disabled-key", reqDisabled)
		if err != nil {
			t.Errorf("Disabled rule returned error: %v", err)
		}
		if !allowed {
			t.Errorf("Disabled rule blocked at request %d", i+1)
		}
	}
}

func TestRateLimitService_ClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		ipHeaders  []string
		expected   string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
			},
			remoteAddr: "127.0.0.1:8080",
			ipHeaders:  []string{"X-Forwarded-For", "X-Real-IP"},
			expected:   "203.0.113.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.1",
			},
			remoteAddr: "127.0.0.1:8080",
			ipHeaders:  []string{"X-Forwarded-For", "X-Real-IP"},
			expected:   "203.0.113.1",
		},
		{
			name: "X-Forwarded-For with multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 198.51.100.1",
			},
			remoteAddr: "127.0.0.1:8080",
			ipHeaders:  []string{"X-Forwarded-For", "X-Real-IP"},
			expected:   "203.0.113.1",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "203.0.113.1:8080",
			ipHeaders:  []string{"X-Forwarded-For", "X-Real-IP"},
			expected:   "203.0.113.1",
		},
		{
			name:       "RemoteAddr fallback without port",
			headers:    map[string]string{},
			remoteAddr: "203.0.113.1",
			ipHeaders:  []string{"X-Forwarded-For", "X-Real-IP"},
			expected:   "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &domain.Config{
				RateLimit: domain.RateLimitConfig{
					Enabled:     true,
					Window:      1 * time.Minute,
					Max:         10,
					Algorithm:   domain.RateLimitAlgorithmFixedWindow,
					Prefix:      "test:",
					CustomRules: map[string]domain.RateLimitCustomRuleFunc{},
					IP: domain.IPConfig{
						Headers: tt.ipHeaders,
					},
				},
				SecondaryStorage: domain.SecondaryStorageConfig{
					Storage: storage.NewMemorySecondaryStorage(nil),
				},
			}

			service := NewRateLimitService(config, plugins.NewPluginRegistry(config, nil, nil))

			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := service.GetClientIP(req)
			if ip != tt.expected {
				t.Errorf("GetClientIP() = %v, want %v", ip, tt.expected)
			}
		})
	}
}

func TestRateLimitService_PluginRule(t *testing.T) {
	config := &domain.Config{
		RateLimit: domain.RateLimitConfig{
			Enabled:     true,
			Window:      1 * time.Minute,
			Max:         100,
			Algorithm:   domain.RateLimitAlgorithmFixedWindow,
			Prefix:      "test:",
			CustomRules: map[string]domain.RateLimitCustomRuleFunc{},
			IP: domain.IPConfig{
				Headers: []string{"X-Forwarded-For", "X-Real-IP"},
			},
		},
		SecondaryStorage: domain.SecondaryStorageConfig{
			Storage: storage.NewMemorySecondaryStorage(nil),
		},
	}

	registry := plugins.NewPluginRegistry(config, nil, nil)

	registry.Register(&mockPlugin{})

	service := NewRateLimitService(config, registry)
	ctx := context.Background()

	req, _ := http.NewRequest("GET", "/plugin", nil)

	allowed1, err1 := service.Allow(ctx, "plugin-key", req)
	if err1 != nil {
		t.Fatalf("unexpected error on first allow: %v", err1)
	}
	if !allowed1 {
		t.Fatalf("expected first request to be allowed")
	}

	allowed2, err2 := service.Allow(ctx, "plugin-key", req)
	if err2 != nil {
		t.Fatalf("unexpected error on second allow: %v", err2)
	}
	if allowed2 {
		t.Fatalf("expected second request to be blocked by plugin rate limit")
	}
}
