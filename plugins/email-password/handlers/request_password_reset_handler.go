package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
	"github.com/Authula/authula/util"
)

type RequestPasswordResetHandler struct {
	UseCase usecases.RequestPasswordResetUseCase
}

func (h *RequestPasswordResetHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var request types.RequestPasswordResetRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		_ = h.UseCase.RequestReset(ctx, request.Email, request.CallbackURL)

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "If an account exists, password reset link sent to email",
		})
	}
}
