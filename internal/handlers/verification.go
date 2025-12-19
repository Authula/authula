package handlers

import (
	"net/http"

	verifyemail "github.com/GoBetterAuth/go-better-auth/internal/auth/verify-email"
	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type VerifyEmailHandler struct {
	Config  *models.Config
	UseCase verifyemail.VerifyEmailUseCase
}

func (h *VerifyEmailHandler) Handle(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "missing token"})
		return
	}
	callbackURL := r.URL.Query().Get("callback_url")

	result, err := h.UseCase.VerifyEmail(r.Context(), token)
	if err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
		return
	}

	if callbackURL != "" {
		http.Redirect(w, r, callbackURL, http.StatusSeeOther)
		return
	}

	util.JSONResponse(w, http.StatusOK, result)
}

func (h *VerifyEmailHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
