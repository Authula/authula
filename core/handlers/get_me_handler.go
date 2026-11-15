package handlers

import (
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/types"
	"github.com/Authula/authula/core/usecases"
	"github.com/Authula/authula/models"
)

type GetMeHandler struct {
	UseCase *usecases.GetMeUseCase
}

func (h *GetMeHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var sessionID *string
		if id, ok := reqCtx.Values[models.ContextSessionID.String()].(string); ok && id != "" {
			sessionID = &id
		}

		result, err := h.UseCase.GetMe(ctx, reqCtx.Actor.ID, sessionID)
		if err != nil {
			coreerrors.HandleErrorWithFallback(err, reqCtx, http.StatusInternalServerError)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetMeResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
