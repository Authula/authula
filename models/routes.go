package models

import (
	"net/http"
)

type Route struct {
	Method     string
	Path       string
	Pattern    string
	Handler    http.Handler
	Middleware []func(http.Handler) http.Handler
	// Metadata holds route-specific metadata, including plugin IDs ("plugins"),
	// custom tags, and plugin-specific attributes for conditional hook execution.
	Metadata map[string]any
}

type RouteGroup struct {
	Path     string
	Routes   []Route
	Metadata map[string]any
}
