package handlers

import (
	"net/http"
	"time"

	internaloauth2 "github.com/GoBetterAuth/go-better-auth/internal/auth/oauth2"
	"github.com/GoBetterAuth/go-better-auth/internal/common"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type OAuth2LoginHandler struct {
	Config  *models.Config
	UseCase internaloauth2.OAuth2UseCase
}

func (h *OAuth2LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	providerName := util.ExtractProviderName(r.URL.Path)
	if providerName == "" {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "oauth2 provider is required"})
		return
	}

	// Use the usecase to prepare the OAuth2 login flow
	// This generates the state, PKCE verifier if needed, and the authorization URL
	loginResult, err := h.UseCase.PrepareOAuth2Login(r.Context(), providerName)
	if err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
		return
	}

	isSecure, sameSite := util.GetCookieOptions(h.Config)

	// Set the OAuth2 state cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth2_state",
		Value:    loginResult.State,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
		Expires:  time.Now().Add(10 * time.Minute),
	})

	// Set the PKCE verifier cookie if PKCE is required
	if loginResult.Verifier != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth2_verifier",
			Value:    *loginResult.Verifier,
			Path:     "/",
			HttpOnly: true,
			Secure:   isSecure,
			SameSite: sameSite,
			Expires:  time.Now().Add(10 * time.Minute),
		})
	}

	// Set the redirect_to cookie if provided in the request
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth2_redirect_to",
			Value:    redirectTo,
			Path:     "/",
			HttpOnly: true,
			Secure:   isSecure,
			SameSite: sameSite,
			Expires:  time.Now().Add(10 * time.Minute),
		})
	}

	// Redirect to the OAuth2 provider's authorization URL
	http.Redirect(w, r, loginResult.AuthURL, http.StatusTemporaryRedirect)
}

func (h *OAuth2LoginHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}

type OAuth2CallbackHandler struct {
	Config  *models.Config
	UseCase internaloauth2.OAuth2UseCase
}

func (h *OAuth2CallbackHandler) Handle(w http.ResponseWriter, r *http.Request) {
	providerName := util.ExtractProviderName(r.URL.Path)

	// Extract the authorization code from the callback
	code := r.URL.Query().Get("code")
	if code == "" {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "missing code"})
		return
	}

	// Extract the state from the callback for CSRF validation
	state := r.URL.Query().Get("state")
	if state == "" {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "missing state"})
		return
	}

	// Verify the state matches what we set in the login flow
	stateCookie, err := r.Cookie("oauth2_state")
	if err != nil || stateCookie.Value != state {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid state"})
		return
	}

	isSecure, sameSite := util.GetCookieOptions(h.Config)

	// Clear the OAuth2 state cookie as it's been used
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth2_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
	})

	// Get the PKCE verifier from the cookie if it exists
	var verifier *string
	if verifierCookie, err := r.Cookie("oauth2_verifier"); err == nil && verifierCookie.Value != "" {
		verifier = &verifierCookie.Value

		// Clear the verifier cookie as it's been used
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth2_verifier",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   isSecure,
			SameSite: sameSite,
		})
	}

	// Use the usecase to handle the OAuth2 callback
	// This validates the state, exchanges the code for tokens, and creates/updates the user
	result, err := h.UseCase.SignInWithOAuth2(r.Context(), providerName, code, state, verifier)
	if err != nil {
		util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": err.Error()})
		return
	}

	// Set the session cookie with the generated session token
	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.Session.CookieName,
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
		Expires:  time.Now().Add(h.Config.Session.ExpiresIn),
	})

	// Set the CSRF cookie if CSRF protection is enabled
	if h.Config.CSRF.Enabled && result.CSRFToken != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     h.Config.CSRF.CookieName,
			Value:    *result.CSRFToken,
			Path:     "/",
			HttpOnly: false,
			Secure:   isSecure,
			SameSite: sameSite,
			MaxAge:   int(h.Config.CSRF.ExpiresIn.Seconds()),
		})
	}

	// Determine the redirect target
	target := "/"
	if cookie, err := r.Cookie("oauth2_redirect_to"); err == nil && cookie.Value != "" {
		redirectTo := cookie.Value
		// Only redirect to trusted origins for security
		if util.IsTrustedRedirect(redirectTo, h.Config.TrustedOrigins.Origins) {
			target = redirectTo
		}
	}

	// Clear the redirect_to cookie as it's been used
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth2_redirect_to",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
	})

	// Redirect to the target URL
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

func (h *OAuth2CallbackHandler) Handler() models.CustomRouteHandler {
	return common.WrapHandler(h)
}
