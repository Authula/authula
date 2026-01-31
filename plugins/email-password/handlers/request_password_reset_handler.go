package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/usecases"
)

type RequestPasswordResetPayload struct {
	Email       string `json:"email"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type RequestPasswordResetHandler struct {
	UseCase *usecases.RequestPasswordResetUseCase
}

func (h *RequestPasswordResetHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload RequestPasswordResetPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		// Always return 200 to prevent account enumeration
		_ = h.UseCase.RequestReset(ctx, payload.Email, &payload.CallbackURL)

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "if account exists, password reset link sent to email",
		})
	}
}
