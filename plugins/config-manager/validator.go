package configmanager

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// ValidateAndMergeConfig converts a config to a map, merges a key-value pair,
// validates the result, and returns the updated config struct.
// Optimized to reduce unnecessary marshal/unmarshal operations.
func ValidateAndMergeConfig(current *models.Config, key string, value any) (*models.Config, error) {
	// Create a shallow copy to avoid modifying the original
	updatedConfig := *current

	// Apply the update using reflection for single field updates (much faster than serialization)
	if err := applyConfigUpdate(&updatedConfig, key, value); err != nil {
		// Fall back to marshal/unmarshal method if reflection fails
		configMap, mapErr := structToMap(current)
		if mapErr != nil {
			return nil, fmt.Errorf("failed to convert config to map: %w", mapErr)
		}

		setNestedMapValue(configMap, key, value)

		configJSON, jsonErr := json.Marshal(configMap)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to marshal config map: %w", jsonErr)
		}

		if jsonErr := json.Unmarshal(configJSON, &updatedConfig); jsonErr != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", jsonErr)
		}
	}

	// Validate the updated config
	if util.Validate != nil {
		if err := util.Validate.Struct(&updatedConfig); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return &updatedConfig, nil
}

// applyConfigUpdate applies a single config update using reflection instead of serialization.
// This is significantly faster for single field updates than marshal/unmarshal.
// Returns error if the path cannot be resolved, in which case the caller should fall back to serialization.
func applyConfigUpdate(config *models.Config, keyPath string, value any) error {
	parts := strings.Split(keyPath, ".")
	if len(parts) == 0 {
		return fmt.Errorf("empty key path")
	}

	current := reflect.ValueOf(config).Elem()

	// Navigate to the target field using reflection
	for i := 0; i < len(parts)-1; i++ {
		field := getFieldByJSONTag(current, parts[i])
		if !field.IsValid() {
			return fmt.Errorf("field not found: %s", parts[i])
		}

		// Dereference pointer if needed
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				return fmt.Errorf("pointer field is nil: %s", parts[i])
			}
			current = field.Elem()
		} else {
			current = field
		}
	}

	// Set the final value
	finalFieldName := parts[len(parts)-1]
	field := getFieldByJSONTag(current, finalFieldName)
	if !field.IsValid() {
		return fmt.Errorf("field not found: %s", finalFieldName)
	}

	if !field.CanSet() {
		return fmt.Errorf("field cannot be set: %s", finalFieldName)
	}

	// Convert value to the target type
	valueReflect := reflect.ValueOf(value)
	if !valueReflect.Type().AssignableTo(field.Type()) {
		// Try to convert if types don't match exactly
		if !valueReflect.Type().ConvertibleTo(field.Type()) {
			return fmt.Errorf("value type %s not assignable to field type %s", valueReflect.Type(), field.Type())
		}
		valueReflect = valueReflect.Convert(field.Type())
	}

	field.Set(valueReflect)
	return nil
}

// getFieldByJSONTag finds a struct field by its JSON tag name
func getFieldByJSONTag(v reflect.Value, tagName string) reflect.Value {
	t := v.Type()

	// Check if the type is a struct before attempting FieldByName
	if t.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	// Try direct field name first
	if field, ok := t.FieldByName(tagName); ok {
		return v.FieldByIndex(field.Index)
	}

	// Try to find by JSON tag
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		// Extract the field name from the tag (before any comma)
		jsonFieldName := strings.Split(jsonTag, ",")[0]
		if jsonFieldName == tagName {
			return v.Field(i)
		}
	}

	return reflect.Value{}
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
