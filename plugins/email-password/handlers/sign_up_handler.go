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

type SignUpRequestPayload struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Image       string `json:"image,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type SignUpHandler struct {
	Logger                       models.Logger
	Config                       *models.Config
	PluginConfig                 types.EmailPasswordPluginConfig
	SignUpUseCase                *usecases.SignUpUseCase
	SendEmailVerificationUseCase *usecases.SendEmailVerificationUseCase
}

func (h *SignUpHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload SignUpRequestPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}

		userAgent := r.UserAgent()
		result, err := h.SignUpUseCase.SignUp(
			ctx,
			payload.Name,
			payload.Email,
			payload.Password,
			&payload.Image,
			&payload.CallbackURL,
			&reqCtx.ClientIP,
			&userAgent,
		)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		if h.PluginConfig.RequireEmailVerification && h.PluginConfig.SendEmailOnSignUp {
			go func() {
				detachedCtx := context.WithoutCancel(ctx)
				taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
				defer cancel()

				if err := h.SendEmailVerificationUseCase.Send(taskCtx, payload.Email, &payload.CallbackURL); err != nil {
					h.Logger.Error("failed to send email", "err", err)
				}
			}()
		}

		reqCtx.SetUserIDInContext(result.User.ID)
		if h.PluginConfig.AutoSignIn {
			reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
			reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
			reqCtx.Values[models.ContextAuthSuccess.String()] = true
		}

		reqCtx.SetJSONResponse(http.StatusCreated, types.SignUpResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
