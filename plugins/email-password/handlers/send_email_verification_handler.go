package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/usecases"
)

type SendEmailVerificationRequestPayload struct {
	Email       string `json:"email"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type SendEmailVerificationHandler struct {
	UseCase *usecases.SendEmailVerificationUseCase
}

func (h *SendEmailVerificationHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload SendEmailVerificationRequestPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		err := h.UseCase.Send(ctx, payload.Email, &payload.CallbackURL)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "If an account exists with this email, a verification link has been sent.",
		})
	}
}
