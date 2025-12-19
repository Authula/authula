package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type AdminGetConfigHandler struct {
	ConfigManager models.ConfigManager
}

func (h *AdminGetConfigHandler) Handle(w http.ResponseWriter, r *http.Request) {
	config := h.ConfigManager.GetConfig()
	util.JSONResponse(w, http.StatusOK, config)
}

func (h *AdminGetConfigHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
