package models

import (
	"context"
	"encoding/json"
	"net/http"
)

type ContextKey string

const (
	ContextUserID                       ContextKey = "user_id"
	ContextSessionID                    ContextKey = "session_id"
	ContextSessionToken                 ContextKey = "session_token"
	ContextRequestContext               ContextKey = "request_context"
	ContextAuthSuccess                  ContextKey = "auth.success"
	ContextAuthSignOut                  ContextKey = "auth.sign_out"
	ContextAuthIdempotentSkipTokensMint ContextKey = "auth.idempotent_skip_tokens_mint"
)

func (k ContextKey) String() string {
	return string(k)
}

// RequestContext provides a structured abstraction for passing context
// through request lifecycle hooks. It encapsulates all request-related
// information and provides control mechanisms for hooks.
type RequestContext struct {
	// Core HTTP components
	Request        *http.Request
	ResponseWriter http.ResponseWriter

	// Parsed request metadata
	Path    string
	Method  string
	Headers http.Header

	// User information (may be nil if not authenticated)
	UserID *string

	// Generic key-value storage for hooks to share data
	Values map[string]any

	// Route is the matched route, assigned by the router after route matching.
	// Plugins use Route.Metadata["plugins"] to determine if they should execute.
	Route *Route

	// Handled flag indicates whether a hook has handled the request
	// and subsequent handlers should not be called
	Handled bool

	// Response capture fields allow handlers and hooks to override
	// the final HTTP response written to the client
	ResponseStatus  int
	ResponseHeaders http.Header
	ResponseBody    []byte
	ResponseReady   bool
	ResponseData    any
}

// NewContextWithRequestContext returns a new context with the RequestContext attached
func NewContextWithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, ContextRequestContext, rc)
}

// SetRequestContext is an alias for NewContextWithRequestContext for convenience
func SetRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return NewContextWithRequestContext(ctx, rc)
}

// GetRequestContext retrieves the RequestContext from a context.Context
func GetRequestContext(ctx context.Context) (*RequestContext, bool) {
	rc, ok := ctx.Value(ContextRequestContext).(*RequestContext)
	return rc, ok
}

// SetResponse sets the captured response fields that will be written once
// the request lifecycle (including hooks) completes.
func (reqCtx *RequestContext) SetResponse(status int, headers http.Header, body []byte) {
	reqCtx.ResponseStatus = status
	if headers != nil {
		reqCtx.ResponseHeaders = make(http.Header)
		for key, values := range headers {
			reqCtx.ResponseHeaders[key] = append([]string(nil), values...)
		}
	} else {
		reqCtx.ResponseHeaders = nil
	}
	reqCtx.ResponseBody = body
	reqCtx.ResponseReady = true
}

// SetJSONResponse marshals the provided payload and stores it as the
// captured response body with the appropriate content type header.
// If marshaling fails, it falls back to a 500 Internal Server Error response.
func (reqCtx *RequestContext) SetJSONResponse(status int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		// Fallback: set a plain text error response
		headers := make(http.Header)
		headers.Set("Content-Type", "text/plain")
		reqCtx.SetResponse(500, headers, []byte("Internal Server Error"))
		return
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	reqCtx.SetResponse(status, headers, data)
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	// First, check direct Go context (fast path for third-party plugins)
	value := ctx.Value(ContextUserID)
	if value != nil {
		if id, ok := value.(string); ok {
			return id, true
		}
	}

	// Fallback: check RequestContext (for custom runtime)
	if rc, ok := GetRequestContext(ctx); ok && rc.UserID != nil {
		return *rc.UserID, true
	}

	return "", false
}

// SetUserIDInContext sets the user ID in both the RequestContext and the underlying Go context.
func (reqCtx *RequestContext) SetUserIDInContext(userID string) {
	reqCtx.UserID = &userID
	reqCtx.Request = reqCtx.Request.WithContext(context.WithValue(reqCtx.Request.Context(), ContextUserID, userID))
}

// GetUserIDFromRequest extracts the user ID from an HTTP request's context.
// It returns the user ID and a boolean indicating whether it was found.
func GetUserIDFromRequest(req *http.Request) (string, bool) {
	return GetUserIDFromContext(req.Context())
}
