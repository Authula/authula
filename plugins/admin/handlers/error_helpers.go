package handlers

import (
	"errors"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
)

func mapAdminHttpErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	switch {
	case errors.Is(err, constants.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, constants.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, constants.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, constants.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, constants.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func mapAdminHttpErrorMessage(err error) string {
	if err == nil {
		return "internal server error"
	}

	switch {
	case errors.Is(err, constants.ErrBadRequest):
		return "bad request"
	case errors.Is(err, constants.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, constants.ErrForbidden):
		return "forbidden"
	case errors.Is(err, constants.ErrNotFound):
		return "not found"
	case errors.Is(err, constants.ErrConflict):
		return "conflict"
	default:
		return err.Error()
	}
}
