package gobetterauth

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/GoBetterAuth/go-better-auth/internal/router"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
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
	// MaxBufferSize is the maximum amount of data to buffer in response body (default: 10MB)
	// When exceeded, buffering is bypassed for that response
	MaxBufferSize int
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
		MaxBufferSize:    10 * 1024 * 1024, // 10MB
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
func NewRouter(logger models.Logger, basePath string, opts *RouterOptions) *Router {
	if opts == nil {
		opts = DefaultRouterOptions()
	}

	r := chi.NewRouter()

	// Set default NotFound handler
	r.NotFound(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))

	// Set default MethodNotAllowed handler
	r.MethodNotAllowed(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}))

	return &Router{
		logger:        logger,
		basePath:      basePath,
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
	r.logger.Debug("runHooks start",
		"stage", stage,
		"path", ctx.Path,
		"method", ctx.Method,
		"total_hooks", len(r.hooks))

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

		matchedPattern := "<unknown>"
		if ctx.Route != nil && ctx.Route.Metadata != nil {
			if val, ok := ctx.Route.Metadata["_pattern"]; ok {
				if s, ok2 := val.(string); ok2 {
					matchedPattern = s
				}
			}
		}

		r.logger.Debug("Executing hook",
			"stage", stage,
			"plugin_id", hook.PluginID,
			"async", hook.Async,
			"path", ctx.Path,
			"matched_pattern", matchedPattern)

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
			r.logger.Debug("Executed hook",
				"stage", stage,
				"plugin_id", hook.PluginID,
				"path", ctx.Path,
				"matched_pattern", matchedPattern)
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
	r.logger.Debug("getRouteMetadata", "method", method, "path", path)

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
	// Start timing the request
	requestStart := time.Now()

	// Check if this is a streaming response that should skip buffering
	skipBuffer := r.isStreamingResponse(w.Header())

	// Wrap response writer to defer writes
	wrappedWriter := &router.DeferredResponseWriter{
		Wrapped:       w,
		Logger:        r.logger,
		MaxBufferSize: r.opts.MaxBufferSize,
		SkipBuffer:    skipBuffer,
	}

	// Create request context
	ctx := &models.RequestContext{
		Request:         req,
		ResponseWriter:  wrappedWriter,
		Path:            req.URL.Path,
		Method:          req.Method,
		Headers:         req.Header,
		Values:          make(map[string]any),
		ResponseHeaders: make(http.Header),
		Handled:         false,
	}

	wrappedWriter.SetRequestContext(ctx)

	// Allow manual override of buffer skipping via context
	if skipVal, ok := ctx.Values["skipBuffer"]; ok {
		if skipBool, ok := skipVal.(bool); ok && skipBool {
			wrappedWriter.SkipBuffer = true
		}
	}

	metadata, pattern, exists := r.getRouteMetadata(req.Method, req.URL.Path)
	if exists {
		if metadata["_pattern"] == nil {
			metadata["_pattern"] = pattern
		}
		ctx.Route = &models.Route{
			Method:   req.Method,
			Path:     req.URL.Path,
			Metadata: metadata,
		}
		r.logger.Debug("Route metadata matched",
			"path", req.URL.Path,
			"pattern", pattern,
			"plugins", metadata["plugins"])
	} else {
		r.logger.Debug("No route metadata matched", "path", req.URL.Path)
		ctx.Route = &models.Route{
			Method:   req.Method,
			Path:     req.URL.Path,
			Metadata: make(map[string]any),
		}
	}

	// Store context in request
	reqWithCtx := req.WithContext(models.NewContextWithRequestContext(req.Context(), ctx))

	// Stage 1: OnRequest hooks
	onRequestStart := time.Now()
	r.runHooks(models.HookOnRequest, ctx)
	onRequestDuration := time.Since(onRequestStart).Milliseconds()

	if ctx.Handled {
		r.finalizeResponse(ctx, wrappedWriter)
		totalDuration := time.Since(requestStart).Milliseconds()
		r.logger.Debug("request completed (early exit)",
			"method", req.Method, "path", req.URL.Path,
			"status", ctx.ResponseStatus,
			"total_ms", totalDuration,
			"on_request_ms", onRequestDuration)
		return
	}

	// Stage 2: Before hooks
	beforeStart := time.Now()
	r.runHooks(models.HookBefore, ctx)
	beforeDuration := time.Since(beforeStart).Milliseconds()

	if ctx.Handled {
		r.finalizeResponse(ctx, wrappedWriter)
		totalDuration := time.Since(requestStart).Milliseconds()
		r.logger.Debug("request completed (before hook handled)",
			"method", req.Method, "path", req.URL.Path,
			"status", ctx.ResponseStatus,
			"total_ms", totalDuration,
			"on_request_ms", onRequestDuration,
			"before_ms", beforeDuration)
		return
	}

	// Stage 3: Route handler (via Chi)
	handlerStart := time.Now()
	r.router.ServeHTTP(wrappedWriter, reqWithCtx)
	handlerDuration := time.Since(handlerStart).Milliseconds()

	// Stage 4: After hooks
	afterStart := time.Now()
	r.runHooks(models.HookAfter, ctx)
	afterDuration := time.Since(afterStart).Milliseconds()

	if ctx.Handled {
		r.finalizeResponse(ctx, wrappedWriter)
		totalDuration := time.Since(requestStart).Milliseconds()
		r.logger.Debug("request completed (after hook handled)",
			"method", req.Method, "path", req.URL.Path,
			"status", ctx.ResponseStatus,
			"total_ms", totalDuration,
			"on_request_ms", onRequestDuration,
			"before_ms", beforeDuration,
			"handler_ms", handlerDuration,
			"after_ms", afterDuration)
		return
	}

	// Stage 5: OnResponse hooks
	onResponseStart := time.Now()
	r.runHooks(models.HookOnResponse, ctx)
	onResponseDuration := time.Since(onResponseStart).Milliseconds()

	// Flush deferred writes or captured response
	r.finalizeResponse(ctx, wrappedWriter)

	// Log complete request timing (sync hooks only - async hooks run in background)
	totalDuration := time.Since(requestStart).Milliseconds()
	r.logger.Debug("request completed",
		"method", req.Method, "path", req.URL.Path,
		"status", ctx.ResponseStatus,
		"total_ms", totalDuration,
		"on_request_ms", onRequestDuration,
		"before_ms", beforeDuration,
		"handler_ms", handlerDuration,
		"after_ms", afterDuration,
		"on_response_ms", onResponseDuration)
}

// isStreamingResponse checks if the response appears to be streaming based on headers
func (r *Router) isStreamingResponse(headers http.Header) bool {
	contentType := headers.Get("Content-Type")
	if contentType == "" {
		return false
	}

	// Check for streaming content types
	streamingPatterns := []string{
		"text/event-stream",
		"application/octet-stream",
		"multipart/",
		"application/x-ndjson",
	}

	for _, pattern := range streamingPatterns {
		if strings.Contains(contentType, pattern) {
			r.logger.Debug("Detected streaming response, skipping buffering", "content_type", contentType)
			return true
		}
	}

	// Check for chunked encoding
	transferEncoding := headers.Get("Transfer-Encoding")
	if strings.Contains(strings.ToLower(transferEncoding), "chunked") {
		r.logger.Debug("Detected chunked transfer encoding, skipping buffering")
		return true
	}

	return false
}

func (r *Router) finalizeResponse(ctx *models.RequestContext, w *router.DeferredResponseWriter) {
	if ctx.ResponseReady {
		w.OverrideWithContext(ctx)
	}
	w.Flush()
}
