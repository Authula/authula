package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type ConfigManagerUpdateConfigHandler struct {
	ConfigManager models.ConfigManager
}

func (h *ConfigManagerUpdateConfigHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reqCtx, ok := models.GetRequestContext(ctx)
	if !ok {
		reqCtx = &models.RequestContext{
			Method:         r.Method,
			Path:           r.URL.Path,
			ResponseWriter: w,
			Request:        r,
			Values:         make(map[string]any),
		}
		r = r.WithContext(models.SetRequestContext(ctx, reqCtx))
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid request body"})
		reqCtx.Handled = true
		return
	}

	if len(updates) == 0 {
		reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "no configuration updates provided"})
		reqCtx.Handled = true
		return
	}

	// Track the last updated config to avoid fetching it again at the end
	var lastUpdatedConfig *models.Config

	// Flatten nested config updates into dot-notation keys and apply them
	for key, value := range flattenConfig(updates) {
		if err := h.ConfigManager.UpdateWithResult(key, value, &lastUpdatedConfig); err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to update configuration"})
			reqCtx.Handled = true
			return
		}
	}

	// Use the last updated config instead of fetching it again
	// This avoids an extra GetConfig() call which serializes the entire config
	responseConfig := lastUpdatedConfig
	if responseConfig == nil {
		responseConfig = h.ConfigManager.GetConfig()
	}

	reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
		"message": "configuration updated successfully",
		"data":    responseConfig,
	})
}

// flattenConfig converts nested map into dot-notation keys for the config manager
func flattenConfig(m map[string]any) map[string]any {
	result := make(map[string]any)
	flatten("", m, result)
	return result
}

func flatten(prefix string, m map[string]any, result map[string]any) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			flatten(fullKey, v, result)
		default:
			result[fullKey] = value
		}
	}
}
