package models

import (
	"net/http"
)

type Route struct {
	Method     string
	Path       string
	Handler    http.Handler
	Middleware []func(http.Handler) http.Handler
	// Metadata holds route-specific metadata, including plugin IDs ("plugins"),
	// custom tags, and plugin-specific attributes for conditional hook execution.
	Metadata map[string]any
}
