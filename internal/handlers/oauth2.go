package handlers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	oauth2auth "github.com/GoBetterAuth/go-better-auth/internal/auth/oauth2"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

type OAuth2LoginHandler struct {
	Config      *domain.Config
	AuthService *auth.Service
}

func (h *OAuth2LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid path"})
		return
	}
	// Expect path to end with /oauth2/{provider}/login
	// So provider is at len-2
	providerName := parts[len(parts)-2]

	provider, err := h.AuthService.OAuth2ProviderRegistry.Get(providerName)
	if err != nil {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid provider"})
		return
	}

	verifier, challenge, err := oauth2auth.GeneratePKCE()
	if err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "failed to generate pkce"})
		return
	}

	state, err := h.AuthService.TokenService.GenerateToken()
	if err != nil {
		util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "failed to generate state"})
		return
	}

	isSecure := strings.HasPrefix(h.Config.BaseURL, "https")

	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth2_redirect_to",
			Value:    redirectTo,
			Path:     "/",
			HttpOnly: true,
			Secure:   isSecure,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(10 * time.Minute),
		})
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth2_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(10 * time.Minute),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth2_verifier",
		Value:    verifier,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(10 * time.Minute),
	})

	authURL := provider.GetAuthURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	slog.Debug("Auth URL", "url", authURL)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *OAuth2LoginHandler) Handler() http.Handler {
	return Wrap(h)
}

type OAuth2CallbackHandler struct {
	Config      *domain.Config
	AuthService *auth.Service
}

func (h *OAuth2CallbackHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Extract provider from path: /auth/oauth2/{provider}/callback
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid path"})
		return
	}
	// Expect path to end with /oauth2/{provider}/callback
	// So provider is at len-2
	providerName := parts[len(parts)-2]

	code := r.URL.Query().Get("code")
	if code == "" {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "missing code"})
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "missing state"})
		return
	}

	stateCookie, err := r.Cookie("oauth2_state")
	if err != nil || stateCookie.Value != state {
		util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": "invalid state"})
		return
	}

	isSecure := strings.HasPrefix(h.Config.BaseURL, "https")

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth2_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteNoneMode,
	})

	verifierCookie, err := r.Cookie("oauth2_verifier")
	var opts []oauth2.AuthCodeOption
	if err == nil && verifierCookie.Value != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", verifierCookie.Value))

		// Clear verifier cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth2_verifier",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   isSecure,
			SameSite: http.SameSiteNoneMode,
		})
	}

	result, err := h.AuthService.SignInWithOAuth2(r.Context(), providerName, code, opts...)
	if err != nil {
		util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": err.Error()})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.Session.CookieName,
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteNoneMode,
		Expires:  time.Now().Add(h.Config.Session.ExpiresIn),
	})

	target := "/"
	if cookie, err := r.Cookie("oauth2_redirect_to"); err == nil && cookie.Value != "" {
		target = cookie.Value
		if !strings.HasPrefix(target, "/") {
			target = "/"
		}
	}

	// Clear redirect cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth2_redirect_to",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteNoneMode,
	})

	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

func (h *OAuth2CallbackHandler) Handler() http.Handler {
	return Wrap(h)
}
