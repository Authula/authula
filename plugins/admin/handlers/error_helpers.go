package handlers

import (
	"errors"
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
)

func mapAdminHttpErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	switch {
	case errors.Is(err, coreerrors.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, coreerrors.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, coreerrors.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, coreerrors.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, coreerrors.ErrConflict):
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
	case errors.Is(err, coreerrors.ErrBadRequest):
		return "bad request"
	case errors.Is(err, coreerrors.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, coreerrors.ErrForbidden):
		return "forbidden"
	case errors.Is(err, coreerrors.ErrNotFound):
		return "not found"
	case errors.Is(err, coreerrors.ErrConflict):
		return "conflict"
	default:
		return err.Error()
	}
}
