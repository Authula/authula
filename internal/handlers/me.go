package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/middleware"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type MeResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

type MeHandler struct {
	Config      *models.Config
	AuthService *auth.Service
}

func (h *MeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok || userID == "" {
		util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
		return
	}

	user, err := h.AuthService.UserService.GetUserByID(userID)
	if err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "failed to retrieve user"})
		return
	}
	if user == nil {
		util.JSONResponse(w, http.StatusNotFound, map[string]any{"message": "user not found"})
		return
	}

	session, err := h.AuthService.SessionService.GetSessionByUserID(userID)
	if err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "failed to retrieve session"})
		return
	}
	if session == nil {
		util.JSONResponse(w, http.StatusNotFound, map[string]any{"message": "session not found"})
		return
	}

	response := MeResponse{
		User:    user,
		Session: session,
	}

	util.JSONResponse(w, http.StatusOK, response)
}

func (h *MeHandler) Handler() http.Handler {
	return Wrap(h)
}
