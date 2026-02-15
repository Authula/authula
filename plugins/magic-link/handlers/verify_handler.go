package handlers

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/usecases"
)

type VerifyHandler struct {
	UseCase        usecases.VerifyUseCase
	TrustedOrigins []string
}

func (h *VerifyHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "token is required",
			})
			reqCtx.Handled = true
			return
		}

		userAgent := r.UserAgent()
		tokenForExchange, err := h.UseCase.Verify(ctx, token, &reqCtx.ClientIP, &userAgent)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		callbackURL := r.URL.Query().Get("callback_url")
		if callbackURL != "" {
			parsedURL, err := url.Parse(callbackURL)
			if err != nil {
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
					"message": "invalid callback_url",
				})
				reqCtx.Handled = true
				return
			}
			origin := parsedURL.Scheme + "://" + parsedURL.Host
			trusted := slices.Contains(h.TrustedOrigins, origin)
			if !trusted {
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
					"message": "callback_url is not a trusted origin",
				})
				reqCtx.Handled = true
				return
			}
			query := parsedURL.Query()
			query.Set("token", tokenForExchange)
			parsedURL.RawQuery = query.Encode()
			reqCtx.RedirectURL = parsedURL.String()
			reqCtx.ResponseStatus = http.StatusFound
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.VerifyResponse{
			Message: "magic link successfully verified",
			Token:   tokenForExchange,
		})
	}
}
