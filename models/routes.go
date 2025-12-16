package models

import (
	"net/http"
)

type CustomRouteMiddleware func(http.Handler) http.Handler

type CustomRouteHandler func(config *Config) http.Handler

type CustomRoute struct {
	Method     string
	Path       string
	Middleware []CustomRouteMiddleware
	Handler    CustomRouteHandler
}
