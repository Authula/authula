package util

import (
	"net/http"
	"strconv"
)

// GetQueryString retrieves a query parameter as a string.
// If the parameter is missing, returns the provided default value.
func GetQueryString(r *http.Request, key string, defaultValue string) string {
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
