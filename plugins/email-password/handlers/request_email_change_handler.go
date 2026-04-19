package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

type RequestEmailChangeHandler struct {
	UseCase usecases.RequestEmailChangeUseCase
}

func (h *RequestEmailChangeHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if reqCtx.UserID == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
			reqCtx.Handled = true
			return
		}

		var request types.RequestEmailChangeRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
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

		err := h.UseCase.RequestChange(ctx, *reqCtx.UserID, request.NewEmail, request.CallbackURL)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "verification sent to new email",
		})
	}
}
