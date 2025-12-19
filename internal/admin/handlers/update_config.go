package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// PATCH /admin/config

type AdminUpdateConfigResponse struct {
	Message string `json:"message"`
}

type AdminUpdateConfigHandlerPayload struct {
	Key   string `json:"key" validate:"required"`
	Value any    `json:"value" validate:"required"`
}

type AdminUpdateConfigHandler struct {
	ConfigManager models.ConfigManager
}

func (h *AdminUpdateConfigHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var payload AdminUpdateConfigHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
		return
	}
	if err := util.Validate.Struct(payload); err != nil {
		util.JSONResponse(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}

	if err := h.ConfigManager.Update(payload.Key, payload.Value); err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "Failed to update config: " + err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminUpdateConfigHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
