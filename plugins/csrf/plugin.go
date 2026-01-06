package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// TODO: Phase 2
// AddInsecureBypassPattern() for webhooks/integrations
// Route-level override configuration
// Per-plugin enforcement modes
// Wildcard/regex origin matching

type CSRFPlugin struct {
	globalConfig *models.Config
	pluginConfig CSRFPluginConfig
	ctx          *models.PluginContext
	logger       models.Logger
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

	// Initialize header protection if enabled
	if p.pluginConfig.EnableHeaderProtection {
		if err := p.initializeHeaderProtection(); err != nil {
			return err
		}
	}

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

// generateToken creates a new CSRF token using 32 bytes of cryptographic randomness
func (p *CSRFPlugin) generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// validateCSRFToken validates the CSRF token for unsafe methods on protected endpoints
func (p *CSRFPlugin) validateCSRFToken(ctx *models.RequestContext) error {
	r := ctx.Request
	cookie, err := r.Cookie(p.pluginConfig.CookieName)
	if err != nil {
		if err := ctx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "missing csrf cookie"}); err != nil {
			return err
		}
		ctx.Handled = true
		return nil
	}

	// Get token from header or form
	headerToken := r.Header.Get(p.pluginConfig.HeaderName)
	if headerToken == "" {
		headerToken = r.FormValue(p.pluginConfig.CookieName)
	}

	// Compare cookie value with header/form value
	if headerToken != cookie.Value {
		if err := ctx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "invalid csrf token"}); err != nil {
			return err
		}
		ctx.Handled = true
		return nil
	}

	return nil
}

// Middleware returns a CSRF protection middleware that users can add to custom routes.
func (p *CSRFPlugin) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(models.ContextUserID).(string)
			if !ok || userID == "" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				_, err := r.Cookie(p.pluginConfig.CookieName)
				if err == http.ErrNoCookie {
					token := p.generateToken()
					p.setCSRFCookie(w, r, token)
				}
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(p.pluginConfig.CookieName)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				util.JSONResponse(w, http.StatusForbidden, map[string]string{"message": "missing csrf cookie"})
				return
			}

			headerToken := r.Header.Get(p.pluginConfig.HeaderName)
			if headerToken == "" {
				headerToken = r.FormValue(p.pluginConfig.CookieName)
			}

			if headerToken != cookie.Value {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				util.JSONResponse(w, http.StatusForbidden, map[string]string{"message": "invalid csrf token"})
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
func (p *CSRFPlugin) setCSRFCookie(w http.ResponseWriter, r *http.Request, token string) {
	// Determine SameSite mode from config
	var sf http.SameSite
	switch p.pluginConfig.SameSite {
	case "strict":
		sf = http.SameSiteStrictMode
	case "none":
		sf = http.SameSiteNoneMode
	case "lax":
		sf = http.SameSiteLaxMode
	default:
		sf = http.SameSiteLaxMode
	}

	// Set Secure flag only for HTTPS requests (production),
	// allow HTTP for development on localhost
	secure := r.TLS != nil || r.URL.Scheme == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     p.pluginConfig.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,  // Hardcoded: Required for Double-Submit Cookie pattern
		Secure:   secure, // Conditional: true for HTTPS (production), false for HTTP (development)
		SameSite: sf,
		MaxAge:   int(p.pluginConfig.MaxAge.Seconds()),
	})

	// Set token in response header so client can read it
	w.Header().Set(p.pluginConfig.HeaderName, token)
}
