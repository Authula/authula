package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

type SignOutResponse struct {
	Message string `json:"message"`
}

type SignOutHandler struct {
	Config      *domain.Config
	AuthService *auth.Service
}

func (h *SignOutHandler) Handle(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.Config.Session.CookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			util.JSONResponse(w, http.StatusNotFound, map[string]any{"message": "cookie not found"})
			return
		}
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}

	if err := h.AuthService.SignOut(cookie.Value); err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": err.Error()})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.Session.CookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
	})

	if h.Config.CSRF.Enabled {
		http.SetCookie(w, &http.Cookie{
			Name:     h.Config.CSRF.CookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Secure:   true,
			HttpOnly: false,
			SameSite: http.SameSiteNoneMode,
		})
	}

	resp := SignOutResponse{Message: "Signed out successfully"}
	util.JSONResponse(w, http.StatusOK, resp)
}

func (h *SignOutHandler) Handler() http.Handler {
	return Wrap(h)
}
