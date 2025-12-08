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

	isSecure, sameSite := util.GetCookieOptions(h.Config)

	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.Session.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
	})

	if h.Config.CSRF.Enabled {
		http.SetCookie(w, &http.Cookie{
			Name:     h.Config.CSRF.CookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: false,
			Secure:   isSecure,
			SameSite: sameSite,
		})
	}

	resp := SignOutResponse{Message: "Signed out successfully"}
	util.JSONResponse(w, http.StatusOK, resp)
}

func (h *SignOutHandler) Handler() http.Handler {
	return Wrap(h)
}
