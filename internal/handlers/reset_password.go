package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type ResetPasswordResponse struct {
	Message string `json:"message"`
}

type ResetPasswordHandlerPayload struct {
	Email       string  `json:"email" validate:"required,email"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

type ResetPasswordHandler struct {
	Config      *models.Config
	AuthService *auth.Service
}

func (h *ResetPasswordHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var payload ResetPasswordHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid request"})
		return
	}
	if err := util.Validate.Struct(payload); err != nil {
		util.JSONResponse(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}

	if err := h.AuthService.ResetPassword(payload.Email, payload.CallbackURL); err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "password reset request failed"})
		return
	}

	resp := ResetPasswordResponse{Message: "Password reset email sent"}
	util.JSONResponse(w, http.StatusOK, resp)
}

func (h *ResetPasswordHandler) Handler() http.Handler {
	return Wrap(h)
}
