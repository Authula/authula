package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type ConfigManagerGetConfigHandler struct {
	ConfigManager models.ConfigManager
}

func (h *ConfigManagerGetConfigHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqCtx, _ := models.GetRequestContext(ctx)

	config := h.ConfigManager.GetConfig()
	if config == nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"error": "failed to retrieve configuration"})
		reqCtx.Handled = true
		return
	}

	reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
		"message": "config retrieved successfully",
		"data":    config,
	})
}
