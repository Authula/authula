package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// PATCH /admin/config

type AdminUpdateConfigHandler struct {
	ConfigManager models.ConfigManager
}

func (h *AdminUpdateConfigHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
		return
	}

	if len(updates) == 0 {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "no configuration updates provided"})
		return
	}

	// Flatten nested config updates into dot-notation keys and apply them
	for key, value := range flattenConfig(updates) {
		if err := h.ConfigManager.Update(key, value); err != nil {
			util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "Failed to update config: " + err.Error()})
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
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

func (h *AdminUpdateConfigHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
