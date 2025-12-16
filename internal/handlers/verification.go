package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type VerifyEmailHandler struct {
	Config      *models.Config
	AuthService *auth.Service
}

func (h *VerifyEmailHandler) Handle(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "missing token"})
		return
	}
	callbackURL := r.URL.Query().Get("callback_url")

	result, err := h.AuthService.VerifyEmailToken(token)
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

func (h *VerifyEmailHandler) Handler() http.Handler {
	return Wrap(h)
}
