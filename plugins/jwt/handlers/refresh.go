package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/services"
)

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenHandler struct {
	Service services.RefreshTokenService
	Logger  models.Logger
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

		result, err := h.Service.RefreshTokens(ctx, req.RefreshToken)
		if err != nil {
			h.Logger.Error("refresh token failed", "error", err)

			status := http.StatusUnauthorized
			message := "invalid or expired refresh token"

			// Map specific errors to appropriate responses
			switch err.Error() {
			case "invalid refresh token", "refresh token expired":
				status = http.StatusUnauthorized
			case "session expired or invalid":
				status = http.StatusUnauthorized
				message = "session expired"
			default:
				status = http.StatusInternalServerError
				message = "failed to refresh token"
			}

			reqCtx.SetJSONResponse(status, map[string]any{
				"message": message,
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, RefreshTokenResponse{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
		})
		reqCtx.Handled = true
	}
}
