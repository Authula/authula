package util

import (
	"encoding/json"
)

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
