package handlers

import (
	"net/http"
	"net/url"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/usecases"
)

type VerifyEmailHandler struct {
	VerifyEmailUseCase *usecases.VerifyEmailUseCase
}

func (h *VerifyEmailHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "token is required",
			})
			reqCtx.Handled = true
			return
		}

		verificationType, err := h.VerifyEmailUseCase.VerifyEmail(ctx, tokenStr)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		callbackURL := r.URL.Query().Get("callback_url")
		if callbackURL != "" {
			if verificationType == models.TypePasswordResetRequest {
				u, err := url.Parse(callbackURL)
				if err != nil {
					reqCtx.RedirectURL = callbackURL + "?token=" + url.QueryEscape(tokenStr)
					reqCtx.ResponseStatus = http.StatusFound
				} else {
					q := u.Query()
					q.Set("token", tokenStr)
					u.RawQuery = q.Encode()
					reqCtx.RedirectURL = u.String()
					reqCtx.ResponseStatus = http.StatusFound
				}
			} else {
				reqCtx.RedirectURL = callbackURL
				reqCtx.ResponseStatus = http.StatusFound
			}
			reqCtx.Handled = true
		} else {
			reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
				"message": "email verified successfully",
			})
		}
	}
}
