package configmanager

import (
	"encoding/json"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/repositories"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/services"
)

func TestDebugEncryption() {
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
	}

	cryptoTokenRepo := repositories.NewCryptoTokenRepository(testConfig.Secret)
	tokenService := services.NewTokenService(cryptoTokenRepo)
	encryptor := NewConfigEncryptor(testConfig.Secret, tokenService)

	// Test encryption
	encrypted, err := encryptor.EncryptConfig(testConfig)
	if err != nil {
		fmt.Printf("Encryption failed: %v\n", err)
		return
	}

	fmt.Println("=== ENCRYPTED JSON (first 500 chars) ===")
	fmt.Println(string(encrypted[:500]))

	// Check what's unmarshaled
	var encryptedConfig models.Config
	json.Unmarshal(encrypted, &encryptedConfig)
	fmt.Printf("\nRedis URL in encrypted config: %s\n", encryptedConfig.EventBus.Redis.URL)
	fmt.Printf("PostgreSQL URL in encrypted config: %s\n", encryptedConfig.EventBus.PostgreSQL.URL)
}
