package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
)

type ChangePasswordPayload struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type ChangePasswordHandler struct {
	UseCase *usecases.ChangePasswordUseCase
}

func (h *ChangePasswordHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload ChangePasswordPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "invalid request",
			})
			reqCtx.Handled = true
			return
		}

		err := h.UseCase.ChangePassword(ctx, payload.Token, payload.Password)
		if err != nil {
			switch err {
			case constants.ErrInvalidPasswordLength:
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
					"message": "password length invalid",
				})
			case constants.ErrInvalidOrExpiredToken:
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
					"message": "invalid or expired token",
				})
			case constants.ErrAccountNotFound:
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
					"message": "account not found",
				})
			default:
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
					"message": "internal server error",
				})
			}

			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "password updated",
		})
	}
}
