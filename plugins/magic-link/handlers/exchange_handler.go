package handlers

import (
	"net/http"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/usecases"
)

type ExchangeHandler struct {
	UseCase usecases.ExchangeUseCase
}

func (h *ExchangeHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload types.ExchangeRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}

		code := strings.TrimSpace(payload.Token)
		if code == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "token is required",
			})
			reqCtx.Handled = true
			return
		}

		userAgent := r.UserAgent()
		result, err := h.UseCase.Exchange(ctx, code, &reqCtx.ClientIP, &userAgent)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetUserIDInContext(result.User.ID)
		reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
		reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
		reqCtx.Values[models.ContextAuthSuccess.String()] = true

		reqCtx.SetJSONResponse(http.StatusOK, &types.ExchangeResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
