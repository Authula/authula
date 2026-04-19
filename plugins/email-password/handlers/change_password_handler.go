package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

type ChangePasswordHandler struct {
	UseCase usecases.ChangePasswordUseCase
}

func (h *ChangePasswordHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, ok := models.GetRequestContext(ctx)
		if !ok || reqCtx == nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		var request types.ChangePasswordRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		err := h.UseCase.ChangePassword(ctx, request.Token, request.Password)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "password updated",
		})
	}
}
