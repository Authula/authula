package handlers

import (
	"net/http"
	"net/url"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
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
					http.Redirect(reqCtx.ResponseWriter, r, callbackURL+"?token="+url.QueryEscape(tokenStr), http.StatusFound)
				} else {
					q := u.Query()
					q.Set("token", tokenStr)
					u.RawQuery = q.Encode()
					http.Redirect(reqCtx.ResponseWriter, r, u.String(), http.StatusFound)
				}
			} else {
				http.Redirect(reqCtx.ResponseWriter, r, callbackURL, http.StatusFound)
			}
		} else {
			reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
				"message": "email verified successfully",
			})
		}
	}
}
