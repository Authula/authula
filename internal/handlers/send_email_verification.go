package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/middleware"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type SendEmailVerificationResponse struct {
	Message string `json:"message"`
}

type SendEmailVerificationHandlerPayload struct {
	CallbackURL *string `json:"callback_url,omitempty"`
}

type SendEmailVerificationHandler struct {
	Config      *models.Config
	AuthService *auth.Service
}

func (h *SendEmailVerificationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok || userID == "" {
		util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
		return
	}

	var payload SendEmailVerificationHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid request"})
		return
	}
	if err := util.Validate.Struct(payload); err != nil {
		util.JSONResponse(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}

	if err := h.AuthService.SendVerificationEmail(userID, payload.CallbackURL); err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}

	resp := SendEmailVerificationResponse{Message: "Verification email sent"}
	util.JSONResponse(w, http.StatusOK, resp)
}

func (h *SendEmailVerificationHandler) Handler() http.Handler {
	return Wrap(h)
}
