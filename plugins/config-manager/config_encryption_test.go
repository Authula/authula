package configmanager

import (
	"encoding/json"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/repositories"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/services"
)

func TestConfigEncryption(t *testing.T) {
	// Create a test config with sensitive values
	testConfig := &models.Config{
		AppName:  "TestApp",
		BaseURL:  "http://localhost:8080",
		BasePath: "/api",
		Secret:   "test-secret",
		Database: models.DatabaseConfig{
			Provider: "postgres",
			URL:      "postgresql://user:password@localhost:5432/testdb",
		},
		Logger: models.LoggerConfig{
			Level: "info",
		},
	}

	// Create token service and encryptor
	cryptoTokenRepo := repositories.NewCryptoTokenRepository(testConfig.Secret)
	tokenService := services.NewTokenService(cryptoTokenRepo)
	encryptor := NewConfigEncryptor(testConfig.Secret, tokenService)

	// Test encryption
	encrypted, err := encryptor.EncryptConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to encrypt config: %v", err)
	}

	// Verify that the encrypted data is not empty
	if len(encrypted) == 0 {
		t.Fatal("Encrypted config is empty")
	}

	// Test decryption
	decrypted, err := encryptor.DecryptConfig(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt config: %v", err)
	}

	// Verify that sensitive fields are decrypted correctly
	if decrypted.Secret != testConfig.Secret {
		t.Errorf("Secret mismatch: expected %q, got %q", testConfig.Secret, decrypted.Secret)
	}

	if decrypted.Database.URL != testConfig.Database.URL {
		t.Errorf("Database URL mismatch: expected %q, got %q", testConfig.Database.URL, decrypted.Database.URL)
	}

	// Verify non-sensitive fields remain the same
	if decrypted.AppName != testConfig.AppName {
		t.Errorf("AppName mismatch: expected %q, got %q", testConfig.AppName, decrypted.AppName)
	}

	if decrypted.BaseURL != testConfig.BaseURL {
		t.Errorf("BaseURL mismatch: expected %q, got %q", testConfig.BaseURL, decrypted.BaseURL)
	}
}

func TestConfigEncryptionWithEmptyValues(t *testing.T) {
	// Create a test config with some empty sensitive values
	testConfig := &models.Config{
		AppName:  "TestApp",
		BaseURL:  "http://localhost:8080",
		BasePath: "/api",
		Secret:   "test-secret",
		Database: models.DatabaseConfig{
			Provider: "postgres",
			URL:      "", // Empty URL
		},
		Logger: models.LoggerConfig{
			Level: "info",
		},
	}

	cryptoTokenRepo := repositories.NewCryptoTokenRepository(testConfig.Secret)
	tokenService := services.NewTokenService(cryptoTokenRepo)
	encryptor := NewConfigEncryptor(testConfig.Secret, tokenService)

	// Encryption should handle empty values gracefully
	encrypted, err := encryptor.EncryptConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to encrypt config with empty values: %v", err)
	}

	// Decryption should also handle empty values
	decrypted, err := encryptor.DecryptConfig(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt config with empty values: %v", err)
	}

	// Empty values should remain empty
	if decrypted.Database.URL != "" {
		t.Errorf("Database URL should be empty, got %q", decrypted.Database.URL)
	}
}

func TestConfigEncryptionWithEventBusConfig(t *testing.T) {
	// Create a test config with event bus configurations
	testConfig := &models.Config{
		AppName: "TestApp",
		Secret:  "test-secret",
		EventBus: models.EventBusConfig{
			Provider: "redis",
			Redis: &models.RedisConfig{
				URL:           "redis://localhost:6379",
				ConsumerGroup: "mygroup",
			},
			PostgreSQL: &models.PostgreSQLConfig{
				URL: "postgresql://user:password@localhost:5432/testdb",
			},
		},
		Logger: models.LoggerConfig{
			Level: "info",
		},
	}

	cryptoTokenRepo := repositories.NewCryptoTokenRepository(testConfig.Secret)
	tokenService := services.NewTokenService(cryptoTokenRepo)
	encryptor := NewConfigEncryptor(testConfig.Secret, tokenService)

	// Test encryption
	encrypted, err := encryptor.EncryptConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to encrypt config: %v", err)
	}

	// Test decryption
	decrypted, err := encryptor.DecryptConfig(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt config: %v", err)
	}

	// Verify event bus URLs are decrypted correctly
	if decrypted.EventBus.Redis.URL != testConfig.EventBus.Redis.URL {
		t.Errorf("Redis URL mismatch: expected %q, got %q", testConfig.EventBus.Redis.URL, decrypted.EventBus.Redis.URL)
	}

	if decrypted.EventBus.PostgreSQL.URL != testConfig.EventBus.PostgreSQL.URL {
		t.Errorf("PostgreSQL URL mismatch: expected %q, got %q", testConfig.EventBus.PostgreSQL.URL, decrypted.EventBus.PostgreSQL.URL)
	}
}

func TestEncryptionIsNotPlaintext(t *testing.T) {
	testConfig := &models.Config{
		AppName: "TestApp",
		BaseURL: "http://localhost:8080",
		Secret:  "test-secret",
		Database: models.DatabaseConfig{
			URL: "postgresql://user:password@localhost:5432/testdb",
		},
	}

	cryptoTokenRepo := repositories.NewCryptoTokenRepository(testConfig.Secret)
	tokenService := services.NewTokenService(cryptoTokenRepo)
	encryptor := NewConfigEncryptor(testConfig.Secret, tokenService)

	encrypted, err := encryptor.EncryptConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to encrypt config: %v", err)
	}

	// The encrypted data should not contain the plaintext secret or database URL
	encryptedStr := string(encrypted)
	if contains(encryptedStr, testConfig.Secret) {
		t.Error("Encrypted config contains plaintext secret")
	}

	if contains(encryptedStr, "password@localhost") {
		t.Error("Encrypted config contains plaintext database URL")
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPlaintextMigrationDetection(t *testing.T) {
	// Test that we can detect plaintext JSON and treat it as such
	testConfig := &models.Config{
		AppName: "TestApp",
		Secret:  "test-secret",
		Database: models.DatabaseConfig{
			URL: "postgresql://user:password@localhost:5432/testdb",
		},
		Logger: models.LoggerConfig{
			Level: "info",
		},
	}

	cryptoTokenRepo := repositories.NewCryptoTokenRepository(testConfig.Secret)
	tokenService := services.NewTokenService(cryptoTokenRepo)
	encryptor := NewConfigEncryptor(testConfig.Secret, tokenService)

	// Create plaintext JSON (simulating old config before encryption)
	plaintextJSON, _ := json.Marshal(testConfig)

	// Try to decrypt plaintext - should fail
	_, err := encryptor.DecryptConfig(plaintextJSON)
	if err == nil {
		t.Error("Expected decryption of plaintext to fail, but it succeeded")
	}

	// This test verifies the migration logic will work correctly
	// The Load() method will catch this error and treat it as plaintext
}
