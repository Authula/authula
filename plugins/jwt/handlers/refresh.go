package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/types"
	"github.com/Authula/authula/plugins/jwt/usecases"
)

type RefreshTokenHandler struct {
	Logger              models.Logger
	RefreshTokenUseCase usecases.RefreshTokenUseCase
}

func (h *RefreshTokenHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var request types.RefreshTokenRequest
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

		if request.RefreshToken == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "refresh_token is required",
			})
			reqCtx.Handled = true
			return
		}

		result, err := h.RefreshTokenUseCase.RefreshTokens(ctx, request.RefreshToken)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.RefreshTokenResponse{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
		})
	}
}
