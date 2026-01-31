package configmanager

import (
	"encoding/json"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

// ConfigEncryptor handles encryption and decryption of sensitive configuration values
type ConfigEncryptor struct {
	tokenService services.TokenService
}

// NewConfigEncryptor creates a new ConfigEncryptor using the app secret
func NewConfigEncryptor(secret string, tokenService services.TokenService) *ConfigEncryptor {
	return &ConfigEncryptor{
		tokenService: tokenService,
	}
}

// EncryptConfigSelectively encrypts only fields that have changed between old and new config.
// This is optimized for single-field updates and avoids re-encrypting unchanged sensitive fields.
// Returns the encrypted new config with unchanged sensitive fields copied from the old encrypted config.
func (ce *ConfigEncryptor) EncryptConfigSelectively(oldConfig *models.Config, newConfig *models.Config) ([]byte, error) {
	// First, get the old encrypted JSON to compare and preserve unchanged encrypted fields
	oldJSON, err := json.Marshal(oldConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal old config: %w", err)
	}

	var oldConfigMap map[string]any
	if err := json.Unmarshal(oldJSON, &oldConfigMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal old config map: %w", err)
	}

	// Get new config as map
	newJSON, err := json.Marshal(newConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new config: %w", err)
	}

	var newConfigMap map[string]any
	if err := json.Unmarshal(newJSON, &newConfigMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal new config map: %w", err)
	}

	// Define all sensitive fields that need encryption
	sensitiveFieldsMap := map[string][]string{
		"Secret":   {"Secret"},
		"Database": {"URL"},
		"Email":    {"SMTPPass", "SMTPUser", "SMTPHost"},
		"EventBus": {"URL"},
		"Plugins":  {}, // Special handling
	}

	// Only encrypt fields that changed
	ce.encryptChangedFields(oldConfig, newConfig, sensitiveFieldsMap)

	// Handle plugin configs - always re-encrypt since we don't deeply compare
	if err := ce.encryptPluginConfigs(newConfig); err != nil {
		return nil, fmt.Errorf("failed to encrypt plugin configs: %w", err)
	}

	// Final marshal
	finalJSON, err := json.Marshal(newConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal final encrypted config: %w", err)
	}

	return finalJSON, nil
}

// encryptChangedFields compares old and new configs and only encrypts changed sensitive fields
func (ce *ConfigEncryptor) encryptChangedFields(oldConfig *models.Config, newConfig *models.Config, sensitiveFields map[string][]string) {
	var encErr error

	// Core secrets
	if oldConfig.Secret != newConfig.Secret && newConfig.Secret != "" {
		newConfig.Secret, encErr = ce.encryptField(newConfig.Secret)
		if encErr != nil {
			// Fall back to full encryption on error
			newConfig.Secret, _ = ce.encryptField(newConfig.Secret)
		}
	} else if newConfig.Secret != "" {
		// Keep old encrypted value
		newConfig.Secret = oldConfig.Secret
	}

	// Database configuration
	if oldConfig.Database.URL != newConfig.Database.URL && newConfig.Database.URL != "" {
		newConfig.Database.URL, encErr = ce.encryptField(newConfig.Database.URL)
		if encErr == nil {
			return
		}
	} else if newConfig.Database.URL != "" {
		newConfig.Database.URL = oldConfig.Database.URL
	}

	// Event bus configurations
	if oldConfig.EventBus.PostgreSQL != nil && newConfig.EventBus.PostgreSQL != nil {
		if oldConfig.EventBus.PostgreSQL.URL != newConfig.EventBus.PostgreSQL.URL && newConfig.EventBus.PostgreSQL.URL != "" {
			newConfig.EventBus.PostgreSQL.URL, _ = ce.encryptField(newConfig.EventBus.PostgreSQL.URL)
		} else if newConfig.EventBus.PostgreSQL.URL != "" {
			newConfig.EventBus.PostgreSQL.URL = oldConfig.EventBus.PostgreSQL.URL
		}
	}

	if oldConfig.EventBus.Redis != nil && newConfig.EventBus.Redis != nil {
		if oldConfig.EventBus.Redis.URL != newConfig.EventBus.Redis.URL && newConfig.EventBus.Redis.URL != "" {
			newConfig.EventBus.Redis.URL, _ = ce.encryptField(newConfig.EventBus.Redis.URL)
		} else if newConfig.EventBus.Redis.URL != "" {
			newConfig.EventBus.Redis.URL = oldConfig.EventBus.Redis.URL
		}
	}

	if oldConfig.EventBus.NATS != nil && newConfig.EventBus.NATS != nil {
		if oldConfig.EventBus.NATS.URL != newConfig.EventBus.NATS.URL && newConfig.EventBus.NATS.URL != "" {
			newConfig.EventBus.NATS.URL, _ = ce.encryptField(newConfig.EventBus.NATS.URL)
		} else if newConfig.EventBus.NATS.URL != "" {
			newConfig.EventBus.NATS.URL = oldConfig.EventBus.NATS.URL
		}
	}

	if oldConfig.EventBus.RabbitMQ != nil && newConfig.EventBus.RabbitMQ != nil {
		if oldConfig.EventBus.RabbitMQ.URL != newConfig.EventBus.RabbitMQ.URL && newConfig.EventBus.RabbitMQ.URL != "" {
			newConfig.EventBus.RabbitMQ.URL, _ = ce.encryptField(newConfig.EventBus.RabbitMQ.URL)
		} else if newConfig.EventBus.RabbitMQ.URL != "" {
			newConfig.EventBus.RabbitMQ.URL = oldConfig.EventBus.RabbitMQ.URL
		}
	}
}

// EncryptConfig encrypts sensitive fields in the configuration
// Returns the config as JSON with encrypted values for storage
func (ce *ConfigEncryptor) EncryptConfig(config *models.Config) ([]byte, error) {
	// Create a deep copy using JSON marshal/unmarshal to avoid modifying the original
	tempJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config for copying: %w", err)
	}

	var configCopy models.Config
	if err := json.Unmarshal(tempJSON, &configCopy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config copy: %w", err)
	}

	var encErr error

	// Core secrets
	if configCopy.Secret != "" {
		configCopy.Secret, encErr = ce.encryptField(configCopy.Secret)
		if encErr != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", encErr)
		}
	}

	// Database configuration
	if configCopy.Database.URL != "" {
		configCopy.Database.URL, encErr = ce.encryptField(configCopy.Database.URL)
		if encErr != nil {
			return nil, fmt.Errorf("failed to encrypt database url: %w", encErr)
		}
	}

	// Event bus configurations
	if configCopy.EventBus.PostgreSQL != nil && configCopy.EventBus.PostgreSQL.URL != "" {
		configCopy.EventBus.PostgreSQL.URL, encErr = ce.encryptField(configCopy.EventBus.PostgreSQL.URL)
		if encErr != nil {
			return nil, fmt.Errorf("failed to encrypt postgresql url: %w", encErr)
		}
	}

	if configCopy.EventBus.Redis != nil && configCopy.EventBus.Redis.URL != "" {
		configCopy.EventBus.Redis.URL, encErr = ce.encryptField(configCopy.EventBus.Redis.URL)
		if encErr != nil {
			return nil, fmt.Errorf("failed to encrypt redis url: %w", encErr)
		}
	}

	if configCopy.EventBus.NATS != nil && configCopy.EventBus.NATS.URL != "" {
		configCopy.EventBus.NATS.URL, encErr = ce.encryptField(configCopy.EventBus.NATS.URL)
		if encErr != nil {
			return nil, fmt.Errorf("failed to encrypt nats url: %w", encErr)
		}
	}

	if configCopy.EventBus.RabbitMQ != nil && configCopy.EventBus.RabbitMQ.URL != "" {
		configCopy.EventBus.RabbitMQ.URL, encErr = ce.encryptField(configCopy.EventBus.RabbitMQ.URL)
		if encErr != nil {
			return nil, fmt.Errorf("failed to encrypt rabbitmq url: %w", encErr)
		}
	}

	// Encrypt plugin configurations
	if encErr := ce.encryptPluginConfigs(&configCopy); encErr != nil {
		return nil, fmt.Errorf("failed to encrypt plugin configs: %w", encErr)
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(configCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal encrypted config: %w", err)
	}

	return jsonData, nil
}

// DecryptConfig decrypts sensitive fields in the configuration
// Takes encrypted config JSON and returns decrypted config
func (ce *ConfigEncryptor) DecryptConfig(encryptedJSON []byte) (*models.Config, error) {
	var config models.Config

	// Unmarshal JSON
	if err := json.Unmarshal(encryptedJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Decrypt sensitive fields
	var err error

	// Core secrets
	if config.Secret != "" {
		config.Secret, err = ce.decryptField(config.Secret)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret: %w", err)
		}
	}

	// Database configuration
	if config.Database.URL != "" {
		config.Database.URL, err = ce.decryptField(config.Database.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt database url: %w", err)
		}
	}

	// Event bus configurations
	if config.EventBus.PostgreSQL != nil && config.EventBus.PostgreSQL.URL != "" {
		config.EventBus.PostgreSQL.URL, err = ce.decryptField(config.EventBus.PostgreSQL.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt postgresql url: %w", err)
		}
	}

	if config.EventBus.Redis != nil && config.EventBus.Redis.URL != "" {
		config.EventBus.Redis.URL, err = ce.decryptField(config.EventBus.Redis.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt redis url: %w", err)
		}
	}

	if config.EventBus.NATS != nil && config.EventBus.NATS.URL != "" {
		config.EventBus.NATS.URL, err = ce.decryptField(config.EventBus.NATS.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt nats url: %w", err)
		}
	}

	if config.EventBus.RabbitMQ != nil && config.EventBus.RabbitMQ.URL != "" {
		config.EventBus.RabbitMQ.URL, err = ce.decryptField(config.EventBus.RabbitMQ.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt rabbitmq url: %w", err)
		}
	}

	// Decrypt plugin configurations
	if err := ce.decryptPluginConfigs(&config); err != nil {
		return nil, fmt.Errorf("failed to decrypt plugin configs: %w", err)
	}

	return &config, nil
}

// encryptField encrypts a single field value using AES-256-GCM
func (ce *ConfigEncryptor) encryptField(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	return ce.tokenService.Encrypt(value)
}

// decryptField decrypts a single field value using AES-256-GCM
func (ce *ConfigEncryptor) decryptField(encryptedValue string) (string, error) {
	if encryptedValue == "" {
		return "", nil
	}
	return ce.tokenService.Decrypt(encryptedValue)
}

// encryptPluginConfigs recursively encrypts sensitive fields in plugin configurations
func (ce *ConfigEncryptor) encryptPluginConfigs(config *models.Config) error {
	if config.Plugins == nil {
		return nil
	}

	// Convert plugins config to map to allow dynamic encryption
	pluginsJSON, err := json.Marshal(config.Plugins)
	if err != nil {
		return fmt.Errorf("failed to marshal plugins: %w", err)
	}

	var pluginsMap map[string]any
	if err := json.Unmarshal(pluginsJSON, &pluginsMap); err != nil {
		return fmt.Errorf("failed to unmarshal plugins: %w", err)
	}

	// Encrypt sensitive fields in plugins
	sensitiveKeys := map[string]bool{
		"client_secret": true,
		"client_id":     true,
		"secret":        true,
		"url":           true,
		"auth_url":      true,
		"token_url":     true,
		"user_info_url": true,
		"smtp_pass":     true,
		"smtp_user":     true,
		"smtp_host":     true,
		"brokers":       true,
	}

	encryptMapValues(pluginsMap, sensitiveKeys, ce)

	encryptedJSON, err := json.Marshal(pluginsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted plugins: %w", err)
	}

	if err := json.Unmarshal(encryptedJSON, &config.Plugins); err != nil {
		return fmt.Errorf("failed to update plugins: %w", err)
	}

	return nil
}

// decryptPluginConfigs recursively decrypts sensitive fields in plugin configurations
func (ce *ConfigEncryptor) decryptPluginConfigs(config *models.Config) error {
	if config.Plugins == nil {
		return nil
	}

	// Convert plugins config to map to allow dynamic decryption
	pluginsJSON, err := json.Marshal(config.Plugins)
	if err != nil {
		return fmt.Errorf("failed to marshal plugins: %w", err)
	}

	var pluginsMap map[string]any
	if err := json.Unmarshal(pluginsJSON, &pluginsMap); err != nil {
		return fmt.Errorf("failed to unmarshal plugins: %w", err)
	}

	// Decrypt sensitive fields in plugins
	sensitiveKeys := map[string]bool{
		"client_secret": true,
		"client_id":     true,
		"secret":        true,
		"url":           true,
		"auth_url":      true,
		"token_url":     true,
		"user_info_url": true,
		"smtp_pass":     true,
		"smtp_user":     true,
		"smtp_host":     true,
		"brokers":       true,
	}

	decryptMapValues(pluginsMap, sensitiveKeys, ce)

	decryptedJSON, err := json.Marshal(pluginsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal decrypted plugins: %w", err)
	}

	if err := json.Unmarshal(decryptedJSON, &config.Plugins); err != nil {
		return fmt.Errorf("failed to update plugins: %w", err)
	}

	return nil
}

// encryptMapValues recursively encrypts values in a map if their keys are in sensitiveKeys
func encryptMapValues(data map[string]any, sensitiveKeys map[string]bool, ce *ConfigEncryptor) {
	for key, value := range data {
		if sensitiveKeys[key] {
			if strVal, ok := value.(string); ok && strVal != "" {
				encrypted, err := ce.encryptField(strVal)
				if err == nil {
					data[key] = encrypted
				}
			}
		}

		// Recursively process nested maps
		if nestedMap, ok := value.(map[string]any); ok {
			encryptMapValues(nestedMap, sensitiveKeys, ce)
		}
	}
}

// decryptMapValues recursively decrypts values in a map if their keys are in sensitiveKeys
func decryptMapValues(data map[string]any, sensitiveKeys map[string]bool, ce *ConfigEncryptor) {
	for key, value := range data {
		if sensitiveKeys[key] {
			if strVal, ok := value.(string); ok && strVal != "" {
				decrypted, err := ce.decryptField(strVal)
				if err == nil {
					data[key] = decrypted
				}
			}
		}

		// Recursively process nested maps
		if nestedMap, ok := value.(map[string]any); ok {
			decryptMapValues(nestedMap, sensitiveKeys, ce)
		}
	}
}
