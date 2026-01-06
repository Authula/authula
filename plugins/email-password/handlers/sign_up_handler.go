package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
)

type SignUpRequestPayload struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Image       string `json:"image,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type SignUpHandler struct {
	UseCase *usecases.SignUpUseCase
}

func (h *SignUpHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload SignUpRequestPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "invalid request",
			})
			reqCtx.Handled = true
			return
		}

		result, err := h.UseCase.SignUp(
			ctx,
			payload.Name,
			payload.Email,
			payload.Password,
			&payload.Image,
			&payload.CallbackURL,
		)
		if err != nil {
			switch err {
			case constants.ErrSignUpDisabled:
				reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{
					"message": "sign up is disabled",
				})
			case constants.ErrEmailAlreadyExists:
				reqCtx.SetJSONResponse(http.StatusConflict, map[string]any{
					"message": "email already registered",
				})
			case constants.ErrPasswordLengthInvalid:
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
					"message": "password length invalid",
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
		if h.UseCase.PluginConfig.AutoSignIn && !h.UseCase.PluginConfig.RequireEmailVerification {
			reqCtx.Values["auth_success"] = true
		}

		reqCtx.SetJSONResponse(http.StatusCreated, map[string]any{
			"user": result.User,
		})
	}
}
