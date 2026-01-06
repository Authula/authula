package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
)

type SendVerificationEmailRequestPayload struct {
	Email       string `json:"email"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type SendVerificationEmailHandler struct {
	UseCase *usecases.SendVerificationEmailUseCase
}

func (h *SendVerificationEmailHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload SendVerificationEmailRequestPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "invalid request",
			})
			reqCtx.Handled = true
			return
		}

		err := h.UseCase.Send(ctx, payload.Email, &payload.CallbackURL)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
				"message": "internal server error",
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "verification email sent",
		})
	}
}
