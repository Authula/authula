package handlers

import (
	"net/http"
	"net/url"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

type VerifyEmailHandler struct {
	VerifyEmailUseCase usecases.VerifyEmailUseCase
	TrustedOrigins     []string
}

func (h *VerifyEmailHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		token := r.URL.Query().Get("token")
		var callbackURL string
		var validatedCallbackURL *url.URL
		if cb := r.URL.Query().Get("callback_url"); cb != "" {
			parsedCallbackURL, err := util.IsTrustedCallbackURL(cb, h.TrustedOrigins)
			if err != nil {
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
					"message": err.Error(),
				})
				reqCtx.Handled = true
				return
			}

			validatedCallbackURL = parsedCallbackURL
			callbackURL = cb
		}

		if token == "" {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "token is required",
			})
			reqCtx.Handled = true
			return
		}

		if callbackURL != "" && validatedCallbackURL == nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "invalid callback_url",
			})
			reqCtx.Handled = true
			return
		}
		request := types.VerifyEmailRequest{
			Token:       token,
			CallbackURL: &callbackURL,
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		verificationType, err := h.VerifyEmailUseCase.VerifyEmail(ctx, request.Token)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		if request.CallbackURL != nil && *request.CallbackURL != "" {
			if verificationType == models.TypePasswordResetRequest {
				q := validatedCallbackURL.Query()
				q.Set("token", request.Token)
				validatedCallbackURL.RawQuery = q.Encode()
				reqCtx.RedirectURL = validatedCallbackURL.String()
				reqCtx.ResponseStatus = http.StatusFound
			} else {
				reqCtx.RedirectURL = validatedCallbackURL.String()
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
