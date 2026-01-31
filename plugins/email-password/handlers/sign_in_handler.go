package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/usecases"
)

type SignInRequestPayload struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type SignInHandler struct {
	Logger                       models.Logger
	Config                       *models.Config
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
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}

		if reqCtx.UserID != nil && *reqCtx.UserID != "" {
			if sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string); ok && sessionID != "" {
				existingSession, err := h.SignInUseCase.GetSessionByID(ctx, sessionID)
				if err == nil && existingSession != nil && existingSession.ExpiresAt.After(time.Now()) {
					user, _ := h.SignInUseCase.GetUserByID(ctx, existingSession.UserID)
					if user != nil {
						reqCtx.Values[models.ContextAuthIdempotentSkipTokensMint.String()] = true

						reqCtx.SetJSONResponse(http.StatusOK, types.SignInResponse{
							User:    user,
							Session: existingSession,
						})
						return
					}
				}
			}
		}

		userAgent := r.UserAgent()
		result, err := h.SignInUseCase.SignIn(
			ctx,
			payload.Email,
			payload.Password,
			&payload.CallbackURL,
			&reqCtx.ClientIP,
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
		reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
		reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
		reqCtx.Values[models.ContextAuthSuccess.String()] = true

		reqCtx.SetJSONResponse(http.StatusOK, types.SignInResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
