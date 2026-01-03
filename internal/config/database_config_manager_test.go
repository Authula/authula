package config

import (
	"encoding/json"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// TestDatabaseConfigManager_KeyMapping verifies that all Config struct fields
// are properly mapped to/from JSON when persisting to the database.
func TestDatabaseConfigManager_KeyMapping(t *testing.T) {
	testConfig := &models.Config{
		Mode:     models.ModeStandalone,
		AppName:  "Test App",
		BaseURL:  "http://localhost:8080",
		BasePath: "/auth",
		Secret:   "test-secret",
		Database: models.DatabaseConfig{
			Provider:     "postgres",
			URL:          "postgresql://user:pass@localhost/db",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
		Email: models.EmailConfig{
			Provider: "smtp",
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			SMTPUser: "user@example.com",
			SMTPPass: "password",
			From:     "noreply@example.com",
		},
		Session: models.SessionConfig{
			CookieName: "session_id",
		},
		CSRF: models.CSRFConfig{
			Enabled: true,
		},
		RateLimit: models.RateLimitConfig{
			Enabled: true,
		},
	}

	// Marshal the config to JSON
	jsonData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Unmarshal back to verify all fields are preserved
	unmarshaledConfig := &models.Config{}
	if err := json.Unmarshal(jsonData, unmarshaledConfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify critical fields
	if unmarshaledConfig.AppName != testConfig.AppName {
		t.Errorf("AppName not preserved: %s != %s", unmarshaledConfig.AppName, testConfig.AppName)
	}
	if unmarshaledConfig.BaseURL != testConfig.BaseURL {
		t.Errorf("BaseURL not preserved: %s != %s", unmarshaledConfig.BaseURL, testConfig.BaseURL)
	}
	if unmarshaledConfig.Database.Provider != testConfig.Database.Provider {
		t.Errorf("Database.Provider not preserved: %s != %s", unmarshaledConfig.Database.Provider, testConfig.Database.Provider)
	}
	if unmarshaledConfig.Email.SMTPHost != testConfig.Email.SMTPHost {
		t.Errorf("Email.SMTPHost not preserved: %s != %s", unmarshaledConfig.Email.SMTPHost, testConfig.Email.SMTPHost)
	}
	if unmarshaledConfig.Session.CookieName != testConfig.Session.CookieName {
		t.Errorf("Session.CookieName not preserved: %s != %s", unmarshaledConfig.Session.CookieName, testConfig.Session.CookieName)
	}
	if !unmarshaledConfig.CSRF.Enabled {
		t.Error("CSRF.Enabled not preserved")
	}
	if !unmarshaledConfig.RateLimit.Enabled {
		t.Error("RateLimit.Enabled not preserved")
	}
}

// TestAuthSettings_TableName verifies the table name for AuthSettings model
func TestAuthSettings_TableName(t *testing.T) {
	settings := &models.AuthSettings{}
	if settings.TableName() != "auth_settings" {
		t.Errorf("Expected table name 'auth_settings', got '%s'", settings.TableName())
	}
}

// TestConfigJSON_NestedStructures verifies that nested config structures
// are properly serialized with correct JSON tags
func TestConfigJSON_NestedStructures(t *testing.T) {
	testConfig := &models.Config{
		SocialProviders: map[string]models.OAuth2ProviderConfig{
			"google": {
				Enabled:      true,
				ClientID:     "google-id",
				ClientSecret: "google-secret",
				RedirectURL:  "http://localhost:8080/auth/callback/google",
				Scopes:       []string{"email", "profile"},
			},
			"custom": {
				Enabled:      true,
				ClientID:     "custom-id",
				ClientSecret: "custom-secret",
				AuthURL:      "https://custom.com/oauth/authorize",
				TokenURL:     "https://custom.com/oauth/token",
				UserInfoURL:  "https://custom.com/oauth/userinfo",
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Unmarshal back
	unmarshaledConfig := &models.Config{}
	if err := json.Unmarshal(jsonData, unmarshaledConfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify Google OAuth config
	googleCfg, ok := unmarshaledConfig.SocialProviders["google"]
	if !ok {
		t.Fatal("Google OAuth config not preserved")
	}
	if googleCfg.ClientID != "google-id" {
		t.Errorf("Google ClientID not preserved")
	}

	// Verify Generic OAuth config
	if customCfg, ok := unmarshaledConfig.SocialProviders["custom"]; !ok {
		t.Fatal("Custom OAuth config not preserved")
	} else {
		if customCfg.ClientID != "custom-id" {
			t.Errorf("Custom ClientID not preserved")
		}
		if customCfg.AuthURL != "https://custom.com/oauth/authorize" {
			t.Errorf("Custom AuthURL not preserved")
		}
	}
}
