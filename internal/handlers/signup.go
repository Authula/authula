package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

type SignupHandlerPayload struct {
	Name        string  `json:"name" validate:"required"`
	Email       string  `json:"email" validate:"required,email"`
	Password    string  `json:"password" validate:"required"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

type SignUpHandler struct {
	Config      *domain.Config
	AuthService *auth.Service
}

func (h *SignUpHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if h.Config.EmailPassword.DisableSignUp {
		util.JSONResponse(w, http.StatusForbidden, map[string]any{"message": "sign-ups are disabled"})
		return
	}

	var payload SignupHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid request"})
		return
	}
	if err := util.Validate.Struct(payload); err != nil {
		util.JSONResponse(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}

	result, err := h.AuthService.SignUpWithEmailAndPassword(payload.Name, payload.Email, payload.Password, payload.CallbackURL)
	if err != nil {
		util.JSONResponse(w, http.StatusConflict, map[string]any{"message": err.Error()})
		return
	}

	isSecure, sameSite := util.GetCookieOptions(h.Config)

	if result.Token != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     h.Config.Session.CookieName,
			Value:    result.Token,
			Path:     "/",
			HttpOnly: true,
			Secure:   isSecure,
			SameSite: sameSite,
			MaxAge:   int(h.Config.Session.ExpiresIn.Seconds()),
		})
	}

	if h.Config.CSRF.Enabled {
		csrfToken, err := h.AuthService.TokenService.GenerateToken()
		if err != nil {
			util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "failed to generate CSRF token"})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     h.Config.CSRF.CookieName,
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
			Secure:   isSecure,
			SameSite: sameSite,
			MaxAge:   int(h.Config.CSRF.ExpiresIn.Seconds()),
		})

		result.CSRFToken = csrfToken
	}

	util.JSONResponse(w, http.StatusOK, result)
}

func (h *SignUpHandler) Handler() http.Handler {
	return Wrap(h)
}
