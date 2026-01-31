package gobetterauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/router"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// HookErrorMode defines how the router handles errors from hook handlers
type HookType string

const (
	// HookTypeSync indicates a synchronous hook that runs in the main request flow
	HookTypeSync HookType = "sync"
	// HookTypeAsync indicates an asynchronous hook that runs in a background goroutine
	HookTypeAsync HookType = "async"
)

// HookErrorMode defines how the router handles errors from hook handlers
type HookErrorMode string

const (
	// HookErrorModeContinue logs errors but continues to next hook (default)
	HookErrorModeContinue HookErrorMode = "error-log-continue"
	// HookErrorModeFailFast logs error and sets ctx.Handled=true to skip remaining hooks in current stage
	HookErrorModeFailFast HookErrorMode = "error-log-fail-fast"
	// HookErrorModeSilent silently ignores errors without logging
	HookErrorModeSilent HookErrorMode = "error-silent"
)

// RouterOptions contains configuration options for the Router
type RouterOptions struct {
	// AsyncHookTimeout is the timeout for async hook execution (default: 30 seconds)
	// If a hook takes longer than this, it will be cancelled
	AsyncHookTimeout time.Duration
	// HookErrorMode defines how errors from hooks are handled (default: HookErrorModeContinue)
	// Controls whether errors cause early exit, silent ignoring, or just logging
	HookErrorMode HookErrorMode
}

// DefaultRouterOptions returns router options with sensible defaults
func DefaultRouterOptions() *RouterOptions {
	return &RouterOptions{
		AsyncHookTimeout: 30 * time.Second,
		HookErrorMode:    HookErrorModeContinue,
	}
}

type routeEntry struct {
	Method   string
	Segments []string // path split by "/", e.g. ["oauth2", "callback", "{provider}"]
	Metadata map[string]any
}

// Router wraps chi.Router and manages hooks for the request lifecycle
type Router struct {
	config        *models.Config
	logger        models.Logger
	basePath      string
	router        chi.Router
	hooks         []models.Hook
	opts          *RouterOptions
	routeMetadata map[string]map[string]any
	routeEntries  []routeEntry
}

// NewRouter creates a new Router with Chi as the underlying router
// opts can be nil to use default options
func NewRouter(config *models.Config, logger models.Logger, opts *RouterOptions) *Router {
	if opts == nil {
		opts = DefaultRouterOptions()
	}

	r := chi.NewRouter()

	// Set default NotFound handler
	r.NotFound(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		util.JSONResponse(w, http.StatusNotFound, map[string]any{"message": "Not Found"})
	}))

	// Set default MethodNotAllowed handler
	r.MethodNotAllowed(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}))

	util.ValidateTrustedHeadersAndProxies(
		logger,
		config.Security.TrustedHeaders,
		config.Security.TrustedProxies,
	)

	return &Router{
		config:        config,
		logger:        logger,
		basePath:      config.BasePath,
		router:        r,
		hooks:         []models.Hook{},
		opts:          opts,
		routeMetadata: make(map[string]map[string]any),
	}
}

// Get returns the underlying chi.Router for direct access
func (r *Router) Get() chi.Router {
	return r.router
}

// RegisterMiddleware registers global middleware with Chi
func (r *Router) RegisterMiddleware(middleware ...func(http.Handler) http.Handler) {
	for _, mw := range middleware {
		r.router.Use(mw)
	}
}

// RegisterRoute registers a single route with Chi
func (r *Router) RegisterRoute(route models.Route) {
	r.registerRouteWithPrefix(r.basePath, route)
}

// RegisterCustomRoute registers a custom route without the basePath prefix
// This is useful for application routes that should not be under the auth basePath
func (r *Router) RegisterCustomRoute(route models.Route) {
	r.registerRouteWithPrefix("", route)
}

// RegisterRoutes registers multiple routes with an optional base path
func (r *Router) RegisterRoutes(routes []models.Route) {
	for _, route := range routes {
		r.registerRouteWithPrefix(r.basePath, route)
	}
}

// RegisterCustomRoutes registers multiple custom routes without the basePath prefix
// This is useful for application routes that should not be under the auth basePath
func (r *Router) RegisterCustomRoutes(routes []models.Route) {
	for _, route := range routes {
		r.registerRouteWithPrefix("", route)
	}
}

// SetRouteMetadataFromConfig sets route metadata mappings from RouteMappings.
// This populates the internal metadata map used to assign ctx.Route.Metadata["plugins"] during request handling.
// Supports both static and dynamic (parameterized) routes.
// Format: routeMetadata["METHOD:path"] = {"plugins": ["plugin1", "plugin2"], ...}
// Examples:
//   - Static route: "GET:/me" -> plugins for GET /me
//   - Dynamic route: "POST:/oauth2/callback/{provider}" -> plugins for POST /oauth2/callback/{provider} (matches any provider value)
//   - Multi-param: "GET:/users/{id}/posts/{postId}" -> plugins for any GET request with that pattern
//
// Dynamic routes use {paramName} syntax and match any actual parameter value at that position.
// At request time, the router first tries exact path matching, then falls back to pattern matching.
func (r *Router) SetRouteMetadataFromConfig(routeMetadata map[string]map[string]any) {
	r.routeMetadata = make(map[string]map[string]any)
	r.routeEntries = make([]routeEntry, 0, len(routeMetadata))

	for key, metadata := range routeMetadata {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}

		method := parts[0]
		path := "/" + strings.Trim(parts[1], "/")

		if r.basePath != "" {
			base := "/" + strings.Trim(r.basePath, "/")
			if !strings.HasPrefix(path, base) {
				path = base + path
			}
		}

		fullKey := method + ":" + path
		r.routeMetadata[fullKey] = metadata

		segments := strings.Split(strings.Trim(path, "/"), "/")
		r.routeEntries = append(r.routeEntries, routeEntry{
			Method:   method,
			Segments: segments,
			Metadata: metadata,
		})
	}
}

// registerRouteWithPrefix registers a route with Chi, applying any route-scoped middleware
func (r *Router) registerRouteWithPrefix(basePath string, route models.Route) {
	path := basePath + route.Path
	handler := route.Handler

	// Apply route-scoped middleware if present
	if len(route.Middleware) > 0 {
		for i := len(route.Middleware) - 1; i >= 0; i-- {
			handler = route.Middleware[i](handler)
		}
	}

	// Store route metadata if provided (will be assigned to ctx.Route during request handling)
	if route.Metadata != nil {
		metadataKey := route.Method + ":" + path
		r.routeMetadata[metadataKey] = route.Metadata
	}

	// Register with Chi
	method := route.Method
	switch method {
	case http.MethodGet:
		r.router.Get(path, handler.ServeHTTP)
	case http.MethodPost:
		r.router.Post(path, handler.ServeHTTP)
	case http.MethodPut:
		r.router.Put(path, handler.ServeHTTP)
	case http.MethodPatch:
		r.router.Patch(path, handler.ServeHTTP)
	case http.MethodDelete:
		r.router.Delete(path, handler.ServeHTTP)
	case http.MethodHead:
		r.router.Head(path, handler.ServeHTTP)
	case http.MethodOptions:
		r.router.Options(path, handler.ServeHTTP)
	default:
		r.router.MethodFunc(method, path, handler.ServeHTTP)
	}
}

// RegisterHooks registers multiple hooks
func (r *Router) RegisterHooks(hooks []models.Hook) {
	r.hooks = append(r.hooks, hooks...)
	r.sortHooks()
}

// RegisterHook registers a single hook
func (r *Router) RegisterHook(hook models.Hook) {
	r.hooks = append(r.hooks, hook)
	r.sortHooks()
}

// sortHooks sorts hooks by stage, then by Order.
// This allows controlling execution order across plugins using the Order field.
func (r *Router) sortHooks() {
	slices.SortStableFunc(r.hooks, func(a, b models.Hook) int {
		// First, sort by stage
		if a.Stage != b.Stage {
			if a.Stage < b.Stage {
				return -1
			}
			return 1
		}
		// Within same stage, sort by Order
		if a.Order != b.Order {
			if a.Order < b.Order {
				return -1
			}
			return 1
		}
		return 0
	})
}

func (r *Router) runHooks(stage models.HookStage, ctx *models.RequestContext) {
	for _, hook := range r.hooks {
		if hook.Stage != stage {
			continue
		}

		// Skip hooks not in route metadata
		if hook.PluginID != "" {
			if ctx.Route == nil {
				continue
			}

			pluginIDs, ok := ctx.Route.Metadata["plugins"].([]string)
			if !ok || !contains(pluginIDs, hook.PluginID) {
				continue
			}
		}

		if hook.Matcher != nil && !hook.Matcher(ctx) {
			continue
		}

		// Async execution
		if hook.Async {
			go func(h models.Hook, originalCtx *models.RequestContext) {
				defer r.recoverFromPanic(string(HookTypeAsync), h.PluginID, stage)

				clonedCtx := util.CloneRequestContext(originalCtx)

				asyncCtx, cancel := context.WithTimeout(context.Background(), r.opts.AsyncHookTimeout)
				defer cancel()
				clonedCtx.Request = clonedCtx.Request.WithContext(asyncCtx)

				if err := h.Handler(clonedCtx); err != nil {
					if asyncCtx.Err() == context.DeadlineExceeded {
						r.logger.Error("Async hook timeout",
							"stage", stage,
							"plugin_id", h.PluginID,
							"timeout", r.opts.AsyncHookTimeout)
					} else {
						r.handleHookError(h.PluginID, stage, err, true)
					}
				}
			}(hook, ctx)
			continue
		}

		// Synchronous execution
		func() {
			defer r.recoverFromPanic(string(HookTypeSync), hook.PluginID, stage)
			if err := hook.Handler(ctx); err != nil {
				r.handleHookError(hook.PluginID, stage, err, false)
				if r.opts.HookErrorMode == HookErrorModeFailFast {
					ctx.Handled = true
				}
			}
		}()

		if ctx.Handled {
			break
		}
	}
}

func (r *Router) recoverFromPanic(hookType, pluginID string, stage models.HookStage) {
	if err := recover(); err != nil {
		// Capture stack trace
		stackTrace := string(debug.Stack())

		// Log panic with context
		r.logger.Error(
			fmt.Sprintf("Panic in %s", hookType),
			"plugin_id", pluginID,
			"stage", stage,
			"panic", fmt.Sprintf("%v", err),
			"stack", stackTrace,
		)
	}
}

func (r *Router) handleHookError(pluginID string, stage models.HookStage, err error, isAsync bool) {
	switch r.opts.HookErrorMode {
	case HookErrorModeFailFast, HookErrorModeContinue:
		hookType := string(HookTypeSync)
		if isAsync {
			hookType = string(HookTypeAsync)
		}
		r.logger.Error(
			fmt.Sprintf("Hook handler error (%s)", hookType),
			"stage", stage,
			"plugin_id", pluginID,
			"error", err,
		)
	case HookErrorModeSilent:
		// Silently ignore errors
	}
}

func contains(slice []string, value string) bool {
	return slices.Contains(slice, value)
}

func matchRoutePath(requestPath, pattern string) bool {
	normalize := func(p string) []string {
		p = strings.Trim(p, "/")
		if p == "" {
			return nil
		}
		return strings.FieldsFunc(p, func(r rune) bool { return r == '/' })
	}

	reqSegs := normalize(requestPath)
	patSegs := normalize(pattern)

	if len(reqSegs) != len(patSegs) {
		return false
	}

	for i := range reqSegs {
		if strings.HasPrefix(patSegs[i], "{") && strings.HasSuffix(patSegs[i], "}") {
			continue
		}
		if patSegs[i] != reqSegs[i] {
			return false
		}
	}
	return true
}

// getRouteMetadata looks up route metadata for a given request method and path.
// Returns the metadata, the matched pattern (for dynamic paths), and whether a match was found.
func (r *Router) getRouteMetadata(method, path string) (map[string]any, string, bool) {
	// Try exact match first
	exactKey := method + ":" + path
	if metadata, exists := r.routeMetadata[exactKey]; exists {
		return metadata, exactKey, true
	}

	// Pattern matching for dynamic routes
	for key, metadata := range r.routeMetadata {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		storedMethod := parts[0]
		pattern := parts[1]

		if storedMethod != method {
			continue
		}

		if matchRoutePath(path, pattern) {
			return metadata, key, true
		}
	}

	return nil, "", false
}

// Handler returns the configured HTTP handler - the Router with hook middleware
func (r *Router) Handler() http.Handler {
	return r
}

// ServeHTTP implements http.Handler for testing and direct use
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Wrap response writer to defer writes
	wrappedWriter := &router.DeferredResponseWriter{
		Wrapped: w,
		Logger:  r.logger,
	}

	clientIP, err := util.ExtractClientIP(
		r.logger,
		req,
		r.config.Security.TrustedHeaders,
		r.config.Security.TrustedProxies,
	)
	if err != nil {
		r.logger.Error("Failed to extract client IP", "error", err, "remoteAddr", req.RemoteAddr)
		clientIP = net.ParseIP("0.0.0.0") // Safe fallback IP
	}

	// Create request context
	reqCtx := &models.RequestContext{
		Request:         req,
		ResponseWriter:  wrappedWriter,
		Path:            req.URL.Path,
		Method:          req.Method,
		Headers:         req.Header,
		ClientIP:        clientIP.String(),
		Values:          make(map[string]any),
		ResponseHeaders: make(http.Header),
		Handled:         false,
	}

	// Associate request context with wrapped writer
	wrappedWriter.SetRequestContext(reqCtx)

	// Lookup route metadata
	metadata, pattern, exists := r.getRouteMetadata(req.Method, req.URL.Path)
	if exists {
		if metadata["_pattern"] == nil {
			metadata["_pattern"] = pattern
		}
		reqCtx.Route = &models.Route{
			Method:   req.Method,
			Path:     req.URL.Path,
			Metadata: metadata,
		}
	} else {
		reqCtx.Route = &models.Route{
			Method:   req.Method,
			Path:     req.URL.Path,
			Metadata: make(map[string]any),
		}
	}

	// Store context in request
	reqWithCtx := req.WithContext(models.NewContextWithRequestContext(req.Context(), reqCtx))

	if req.Method == http.MethodOptions {
		r.applyCORS(req, wrappedWriter)
		wrappedWriter.WriteHeader(http.StatusOK)
		wrappedWriter.Flush()
		return
	}

	// Stage 1: OnRequest hooks
	r.runHooks(models.HookOnRequest, reqCtx)
	if reqCtx.Handled {
		r.finalizeResponse(reqCtx, wrappedWriter)
		return
	}

	// Stage 2: Before hooks
	r.runHooks(models.HookBefore, reqCtx)
	if reqCtx.Handled {
		r.finalizeResponse(reqCtx, wrappedWriter)
		return
	}

	// Stage 3: Route handler (via Chi)
	r.router.ServeHTTP(wrappedWriter, reqWithCtx)

	// Stage 4: After hooks
	r.runHooks(models.HookAfter, reqCtx)
	if reqCtx.Handled {
		r.finalizeResponse(reqCtx, wrappedWriter)
		return
	}

	// Stage 5: OnResponse hooks
	r.runHooks(models.HookOnResponse, reqCtx)

	// Flush deferred writes or captured response
	r.finalizeResponse(reqCtx, wrappedWriter)
}

func (r *Router) finalizeResponse(
	ctx *models.RequestContext,
	w *router.DeferredResponseWriter,
) {
	// CORS must ALWAYS be applied (success or failure)
	r.applyCORS(ctx.Request, w)

	if ctx.ResponseReady {
		w.OverrideWithContext(ctx)
	}

	w.Flush()
}

func (r *Router) applyCORS(
	req *http.Request,
	w http.ResponseWriter,
) {
	corsConfig := r.config.Security.CORS

	origin := req.Header.Get("Origin")
	if origin == "" {
		return
	}

	// Required for caches when echoing origin
	w.Header().Add("Vary", "Origin")

	// Spec violation guard: credentials + wildcard
	if corsConfig.AllowCredentials && slices.Contains(corsConfig.AllowedOrigins, "*") {
		return
	}

	// Origin not allowed
	if !isOriginAllowed(origin, corsConfig.AllowedOrigins) {
		if req.Method == http.MethodOptions {
			w.Header().Set(
				"Access-Control-Allow-Methods",
				strings.Join(corsConfig.AllowedMethods, ", "),
			)
			w.Header().Set(
				"Access-Control-Allow-Headers",
				strings.Join(corsConfig.AllowedHeaders, ", "),
			)
			w.Header().Set(
				"Access-Control-Max-Age",
				strconv.FormatInt(int64(corsConfig.MaxAge/time.Second), 10),
			)
		}
		return
	}

	// ---- Allowed origin ----
	w.Header().Set("Access-Control-Allow-Origin", origin)

	if corsConfig.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if len(corsConfig.ExposedHeaders) > 0 {
		w.Header().Set(
			"Access-Control-Expose-Headers",
			strings.Join(corsConfig.ExposedHeaders, ", "),
		)
	}

	if req.Method == http.MethodOptions {
		w.Header().Set(
			"Access-Control-Allow-Methods",
			strings.Join(corsConfig.AllowedMethods, ", "),
		)
		w.Header().Set(
			"Access-Control-Allow-Headers",
			strings.Join(corsConfig.AllowedHeaders, ", "),
		)
		w.Header().Set(
			"Access-Control-Max-Age",
			strconv.FormatInt(int64(corsConfig.MaxAge/time.Second), 10),
		)
	}
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		if strings.HasPrefix(allowed, "*.") &&
			strings.HasSuffix(origin, strings.TrimPrefix(allowed, "*")) {
			return true
		}
	}
	return false
}
