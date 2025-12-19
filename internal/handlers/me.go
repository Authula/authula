package handlers

import (
	"net/http"

	me "github.com/GoBetterAuth/go-better-auth/internal/auth/me"
	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/middleware"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type MeHandler struct {
	Config  *models.Config
	UseCase me.MeUseCase
}

func (h *MeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok || userID == "" {
		util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
		return
	}

	result, err := h.UseCase.GetMe(r.Context(), userID)
	if err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}
	if result.User == nil {
		util.JSONResponse(w, http.StatusNotFound, map[string]any{"message": "user not found"})
		return
	}
	if result.Session == nil {
		util.JSONResponse(w, http.StatusNotFound, map[string]any{"message": "session not found"})
		return
	}

	util.JSONResponse(w, http.StatusOK, result)
}

func (h *MeHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
