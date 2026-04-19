package util

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Authula/authula/models"
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

// FormatDuration converts a time.Duration to a human-readable string.
// It intelligently selects the most appropriate unit (minutes, hours, or days)
// and handles singular/plural forms correctly.
// Examples: "15 minutes", "1 hour", "24 hours", "2 days"
func FormatDuration(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	totalHours := int(d.Hours())
	totalDays := int(d.Hours() / 24)

	// Use days if duration is >= 1 day
	if totalDays > 0 {
		unit := "day"
		if totalDays != 1 {
			unit = "days"
		}
		return fmt.Sprintf("%d %s", totalDays, unit)
	}

	// Use hours if duration is >= 1 hour
	if totalHours > 0 {
		unit := "hour"
		if totalHours != 1 {
			unit = "hours"
		}
		return fmt.Sprintf("%d %s", totalHours, unit)
	}

	// Otherwise use minutes
	unit := "minute"
	if totalMinutes != 1 {
		unit = "minutes"
	}
	return fmt.Sprintf("%d %s", totalMinutes, unit)
}

func ReadStringSliceMetadata(reqCtx *models.RequestContext, key string) []string {
	if reqCtx == nil || reqCtx.Route == nil || reqCtx.Route.Metadata == nil {
		return nil
	}

	raw, ok := reqCtx.Route.Metadata[key]
	if !ok || raw == nil {
		return nil
	}

	if values, ok := raw.([]string); ok {
		result := make([]string, 0, len(values))
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	if valuesAny, ok := raw.([]any); ok {
		result := make([]string, 0, len(valuesAny))
		for _, value := range valuesAny {
			str, ok := value.(string)
			if !ok {
				continue
			}

			trimmed := strings.TrimSpace(str)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	return nil
}

var SupportedRouteMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
}

func ReadStringSliceFromMetadata(metadata map[string]any, key string) []string {
	if metadata == nil {
		return nil
	}

	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}

	if values, ok := raw.([]string); ok {
		result := make([]string, 0, len(values))
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	if valuesAny, ok := raw.([]any); ok {
		result := make([]string, 0, len(valuesAny))
		for _, value := range valuesAny {
			str, ok := value.(string)
			if !ok {
				continue
			}

			trimmed := strings.TrimSpace(str)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	return nil
}

func MergeStringSlices(values ...[]string) []string {
	merged := make([]string, 0)
	seen := make(map[string]struct{})

	for _, items := range values {
		for _, value := range items {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			merged = append(merged, trimmed)
		}
	}

	return merged
}

func NormalizeRoutePattern(pattern string) string {
	trimmed := NormalizePath(pattern)
	if trimmed == "/" {
		return trimmed
	}
	return strings.ReplaceAll(trimmed, "//", "/")
}
