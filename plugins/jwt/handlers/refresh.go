package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/usecases"
)

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenHandler struct {
	Logger              models.Logger
	RefreshTokenUseCase usecases.RefreshTokenUseCase
}

func (h *RefreshTokenHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var req RefreshTokenRequest
		if err := util.ParseJSON(r, &req); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}

		if req.RefreshToken == "" {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "refresh_token is required",
			})
			reqCtx.Handled = true
			return
		}

		result, err := h.RefreshTokenUseCase.RefreshTokens(ctx, req.RefreshToken)
		if err != nil {
			h.Logger.Error(err.Error())
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "invalid or expired refresh token",
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, RefreshTokenResponse{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
		})
	}
}
