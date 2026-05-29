package handlers

import (
	"errors"
	"net/http"

	internalerrors "github.com/Authula/authula/internal/errors"
)

func mapAdminHttpErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	switch {
	case errors.Is(err, internalerrors.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, internalerrors.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, internalerrors.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, internalerrors.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, internalerrors.ErrConflict):
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
	case errors.Is(err, internalerrors.ErrBadRequest):
		return "bad request"
	case errors.Is(err, internalerrors.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, internalerrors.ErrForbidden):
		return "forbidden"
	case errors.Is(err, internalerrors.ErrNotFound):
		return "not found"
	case errors.Is(err, internalerrors.ErrConflict):
		return "conflict"
	default:
		return err.Error()
	}
}
