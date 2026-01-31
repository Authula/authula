package csrf

import (
	"fmt"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

// TODO: Phase 2
// AddInsecureBypassPattern() for webhooks/integrations
// Route-level override configuration
// Per-plugin enforcement modes
// Wildcard/regex origin matching

type CSRFPlugin struct {
	globalConfig *models.Config
	pluginConfig CSRFPluginConfig
	logger       models.Logger
	tokenService rootservices.TokenService
	ctx          *models.PluginContext
	cop          *http.CrossOriginProtection
}

func New(config CSRFPluginConfig) *CSRFPlugin {
	config.ApplyDefaults()
	return &CSRFPlugin{pluginConfig: config}
}

func (p *CSRFPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginCSRF.String(),
		Version:     "1.0.0",
		Description: "Provides CSRF protection via request lifecycle hooks",
	}
}

func (p *CSRFPlugin) Config() any {
	return p.pluginConfig
}

func (p *CSRFPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger
	globalConfig := ctx.GetConfig()
	p.globalConfig = globalConfig

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.pluginConfig); err != nil {
		return err
	}

	p.pluginConfig.ApplyDefaults()

	if p.pluginConfig.EnableHeaderProtection {
		if err := p.initializeHeaderProtection(); err != nil {
			return err
		}
	}

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("token service not available in service registry")
	}
	p.tokenService = tokenService

	return nil
}

func (p *CSRFPlugin) Close() error {
	return nil
}

func (p *CSRFPlugin) OnConfigUpdate(config *models.Config) error {
	if err := util.LoadPluginConfig(config, p.Metadata().ID, &p.pluginConfig); err != nil {
		p.logger.Error("failed to parse csrf plugin config on update", "error", err)
		return err
	}

	p.pluginConfig.ApplyDefaults()

	// Reinitialize header protection if enabled
	if p.pluginConfig.EnableHeaderProtection {
		if err := p.initializeHeaderProtection(); err != nil {
			return err
		}
	} else {
		p.cop = nil
	}

	return nil
}

// initializeHeaderProtection initializes the CrossOriginProtection with trusted origins
// and sets up the custom deny handler. This method is used by both Init and OnConfigUpdate.
func (p *CSRFPlugin) initializeHeaderProtection() error {
	if err := util.ValidateTrustedOrigins(p.globalConfig.Security.TrustedOrigins); err != nil {
		p.logger.Error("invalid trusted origins configuration", "error", err)
		return err
	}

	// Initialize CrossOriginProtection with trusted origins
	p.cop = http.NewCrossOriginProtection()

	for _, origin := range p.globalConfig.Security.TrustedOrigins {
		if err := p.cop.AddTrustedOrigin(origin); err != nil {
			p.logger.Error("failed to add trusted origin", "origin", origin, "error", err)
			return err
		}
	}

	// Set custom deny handler to match our error format
	p.cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.logger != nil {
			p.logger.Debug("cross-origin request rejected", "origin", r.Header.Get("Origin"), "host", r.Host)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		util.JSONResponse(w, http.StatusForbidden, map[string]string{
			"error": "csrf validation failed",
		})
	}))

	return nil
}

// Hooks implements models.PluginWithHooks to provide CSRF protection via hooks
func (p *CSRFPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

// validateHeaderProtection validates cross-origin requests using Go 1.25's CrossOriginProtection
// Returns nil if header validation passes or is disabled, error if validation fails
func (p *CSRFPlugin) validateHeaderProtection(r *http.Request) error {
	// If header protection is disabled, skip this check
	if !p.pluginConfig.EnableHeaderProtection || p.cop == nil {
		return nil
	}

	// Perform cross-origin protection check
	// This checks Sec-Fetch-Site, Origin, and Host headers
	if err := p.cop.Check(r); err != nil {
		if p.logger != nil {
			p.logger.Debug("header-based csrf check failed", "error", err)
		}
		return err
	}

	return nil
}

// validateCSRFToken validates the CSRF token for unsafe methods on protected endpoints
func (p *CSRFPlugin) validateCSRFToken(reqCtx *models.RequestContext) error {
	r := reqCtx.Request
	cookie, err := r.Cookie(p.pluginConfig.CookieName)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "missing csrf cookie"})
		reqCtx.Handled = true
		return nil
	}

	headerToken := r.Header.Get(p.pluginConfig.HeaderName)
	if headerToken == "" {
		reqCtx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "missing csrf token in header"})
		reqCtx.Handled = true
		return nil
	}

	if headerToken != cookie.Value {
		reqCtx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "invalid csrf token"})
		reqCtx.Handled = true
		return nil
	}

	return nil
}

// Middleware returns a CSRF protection middleware that users can add to custom routes.
func (p *CSRFPlugin) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			reqCtx, ok := models.GetRequestContext(ctx)
			if !ok || reqCtx.UserID == nil || *reqCtx.UserID == "" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				_, err := r.Cookie(p.pluginConfig.CookieName)
				if err == http.ErrNoCookie {
					token, err := p.tokenService.Generate()
					if err != nil {
						reqCtx.SetJSONResponse(
							http.StatusInternalServerError,
							map[string]string{"message": "failed to generate csrf token"},
						)
						reqCtx.Handled = true
						return
					}
					p.setCSRFCookie(reqCtx, token)
				}
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(p.pluginConfig.CookieName)
			if err != nil {
				reqCtx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "missing csrf cookie"})
				reqCtx.Handled = true
				return
			}

			headerToken := r.Header.Get(p.pluginConfig.HeaderName)
			if headerToken == "" {
				reqCtx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "missing csrf token in header"})
				reqCtx.Handled = true
				return
			}

			if headerToken != cookie.Value {
				reqCtx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "invalid csrf token"})
				reqCtx.Handled = true
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// setCSRFCookie sets the CSRF token cookie with hardcoded security settings.
// Both HttpOnly and Secure are hardcoded to ensure the Double-Submit Cookie pattern works correctly:
// - HttpOnly=false: Allows JavaScript to read the cookie value
// - Secure: Set to true only for HTTPS requests (allows development over HTTP on localhost)
// Also sets the token in a response header so the client can read and use it.
func (p *CSRFPlugin) setCSRFCookie(reqCtx *models.RequestContext, token string) {
	var samesite http.SameSite
	switch p.pluginConfig.SameSite {
	case "strict":
		samesite = http.SameSiteStrictMode
	case "none":
		samesite = http.SameSiteNoneMode
	case "lax":
		samesite = http.SameSiteLaxMode
	default:
		samesite = http.SameSiteLaxMode
	}

	// Secure must be true for HTTPS requests or when SameSite=None
	secure := reqCtx.Request.URL.Scheme == "https" || samesite == http.SameSiteNoneMode || p.pluginConfig.Secure

	http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
		Name:     p.pluginConfig.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Hardcoded: Required for Double-Submit Cookie pattern
		Secure:   secure,
		SameSite: samesite,
		MaxAge:   int(p.pluginConfig.MaxAge.Seconds()),
	})

	// Set token in response header so client can read it
	reqCtx.ResponseWriter.Header().Set(p.pluginConfig.HeaderName, token)
}
