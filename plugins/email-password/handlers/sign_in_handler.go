package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

type SignInHandler struct {
	Logger                       models.Logger
	Config                       *models.Config
	PluginConfig                 types.EmailPasswordPluginConfig
	SignInUseCase                usecases.SignInUseCase
	SendEmailVerificationUseCase usecases.SendEmailVerificationUseCase
}

func (h *SignInHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if reqCtx.UserID != nil && *reqCtx.UserID != "" {
			if sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string); ok && sessionID != "" {
				existingSession, err := h.SignInUseCase.GetSessionByID(ctx, sessionID)
				if err == nil && existingSession != nil && existingSession.ExpiresAt.After(time.Now()) {
					user, err := h.SignInUseCase.GetUserByID(ctx, existingSession.UserID)

					if err != nil {
						h.Logger.Error("failed to get user by ID", "err", err.Error())
						reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
							"message": "failed to get user for existing session",
						})
						reqCtx.Handled = true
						return
					}

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

		var request types.SignInRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		userAgent := r.UserAgent()
		result, err := h.SignInUseCase.SignIn(
			ctx,
			request.Email,
			request.Password,
			request.CallbackURL,
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

		if h.PluginConfig.RequireEmailVerification && h.PluginConfig.SendEmailOnSignIn && !result.User.EmailVerified {
			go func() {
				detachedCtx := context.WithoutCancel(ctx)
				taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
				defer cancel()

				if err := h.SendEmailVerificationUseCase.Send(taskCtx, result.User.Email, request.CallbackURL); err != nil {
					h.Logger.Error("failed to send email", "err", err.Error())
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
