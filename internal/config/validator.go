package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// ValidateAndMergeConfig converts a config to a map, merges a key-value pair,
// validates the result, and returns the updated config struct.
func ValidateAndMergeConfig(current *models.Config, key string, value any) (*models.Config, error) {
	configMap, err := structToMap(current)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config to map: %w", err)
	}

	setNestedMapValue(configMap, key, value)

	var updatedConfig models.Config
	configJSON, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config map: %w", err)
	}

	if err := json.Unmarshal(configJSON, &updatedConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if util.Validate != nil {
		if err := util.Validate.Struct(&updatedConfig); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return &updatedConfig, nil
}

// structToMap converts a struct to a map using JSON marshaling/unmarshaling.
// This ensures that struct tags (json, toml) are properly respected.
func structToMap(s any) (map[string]any, error) {
	var m map[string]any
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// setNestedMapValue sets a value in a nested map using dot notation.
// For example: "email.smtp_host" will set the value in m["email"]["smtp_host"]
func setNestedMapValue(m map[string]any, keyPath string, value any) {
	parts := strings.Split(keyPath, ".")
	if len(parts) == 0 {
		return
	}

	// Navigate to the parent map
	current := m
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		if _, ok := current[key]; !ok {
			current[key] = make(map[string]any)
		}
		// Type assert to map[string]any
		if nested, ok := current[key].(map[string]any); ok {
			current = nested
		} else {
			// If it's not a map, replace it with a new map
			nested := make(map[string]any)
			current[key] = nested
			current = nested
		}
	}

	// Set the final value
	current[parts[len(parts)-1]] = value
}
