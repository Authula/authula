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
