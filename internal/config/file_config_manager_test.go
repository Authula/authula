package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// TestFileConfigManager_Watch tests the file watching functionality
// Note: This test uses real time because fsnotify is OS-based and cannot use virtual time
func TestFileConfigManager_Watch(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")

	// Set the config path env var
	os.Setenv("GO_BETTER_AUTH_CONFIG_PATH", configPath)

	// Write initial config
	initialConfig := `
mode = "file"
app_name = "TestApp"
base_url = "http://localhost:8080"
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create initial config struct
	initCfg := &models.Config{}

	// Create FileConfigManager
	cm := NewFileConfigManager(initCfg)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configChan, err := cm.Watch(ctx)
	if err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

	// Update the config file after a short delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		updatedConfig := `
mode = "file"
app_name = "UpdatedApp"
base_url = "http://localhost:9090"
`
		if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
			t.Errorf("Failed to update config: %v", err)
		}
	}()

	// Wait for the update notification
	select {
	case config := <-configChan:
		if config == nil {
			t.Error("Received nil config")
		}
		if config.AppName != "UpdatedApp" {
			t.Errorf("Expected AppName to be 'UpdatedApp', got '%s'", config.AppName)
		}
		if config.BaseURL != "http://localhost:9090" {
			t.Errorf("Expected BaseURL to be 'http://localhost:9090', got '%s'", config.BaseURL)
		}

	case <-ctx.Done():
		t.Error("Timeout waiting for config update")
	}
}

// TestFileConfigManager_Watch_InvalidConfig tests watching with invalid config
// Note: This test uses real time because fsnotify is OS-based and cannot use virtual time
func TestFileConfigManager_Watch_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")

	// Set the config path env var
	os.Setenv("GO_BETTER_AUTH_CONFIG_PATH", configPath)

	// Write initial valid config
	initialConfig := `
mode = "file"
app_name = "TestApp"
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create initial config struct
	initCfg := &models.Config{}

	cm := NewFileConfigManager(initCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configChan, err := cm.Watch(ctx)
	if err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

	// Write invalid TOML
	go func() {
		time.Sleep(200 * time.Millisecond)
		if err := os.WriteFile(configPath, []byte("invalid toml [[["), 0644); err != nil {
			t.Errorf("Failed to write invalid config: %v", err)
		}
	}()

	// The watcher should skip invalid configs and not send anything
	select {
	case config := <-configChan:
		if config != nil {
			t.Error("Should not receive config for invalid TOML")
		}
	case <-time.After(2 * time.Second):
		// Expected - invalid config should be skipped
	case <-ctx.Done():
		// Also acceptable - timeout waiting for valid update
	}
}

// TestFileConfigManager_Update tests that Update persists changes to the config file
func TestFileConfigManager_Update(t *testing.T) {
	util.InitValidator()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")

	// Set the config path env var
	os.Setenv("GO_BETTER_AUTH_CONFIG_PATH", configPath)

	initialConfig := `
mode = "file"
app_name = "OriginalName"
base_url = "http://localhost:8080"
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create initial config struct
	initCfg := &models.Config{}

	cm := NewFileConfigManager(initCfg)

	// Update a top-level field
	err := cm.Update("app_name", "UpdatedName")
	if err != nil {
		t.Fatalf("Expected Update to succeed, got error: %v", err)
	}

	// Verify the change is in memory
	config := cm.GetConfig()
	if config.AppName != "UpdatedName" {
		t.Errorf("Expected AppName to be 'UpdatedName', got '%s'", config.AppName)
	}

	// Verify the change persisted to file
	content, _ := os.ReadFile(configPath)
	if !strings.Contains(string(content), "UpdatedName") {
		t.Error("Updated value not found in config file")
	}
}

// TestFileConfigManager_Update_NestedField tests updating nested configuration fields
func TestFileConfigManager_Update_NestedField(t *testing.T) {
	util.InitValidator()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")

	// Set the config path env var
	os.Setenv("GO_BETTER_AUTH_CONFIG_PATH", configPath)

	initialConfig := `
mode = "file"
app_name = "TestApp"

[email]
provider = "smtp"
smtp_host = "smtp.example.com"
smtp_port = 587
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create initial config struct
	initCfg := &models.Config{}

	cm := NewFileConfigManager(initCfg)

	// Update a nested field
	err := cm.Update("email.smtp_host", "new.smtp.host.com")
	if err != nil {
		t.Fatalf("Expected Update to succeed, got error: %v", err)
	}

	// Verify the change is in memory
	config := cm.GetConfig()
	if config.Email.SMTPHost != "new.smtp.host.com" {
		t.Errorf("Expected Email.SMTPHost to be 'new.smtp.host.com', got '%s'", config.Email.SMTPHost)
	}

	// Verify the change persisted to file
	content, _ := os.ReadFile(configPath)
	if !strings.Contains(string(content), "new.smtp.host.com") {
		t.Error("Updated nested value not found in config file")
	}
}
