package handlers

import (
	"encoding/json"
	"net/http"

	changepassword "github.com/GoBetterAuth/go-better-auth/internal/auth/change-password"
	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type ChangePasswordResponse struct {
	Message string `json:"message"`
}

type ChangePasswordHandlerPayload struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}

type ChangePasswordHandler struct {
	Config  *models.Config
	UseCase changepassword.ChangePasswordUseCase
}

func (h *ChangePasswordHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var payload ChangePasswordHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid request"})
		return
	}
	if err := util.Validate.Struct(payload); err != nil {
		util.JSONResponse(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}

	if err := h.UseCase.ChangePassword(r.Context(), payload.Token, payload.NewPassword); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "password reset failed"})
		return
	}

	resp := ChangePasswordResponse{Message: "Password has been reset successfully"}
	util.JSONResponse(w, http.StatusOK, resp)
}

func (h *ChangePasswordHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
