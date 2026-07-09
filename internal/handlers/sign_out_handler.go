package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/types"
	"github.com/Authula/authula/internal/usecases"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
)

type SignOutHandler struct {
	UseCase *usecases.SignOutUseCase
}

func (h *SignOutHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var request types.SignOutRequest
		if err := util.ParseJSON(r, &request); err != nil {
			// If no body is provided, we'll default to using an empty request.
			request = types.SignOutRequest{}
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		result, err := h.UseCase.SignOut(ctx, reqCtx.Actor.ID, request.SessionID, request.SignOutAll)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
				"message": "failed to sign out",
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.Values[models.ContextAuthSignOut.String()] = true

		reqCtx.SetJSONResponse(http.StatusOK, &types.SignOutResponse{
			Message: result.Message,
		})
	}
}
