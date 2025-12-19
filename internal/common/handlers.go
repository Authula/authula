package common

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type Handler interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

func WrapHandler(h Handler) models.CustomRouteHandler {
	return func(config *models.Config) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			h.Handle(w, req)
		})
	}
}
