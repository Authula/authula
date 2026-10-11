package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
)

func TestStatusFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedOK   bool
	}{
		{name: "unauthorized", err: coreerrors.ErrUnauthorized, expectedCode: http.StatusUnauthorized, expectedOK: true},
		{name: "forbidden", err: coreerrors.ErrForbidden, expectedCode: http.StatusForbidden, expectedOK: true},
		{name: "insufficient permissions maps to forbidden", err: coreerrors.ErrInsufficientPermissions, expectedCode: http.StatusForbidden, expectedOK: true},
		{name: "not found", err: coreerrors.ErrNotFound, expectedCode: http.StatusNotFound, expectedOK: true},
		{name: "conflict", err: coreerrors.ErrConflict, expectedCode: http.StatusConflict, expectedOK: true},
		{name: "bad request", err: coreerrors.ErrBadRequest, expectedCode: http.StatusBadRequest, expectedOK: true},
		{name: "unprocessable entity", err: coreerrors.ErrUnprocessableEntity, expectedCode: http.StatusUnprocessableEntity, expectedOK: true},
		{name: "wrapped sentinel is recognized", err: fmt.Errorf("authorizing: %w", coreerrors.ErrInsufficientPermissions), expectedCode: http.StatusForbidden, expectedOK: true},
		{name: "unknown error is not recognized", err: errors.New("boom"), expectedOK: false},
		{name: "nil is not recognized", err: nil, expectedOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, ok := coreerrors.StatusFor(tt.err)

			require.Equal(t, tt.expectedOK, ok)
			require.Equal(t, tt.expectedCode, status)
		})
	}
}

func TestHandleError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		expectedStatus  int
		expectedMessage string
	}{
		{name: "insufficient permissions is forbidden", err: coreerrors.ErrInsufficientPermissions, expectedStatus: http.StatusForbidden, expectedMessage: "insufficient permissions"},
		{name: "forbidden is forbidden", err: coreerrors.ErrForbidden, expectedStatus: http.StatusForbidden, expectedMessage: "forbidden"},
		{name: "unauthorized is unauthorized", err: coreerrors.ErrUnauthorized, expectedStatus: http.StatusUnauthorized, expectedMessage: "unauthorized"},
		{name: "not found", err: coreerrors.ErrNotFound, expectedStatus: http.StatusNotFound, expectedMessage: "not found"},
		{name: "conflict", err: coreerrors.ErrConflict, expectedStatus: http.StatusConflict, expectedMessage: "conflict"},
		{name: "unprocessable entity", err: coreerrors.ErrUnprocessableEntity, expectedStatus: http.StatusUnprocessableEntity, expectedMessage: "unprocessable entity"},
		{name: "unknown error defaults to bad request", err: errors.New("boom"), expectedStatus: http.StatusBadRequest, expectedMessage: "boom"},
		{name: "wrapped sentinel keeps its status", err: fmt.Errorf("authorizing: %w", coreerrors.ErrInsufficientPermissions), expectedStatus: http.StatusForbidden, expectedMessage: "authorizing: insufficient permissions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := &models.RequestContext{}

			coreerrors.HandleError(tt.err, reqCtx)

			require.True(t, reqCtx.Handled)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			require.JSONEq(t, fmt.Sprintf(`{"message":%q}`, tt.expectedMessage), string(reqCtx.ResponseBody))
		})
	}
}
