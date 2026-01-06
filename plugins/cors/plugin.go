package cors

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type CORSPlugin struct {
	config CORSPluginConfig
	logger models.Logger
	ctx    *models.PluginContext
}

func New(config CORSPluginConfig) *CORSPlugin {
	config.ApplyDefaults()
	return &CORSPlugin{config: config}
}

func (p *CORSPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginCORS.String(),
		Version:     "1.0.0",
		Description: "Provides Cross-Origin Resource Sharing (CORS) support",
	}
}

func (p *CORSPlugin) Config() any {
	return p.config
}

func (p *CORSPlugin) Init(ctx *models.PluginContext) error {
	p.logger = ctx.Logger
	p.ctx = ctx

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}
	p.config.ApplyDefaults()

	return nil
}

func (p *CORSPlugin) Close() error {
	return nil
}

// Middleware returns the CORS middleware that should be registered globally
func (p *CORSPlugin) Middleware() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{p.CORSMiddleware()}
}

// CORSMiddleware returns a middleware that handles CORS requests
func (p *CORSPlugin) CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed
			if !p.isOriginAllowed(origin) {
				next.ServeHTTP(w, r)
				return
			}

			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(p.config.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(p.config.AllowedHeaders, ", "))

			if len(p.config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(p.config.ExposedHeaders, ", "))
			}

			if p.config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Max-Age", strconv.FormatInt(int64(p.config.MaxAge/time.Second), 10))

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (p *CORSPlugin) isOriginAllowed(origin string) bool {
	for _, allowed := range p.config.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}
	return false
}

func (p *CORSPlugin) OnConfigUpdate(config *models.Config) error {
	if pluginCfg, ok := config.Plugins[models.PluginCORS.String()]; ok {
		if err := util.ParsePluginConfig(pluginCfg, &p.config); err != nil {
			p.logger.Error("failed to parse cors plugin config on update", "error", err)
			return err
		}
	}

	p.config.ApplyDefaults()

	return nil
}
