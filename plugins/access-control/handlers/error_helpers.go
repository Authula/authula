package handlers

import (
	"errors"
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
)

func mapHttpErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	switch {
	case errors.Is(err, coreerrors.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, coreerrors.ErrForbidden), errors.Is(err, accesscontrolconstants.ErrCannotUpdateSystemRole):
		return http.StatusForbidden
	case errors.Is(err, coreerrors.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, coreerrors.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, coreerrors.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, coreerrors.ErrUnprocessableEntity):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func mapHttpErrorMessage(err error) string {
	if err == nil {
		return "internal server error"
	}

	switch {
	case errors.Is(err, coreerrors.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, coreerrors.ErrForbidden):
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
