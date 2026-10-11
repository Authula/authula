package handlers

import (
	"errors"
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
)

func mapHttpErrorStatus(err error) int {
	if status, ok := coreerrors.StatusFor(err); ok {
		return status
	}
	if errors.Is(err, accesscontrolconstants.ErrCannotUpdateSystemRole) {
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

func mapHttpErrorMessage(err error) string {
	if err == nil {
		return "internal server error"
	}

	switch {
	case errors.Is(err, coreerrors.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, coreerrors.ErrForbidden), errors.Is(err, coreerrors.ErrInsufficientPermissions):
		return "forbidden"
	case errors.Is(err, coreerrors.ErrNotFound):
		return "not found"
	case errors.Is(err, coreerrors.ErrConflict):
		return "conflict"
	case errors.Is(err, coreerrors.ErrBadRequest):
		return "bad request"
	case errors.Is(err, coreerrors.ErrUnprocessableEntity):
		return "unprocessable entity"
	case errors.Is(err, accesscontrolconstants.ErrCannotUpdateSystemRole):
		return "cannot update system role"
	default:
		return err.Error()
	}
}
