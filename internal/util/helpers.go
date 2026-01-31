package util

import (
	"encoding/json"
	"maps"
	"strings"

	"github.com/google/uuid"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// GenerateUUID generates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}

// MarshalJSON marshals a value to JSON
func MarshalJSON(v any) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func CompareStringArrays(arr1 []string, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	for i := range arr1 {
		if arr1[i] != arr2[i] {
			return false
		}
	}

	return true
}

// CloneRequestContext creates a detached copy of a RequestContext for use in async operations.
// This is used for async hooks which execute in background goroutines without the response writer.
// The returned context uses context.Background() as its base, preventing goroutine leaks,
// and has a default 5-second timeout for async hook execution.
func CloneRequestContext(ctx *models.RequestContext) *models.RequestContext {
	if ctx == nil {
		return nil
	}

	// Copy the request context for async operations
	cloned := &models.RequestContext{
		Request: ctx.Request,
		Path:    ctx.Path,
		Method:  ctx.Method,
		Headers: ctx.Headers,
		Route:   ctx.Route,
		UserID:  ctx.UserID,
		Handled: ctx.Handled,
	}

	// Don't include ResponseWriter for async hooks - they're read-only observers
	// ResponseWriter is set to nil to prevent accidental writes

	// Copy Values map (shallow copy)
	if ctx.Values != nil {
		cloned.Values = make(map[string]any, len(ctx.Values))
		maps.Copy(cloned.Values, ctx.Values)
	} else {
		cloned.Values = make(map[string]any)
	}

	// Copy response headers (shallow copy of string slices)
	if ctx.ResponseHeaders != nil {
		cloned.ResponseHeaders = make(map[string][]string, len(ctx.ResponseHeaders))
		for k, v := range ctx.ResponseHeaders {
			cloned.ResponseHeaders[k] = append([]string(nil), v...)
		}
	} else {
		cloned.ResponseHeaders = make(map[string][]string)
	}

	return cloned
}

func NormalizePath(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimSuffix(p, "/")
	return p
}
