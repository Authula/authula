package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/middleware"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type EmailChangeResponse struct {
	Message string `json:"message"`
}

type EmailChangeHandlerPayload struct {
	Email       string  `json:"email" validate:"required,email"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

type EmailChangeHandler struct {
	Config      *models.Config
	AuthService *auth.Service
}

func (h *EmailChangeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if !h.Config.User.ChangeEmail.Enabled {
		util.JSONResponse(w, http.StatusNotImplemented, map[string]any{"message": "email change is disabled"})
		return
	}

	userID, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok || userID == "" {
		util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
		return
	}

	var payload EmailChangeHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid request"})
		return
	}
	if err := util.Validate.Struct(payload); err != nil {
		util.JSONResponse(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}

	if err := h.AuthService.EmailChange(userID, payload.Email, payload.CallbackURL); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
		return
	}

	resp := EmailChangeResponse{Message: "Email change verification email sent"}
	util.JSONResponse(w, http.StatusOK, resp)
}

func (h *EmailChangeHandler) Handler() http.Handler {
	return Wrap(h)
}
