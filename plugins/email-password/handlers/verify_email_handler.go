package handlers

import (
	"net/http"
	"net/url"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

type VerifyEmailHandler struct {
	VerifyEmailUseCase usecases.VerifyEmailUseCase
}

func (h *VerifyEmailHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		token := r.URL.Query().Get("token")
		var callbackURL string
		if cb := r.URL.Query().Get("callback_url"); cb != "" {
			callbackURL = cb
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

		if request.Token == "" {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "token is required",
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
				u, err := url.Parse(*request.CallbackURL)
				if err != nil {
					reqCtx.RedirectURL = *request.CallbackURL + "?token=" + url.QueryEscape(request.Token)
					reqCtx.ResponseStatus = http.StatusFound
				} else {
					q := u.Query()
					q.Set("token", request.Token)
					u.RawQuery = q.Encode()
					reqCtx.RedirectURL = u.String()
					reqCtx.ResponseStatus = http.StatusFound
				}
			} else {
				reqCtx.RedirectURL = *request.CallbackURL
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
