package handlers

import (
	"errors"
	"net/http"

	internalerrors "github.com/Authula/authula/internal/errors"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
)

func mapHttpErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	switch {
	case errors.Is(err, internalerrors.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, internalerrors.ErrForbidden), errors.Is(err, accesscontrolconstants.ErrCannotUpdateSystemRole):
		return http.StatusForbidden
	case errors.Is(err, internalerrors.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, internalerrors.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, internalerrors.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, internalerrors.ErrUnprocessableEntity):
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
	case errors.Is(err, internalerrors.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, internalerrors.ErrForbidden):
		return "forbidden"
	case errors.Is(err, internalerrors.ErrNotFound):
		return "not found"
	case errors.Is(err, internalerrors.ErrConflict):
		return "conflict"
	case errors.Is(err, internalerrors.ErrBadRequest):
		return "bad request"
	case errors.Is(err, internalerrors.ErrUnprocessableEntity):
		return "unprocessable entity"
	case errors.Is(err, accesscontrolconstants.ErrCannotUpdateSystemRole):
		return "cannot update system role"
	default:
		return err.Error()
	}
}
