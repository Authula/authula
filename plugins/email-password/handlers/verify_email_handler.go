package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
)

type VerifyEmailHandler struct {
	UseCase *usecases.VerifyEmailUseCase
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

		err := h.UseCase.VerifyEmail(ctx, tokenStr)
		if err != nil {
			switch err {
			case constants.ErrInvalidOrExpiredToken:
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
					"message": "invalid or expired token",
				})
			case constants.ErrUserNotFound:
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
					"message": "user not found",
				})
			default:
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
					"message": "internal server error",
				})
			}

			reqCtx.Handled = true
			return
		}

		callbackURL := r.URL.Query().Get("callback_url")
		if callbackURL != "" {
			http.Redirect(w, r, callbackURL, http.StatusFound)
		} else {
			reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
				"message": "email verified successfully",
			})
		}
	}
}
