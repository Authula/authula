package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
	"github.com/Authula/authula/plugins/magic-link/usecases"
	"github.com/Authula/authula/util"
)

type ExchangeHandler struct {
	UseCase usecases.ExchangeUseCase
}

func (h *ExchangeHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if reqCtx.Actor != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "you're already authenticated.",
			})
			reqCtx.Handled = true
			return
		}

		var request types.MagicLinkExchangeRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		userAgent := r.UserAgent()
		result, err := h.UseCase.Exchange(ctx, request.Token, &reqCtx.ClientIP, &userAgent)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetActorInContext(&models.Actor{
			ID:   result.User.ID,
			Type: models.ActorUser,
		})
		reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
		reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
		reqCtx.Values[models.ContextAuthSuccess.String()] = true

		reqCtx.SetJSONResponse(http.StatusOK, &types.MagicLinkExchangeResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
