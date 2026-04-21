package util

import (
	"encoding/json"
	"fmt"
	"strings"

	mapstructure "github.com/go-viper/mapstructure/v2"

	"github.com/Authula/authula/models"
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

func getEnabledFromConfig(config any) (bool, bool) {
	if config == nil {
		return false, false
	}

	if configMap, ok := config.(map[string]any); ok {
		if enabled, found := configMap["enabled"]; found {
			if value, ok := enabled.(bool); ok {
				return value, true
			}
		}
		return false, false
	}

	data, err := json.Marshal(config)
	if err != nil {
		return false, false
	}

	var parsedConfig map[string]any
	if err := json.Unmarshal(data, &parsedConfig); err != nil {
		return false, false
	}

	if enabled, found := parsedConfig["enabled"]; found {
		if value, ok := enabled.(bool); ok {
			return value, true
		}
	}

	return false, false
}

// ConvertRouteMetadata converts a list of RouteMapping configs into the internal
// route metadata map used by the router for plugin routing.
// Returns a map keyed by "METHOD:path" containing metadata with "plugins" and "permissions" fields.
// Route strings can either be METHOD:/path or just /path.
// Bare paths apply to all supported HTTP methods.
func ConvertRouteMetadata(routes []models.RouteMapping) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	for _, route := range routes {
		if len(route.Paths) == 0 {
			return nil, fmt.Errorf("route paths cannot be empty")
		}

		for _, routePath := range route.Paths {
			trimmed := strings.TrimSpace(routePath)
			if trimmed == "" {
				return nil, fmt.Errorf("route path cannot be empty")
			}

			methods, pattern, err := parseRouteMappingPath(trimmed)
			if err != nil {
				return nil, err
			}

			for _, method := range methods {
				key := method + ":" + pattern
				metadata := result[key]
				if metadata == nil {
					metadata = make(map[string]any)
					result[key] = metadata
				}

				metadata["plugins"] = MergeStringSlices(ReadStringSliceFromMetadata(metadata, "plugins"), route.Plugins)
				metadata["permissions"] = MergeStringSlices(ReadStringSliceFromMetadata(metadata, "permissions"), route.Permissions)
			}
		}
	}

	return result, nil
}

func parseRouteMappingPath(routePath string) ([]string, string, error) {
	if strings.HasPrefix(routePath, "/") {
		return SupportedRouteMethods, NormalizeRoutePattern(routePath), nil
	}

	parts := strings.SplitN(routePath, ":", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("route path must be either METHOD:/path or /path, got %q", routePath)
	}

	method := strings.ToUpper(strings.TrimSpace(parts[0]))
	pattern := NormalizeRoutePattern(parts[1])
	if method == "" {
		return nil, "", fmt.Errorf("route method cannot be empty for path %q", routePath)
	}

	return []string{method}, pattern, nil
}

// ApplyBasePathToMetadataKey applies a basePath prefix to a metadata key (METHOD:path)
// If basePath is empty, the key is returned unchanged
// Example: ApplyBasePathToMetadataKey("GET:/me", "/api/auth") returns "GET:/api/auth/me"
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

// IsPathDisabled reports whether a request method/path matches one of the configured disabled paths.
// Disabled entries support exact METHOD:/path rules and wildcard prefixes such as /organizations/*.
func IsPathDisabled(method, path string, disabledPaths []string, basePath string) bool {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	normalizedPath := NormalizeRoutePattern(path)
	normalizedBasePath := NormalizeRoutePattern(basePath)

	for _, disabledPath := range disabledPaths {
		entry := strings.TrimSpace(disabledPath)
		if entry == "" {
			continue
		}

		entryMethod, entryPattern := parseDisabledPathEntry(entry)
		if entryPattern == "" {
			continue
		}

		if entryMethod != "" && entryMethod != normalizedMethod {
			continue
		}

		if disabledPathMatches(normalizedPath, normalizedBasePath, entryPattern) {
			return true
		}
	}

	return false
}

func parseDisabledPathEntry(entry string) (string, string) {
	if strings.Contains(entry, ":") {
		parts := strings.SplitN(entry, ":", 2)
		method := strings.ToUpper(strings.TrimSpace(parts[0]))
		pattern := NormalizeRoutePattern(strings.TrimSpace(parts[1]))
		return method, pattern
	}

	return "", NormalizeRoutePattern(entry)
}

func disabledPathMatches(requestPath, basePath, pattern string) bool {
	if pattern == "" {
		return false
	}

	if prefix, ok := strings.CutSuffix(pattern, "/*"); ok {
		if prefix == "" {
			return true
		}

		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}

		if basePath != "/" {
			candidate := NormalizeRoutePattern(strings.TrimPrefix(requestPath, basePath))
			if candidate == prefix || strings.HasPrefix(candidate, prefix+"/") {
				return true
			}
		}

		return false
	}

	if requestPath == pattern {
		return true
	}

	if basePath != "/" {
		candidate := NormalizeRoutePattern(strings.TrimPrefix(requestPath, basePath))
		return candidate == pattern
	}

	return false
}
