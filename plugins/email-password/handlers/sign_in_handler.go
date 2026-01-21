package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/types"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
)

type SignInRequestPayload struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type SignInHandler struct {
	Logger                       models.Logger
	PluginConfig                 types.EmailPasswordPluginConfig
	SignInUseCase                *usecases.SignInUseCase
	SendEmailVerificationUseCase *usecases.SendEmailVerificationUseCase
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

		ipAddress := util.ExtractClientIP(
			r.Header.Get("X-Forwarded-For"),
			r.Header.Get("X-Real-IP"),
			r.RemoteAddr,
		)
		userAgent := r.UserAgent()

		result, err := h.SignInUseCase.SignIn(
			ctx,
			payload.Email,
			payload.Password,
			&payload.CallbackURL,
			&ipAddress,
			&userAgent,
		)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		if h.PluginConfig.RequireEmailVerification && !result.User.EmailVerified && h.PluginConfig.SendEmailOnSignIn {
			go func() {
				detachedCtx := context.WithoutCancel(ctx)
				taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
				defer cancel()

				if err := h.SendEmailVerificationUseCase.Send(taskCtx, result.User.Email, &payload.CallbackURL); err != nil {
					h.Logger.Error("failed to send email", "err", err)
				}
			}()
		}

		reqCtx.SetUserIDInContext(result.User.ID)
		reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken

		reqCtx.SetJSONResponse(http.StatusOK, types.SignInResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
