package handlers

import (
	"errors"
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/types"
	"github.com/Authula/authula/core/usecases"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/util"
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
			request = types.SignOutRequest{}
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		var currentSessionID *string
		if id, ok := reqCtx.Values[models.ContextSessionID.String()].(string); ok && id != "" {
			currentSessionID = &id
		}

		result, err := h.UseCase.SignOut(ctx, reqCtx.Actor.ID, currentSessionID, request.SessionID, request.SignOutAll)
		if err != nil {
			if errors.Is(err, coreerrors.ErrNotFound) || errors.Is(err, coreerrors.ErrForbidden) || errors.Is(err, coreerrors.ErrNoSessionToSignOut) {
				coreerrors.HandleError(err, reqCtx)
				return
			}
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
				"message": "failed to sign out",
			})
			reqCtx.Handled = true
			return
		}

		if result.ClearedCurrentSession {
			reqCtx.Values[models.ContextAuthSignOut.String()] = true
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.SignOutResponse{
			Message: result.Message,
		})
	}
}
