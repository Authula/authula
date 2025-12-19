package handlers

import (
	"net/http"

	signout "github.com/GoBetterAuth/go-better-auth/internal/auth/sign-out"
	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type SignOutResponse struct {
	Message string `json:"message"`
}

type SignOutHandler struct {
	Config  *models.Config
	UseCase signout.SignOutUseCase
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

	if err := h.UseCase.SignOut(r.Context(), cookie.Value); err != nil {
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

func (h *SignOutHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
