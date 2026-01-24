package handlers

import (
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/services"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/types"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/usecases"
)

type CallbackHandler struct {
	UseCase *usecases.CallbackUseCase
	HMACKey []byte
}

func NewCallbackHandler(useCase *usecases.CallbackUseCase, hmacKey []byte) *CallbackHandler {
	return &CallbackHandler{UseCase: useCase, HMACKey: hmacKey}
}

func (h *CallbackHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		providerID := r.PathValue("provider")
		if providerID == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": "provider is required",
			})
			reqCtx.Handled = true
			return
		}

		req := &types.CallbackRequest{
			ProviderID: providerID,
			Code:       r.URL.Query().Get("code"),
			State:      r.URL.Query().Get("state"),
			Error:      r.URL.Query().Get("error"),
		}

		stateCookie, err := r.Cookie(constants.CookieState)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": "missing state cookie",
			})
			reqCtx.Handled = true
			return
		}

		validatedState, err := services.ValidateCookie(stateCookie.Value, h.HMACKey, 5*time.Minute)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": "invalid state cookie",
			})
			reqCtx.Handled = true
			return
		}

		if validatedState != req.State {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": "state mismatch",
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

		if reqCtx.UserID != nil && *reqCtx.UserID != "" {
			if sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string); ok && sessionID != "" {
				existingSession, err := h.UseCase.GetSessionByID(ctx, sessionID)
				if err == nil && existingSession != nil && existingSession.ExpiresAt.After(time.Now()) {
					user, _ := h.UseCase.GetUserByID(ctx, existingSession.UserID)
					if user != nil {
						reqCtx.SetJSONResponse(http.StatusOK, &types.CallbackResponse{
							User:    user,
							Session: existingSession,
						})
						return
					}
				}
			}
		}

		result, err := h.UseCase.Callback(ctx, req, &ipAddress, &userAgent)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		// Clear cookies
		http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
			Name:     constants.CookieState,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
			Name:     constants.CookieRedirectTo,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
			Name:     constants.CookieVerifier,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})

		reqCtx.SetUserIDInContext(result.User.ID)
		reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
		reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
		reqCtx.Values[models.ContextAuthSuccess.String()] = true

		var redirectTo string
		if cookie, err := r.Cookie(constants.CookieRedirectTo); err == nil {
			if validated, err := services.ValidateCookie(cookie.Value, h.HMACKey, 5*time.Minute); err == nil {
				redirectTo = validated
			}
		}

		if redirectTo != "" {
			http.Redirect(reqCtx.ResponseWriter, r, redirectTo, http.StatusFound)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.CallbackResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
