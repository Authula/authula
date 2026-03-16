package handlers

import (
	"errors"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/constants"
)

func mapHttpErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	switch {
	case errors.Is(err, constants.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, constants.ErrForbidden), errors.Is(err, constants.ErrCannotUpdateSystemRole):
		return http.StatusForbidden
	case errors.Is(err, constants.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, constants.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, constants.ErrBadRequest):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func mapHttpErrorMessage(err error) string {
	if err == nil {
		return "internal server error"
	}

	switch {
	case errors.Is(err, constants.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, constants.ErrForbidden):
		return "forbidden"
	case errors.Is(err, constants.ErrNotFound):
		return "not found"
	case errors.Is(err, constants.ErrConflict):
		return "conflict"
	case errors.Is(err, constants.ErrBadRequest):
		return "bad request"
	case errors.Is(err, constants.ErrCannotUpdateSystemRole):
		return "cannot update system role"
	default:
		return err.Error()
	}
}
