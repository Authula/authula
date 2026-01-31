package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/usecases"
)

type AuthorizeHandler struct {
	UseCase *usecases.AuthorizeUseCase
}

func NewAuthorizeHandler(useCase *usecases.AuthorizeUseCase) *AuthorizeHandler {
	return &AuthorizeHandler{UseCase: useCase}
}

func (h *AuthorizeHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		providerID := r.PathValue("provider")
		if providerID == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": "provider is required",
			})
			reqCtx.Handled = true
			return
		}

		req := &types.AuthorizeRequest{
			ProviderID: providerID,
			RedirectTo: r.URL.Query().Get("redirect_to"),
		}

		resp, err := h.UseCase.Authorize(ctx, req)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")

		http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
			Name:     constants.CookieState,
			Value:    resp.StateCookie,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(time.Minute.Seconds()) * 5,
		})
		http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
			Name:     constants.CookieRedirectTo,
			Value:    resp.RedirectCookie,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(time.Minute.Seconds()) * 5,
		})
		if resp.VerifierCookie != nil {
			http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
				Name:     constants.CookieVerifier,
				Value:    *resp.VerifierCookie,
				Path:     "/",
				HttpOnly: true,
				Secure:   secure,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int(time.Minute.Seconds()) * 5,
			})
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.AuthorizeResponse{
			AuthURL: resp.AuthorizationURL,
		})
	}
}
