package handlers

import (
	"encoding/json"
	"net/http"

	resetpassword "github.com/GoBetterAuth/go-better-auth/internal/auth/reset-password"
	"github.com/GoBetterAuth/go-better-auth/internal/common"
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
	Config  *models.Config
	UseCase resetpassword.ResetPasswordUseCase
}

func (h *ResetPasswordHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if !h.Config.EmailPassword.Enabled {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "email/password authentication is disabled"})
		return
	}

	var payload ResetPasswordHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid request"})
		return
	}
	if err := util.Validate.Struct(payload); err != nil {
		util.JSONResponse(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}

	if err := h.UseCase.ResetPassword(r.Context(), payload.Email, payload.CallbackURL); err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "password reset request failed"})
		return
	}

	resp := ResetPasswordResponse{Message: "Password reset email sent"}
	util.JSONResponse(w, http.StatusOK, resp)
}

func (h *ResetPasswordHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
