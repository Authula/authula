package handlers

import (
	"encoding/json"
	"net/http"

	signin "github.com/GoBetterAuth/go-better-auth/internal/auth/sign-in"
	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type SignInHandlerPayload struct {
	Email       string  `json:"email" validate:"required,email"`
	Password    string  `json:"password" validate:"required"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

type SignInHandler struct {
	Config  *models.Config
	UseCase signin.SignInUseCase
}

func (h *SignInHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var payload SignInHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid request"})
		return
	}
	if err := util.Validate.Struct(payload); err != nil {
		util.JSONResponse(w, http.StatusUnprocessableEntity, map[string]any{"message": "validation failed"})
		return
	}

	result, err := h.UseCase.SignInWithEmailAndPassword(r.Context(), payload.Email, payload.Password, payload.CallbackURL)
	if err != nil {
		util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": err.Error()})
		return
	}

	isSecure, sameSite := util.GetCookieOptions(h.Config)

	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.Session.CookieName,
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(h.Config.Session.ExpiresIn.Seconds()),
		SameSite: sameSite,
		Secure:   isSecure,
	})

	if h.Config.CSRF.Enabled {
		http.SetCookie(w, &http.Cookie{
			Name:     h.Config.CSRF.CookieName,
			Value:    *result.CSRFToken,
			Path:     "/",
			HttpOnly: false,
			Secure:   isSecure,
			SameSite: sameSite,
			MaxAge:   int(h.Config.CSRF.ExpiresIn.Seconds()),
		})
	}

	util.JSONResponse(w, http.StatusOK, result)
}

func (h *SignInHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
