package util

import (
	"fmt"
	"strings"

	mapstructure "github.com/go-viper/mapstructure/v2"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// ParsePluginConfig is a utility function to parse plugin configuration from the generic config map.
// It uses mapstructure with custom decode hooks to handle:
// - Time duration strings (e.g., "5m", "300s") via StringToTimeDurationHookFunc
// - Comma-separated string slices via StringToSliceHookFunc
func ParsePluginConfig(source any, target any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
		Result:  target,
		TagName: "json",
	})
	if err != nil {
		return err
	}
	return decoder.Decode(source)
}

// LoadPluginConfig parses the configuration for a specific plugin from the main config.
// First checks PreParsedConfigs (for type safety when plugins are instantiated manually).
// Falls back to unmarshalling from Plugins map (for config file loading).
func LoadPluginConfig[T any](config *models.Config, pluginID string, target *T) error {
	if config == nil || config.Plugins == nil {
		return nil
	}

	// Check for pre-parsed config (when plugins are instantiated manually)
	if config.PreParsedConfigs != nil {
		if preParsed, ok := config.PreParsedConfigs[pluginID]; ok && preParsed != nil {
			// Direct assignment - preserves type safety, skips marshalling
			if typedConfig, ok := preParsed.(T); ok {
				*target = typedConfig
				return nil
			}
		}
	}

	// Fallback to unmarshalling from Plugins map (when plugins are built from config)
	rawConfig, ok := config.Plugins[pluginID]
	if !ok || rawConfig == nil {
		return nil
	}

	return ParsePluginConfig(rawConfig, target)
}

// IsPluginEnabled checks if a plugin is enabled based on its metadata and configuration.
func IsPluginEnabled(config *models.Config, pluginID string) bool {
	if config == nil {
		return false
	}

	if config.PreParsedConfigs != nil {
		if preParsed, ok := config.PreParsedConfigs[pluginID]; ok && preParsed != nil {
			if enabled, found := getEnabledFromConfig(preParsed); found {
				return enabled
			}
		}
	}

	if config.Plugins == nil {
		return false
	}

	rawConfig, ok := config.Plugins[pluginID]
	if !ok || rawConfig == nil {
		return false
	}

	if enabled, found := getEnabledFromConfig(rawConfig); found {
		return enabled
	}

	return true
}

// ConvertRouteMetadata converts a list of RouteMapping configs into the internal
// route metadata map used by the router for plugin routing.
// Returns a map keyed by "METHOD:path" containing metadata with "plugins" field.
// Example:
//
//	Input: RouteMapping{Path: "/me", Method: "GET", Plugins: ["session.auth"]}
//	Output: {"GET:/me": {"plugins": ["session.auth"]}}
func ConvertRouteMetadata(routes []models.RouteMapping) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	for _, route := range routes {
		if route.Path == "" {
			return nil, fmt.Errorf("route path cannot be empty")
		}
		if route.Method == "" {
			return nil, fmt.Errorf("route method cannot be empty for path %s", route.Path)
		}

		key := route.Method + ":" + route.Path
		metadata := make(map[string]any)
		metadata["plugins"] = route.Plugins
		result[key] = metadata
	}

	return result, nil
}

// ApplyBasePathToMetadataKey applies a basePath prefix to a metadata key (METHOD:path)
// If basePath is empty, the key is returned unchanged
// Example: ApplyBasePathToMetadataKey("GET:/auth/me", "/api") returns "GET:/api/auth/me"
func ApplyBasePathToMetadataKey(key, basePath string) string {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return key
	}
	method := parts[0]
	path := parts[1]

	// Ensure basePath has no trailing slash
	base := strings.TrimSuffix(basePath, "/")
	path = "/" + strings.TrimPrefix(path, "/")

	fullPath := base + path
	fullPath = strings.TrimSuffix(fullPath, "/") // normalize

	return method + ":" + fullPath
}
