package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
)

type SignInRequestPayload struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type SignInHandler struct {
	UseCase *usecases.SignInUseCase
}

func (h *SignInHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload SignInRequestPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "invalid request",
			})
			reqCtx.Handled = true
			return
		}

		result, err := h.UseCase.SignIn(
			ctx,
			payload.Email,
			payload.Password,
			&payload.CallbackURL,
		)

		if err != nil {
			switch err {
			case constants.ErrInvalidCredentials:
				reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
					"message": "invalid credentials",
				})
			case constants.ErrEmailNotVerified:
				reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{
					"message": "email not verified",
				})
			default:
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
					"message": "internal server error",
				})
			}

			reqCtx.Handled = true
			return
		}

		reqCtx.SetUserIDInContext(result.User.ID)
		reqCtx.Values["auth_success"] = true

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"user": result.User,
		})
	}
}
