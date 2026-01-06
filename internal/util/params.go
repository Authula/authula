package util

import (
	"net/http"
	"strconv"
)

// GetQueryParam retrieves a query parameter from the request.
// If the parameter is missing, returns the provided default value.
func GetQueryParam(r *http.Request, key string, defaultValue string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetQueryInt retrieves a query parameter as an int.
// If missing or invalid, returns the provided default value.
func GetQueryInt(r *http.Request, key string, defaultValue int) int {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetQueryBool retrieves a query parameter as a bool.
// Accepts "true", "1", "yes" (case insensitive) as true.
// Returns defaultValue if missing or invalid.
func GetQueryBool(r *http.Request, key string, defaultValue bool) bool {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}

	switch valStr {
	case "true", "1", "yes", "TRUE", "Yes":
		return true
	case "false", "0", "no", "FALSE", "No":
		return false
	default:
		return defaultValue
	}
}
