package handlers

import (
	"errors"
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
)

func mapAdminHttpErrorStatus(err error) int {
	if status, ok := coreerrors.StatusFor(err); ok {
		return status
	}
	return http.StatusInternalServerError
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
	case errors.Is(err, coreerrors.ErrForbidden), errors.Is(err, coreerrors.ErrInsufficientPermissions):
		return "forbidden"
	case errors.Is(err, coreerrors.ErrNotFound):
		return "not found"
	case errors.Is(err, coreerrors.ErrConflict):
		return "conflict"
	case errors.Is(err, coreerrors.ErrUnprocessableEntity):
		return "unprocessable entity"
	default:
		return err.Error()
	}
}
