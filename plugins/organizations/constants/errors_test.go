package constants_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/organizations/constants"
)

func TestHandleError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		expectedStatus  int
		expectedCode    string
		expectedMessage string
	}{
		{
			name:            "organizations quota is a conflict, not a rate limit",
			err:             constants.ErrOrganizationsQuotaExceeded,
			expectedStatus:  http.StatusConflict,
			expectedCode:    constants.CodeOrganizationsQuotaExceeded,
			expectedMessage: "organizations quota exceeded",
		},
		{
			name:            "members quota is a conflict, not a rate limit",
			err:             constants.ErrMembersQuotaExceeded,
			expectedStatus:  http.StatusConflict,
			expectedCode:    constants.CodeMembersQuotaExceeded,
			expectedMessage: "members quota exceeded",
		},
		{
			name:            "invitations quota is a conflict, not a rate limit",
			err:             constants.ErrInvitationsQuotaExceeded,
			expectedStatus:  http.StatusConflict,
			expectedCode:    constants.CodeInvitationsQuotaExceeded,
			expectedMessage: "invitations quota exceeded",
		},
		{
			name:            "invitation email mismatch stays forbidden",
			err:             constants.ErrInvitationEmailMismatch,
			expectedStatus:  http.StatusForbidden,
			expectedCode:    constants.CodeInvitationEmailMismatch,
			expectedMessage: "this invitation was sent to a different email address",
		},
		{
			// The regression test for dispatching on errors.AsType rather than
			// on error identity: a wrapped sentinel used to fall through to the
			// core handler and silently degrade to 400.
			name:            "a wrapped quota error keeps its status and code",
			err:             fmt.Errorf("creating organization: %w", constants.ErrMembersQuotaExceeded),
			expectedStatus:  http.StatusConflict,
			expectedCode:    constants.CodeMembersQuotaExceeded,
			expectedMessage: "members quota exceeded",
		},
		{
			name:            "doubly wrapped quota error keeps its status and code",
			err:             fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", constants.ErrOrganizationsQuotaExceeded)),
			expectedStatus:  http.StatusConflict,
			expectedCode:    constants.CodeOrganizationsQuotaExceeded,
			expectedMessage: "organizations quota exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := internaltests.NewRequestContext(t, http.MethodPost, "/organizations", nil)

			constants.HandleError(tt.err, reqCtx)

			internaltests.AssertErrorResponse(t, reqCtx, tt.expectedStatus, tt.expectedCode, tt.expectedMessage)
		})
	}
}

// Errors the plugin does not own must still reach the core handler unchanged.
func TestHandleErrorDelegatesToCore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		expectedStatus  int
		expectedMessage string
	}{
		{name: "not found", err: coreerrors.ErrNotFound, expectedStatus: http.StatusNotFound, expectedMessage: "not found"},
		{name: "forbidden", err: coreerrors.ErrForbidden, expectedStatus: http.StatusForbidden, expectedMessage: "forbidden"},
		{name: "conflict", err: coreerrors.ErrConflict, expectedStatus: http.StatusConflict, expectedMessage: "conflict"},
		{name: "unauthorized", err: coreerrors.ErrUnauthorized, expectedStatus: http.StatusUnauthorized, expectedMessage: "unauthorized"},
		{name: "unknown error falls through", err: errors.New("some repository failure"), expectedStatus: http.StatusBadRequest, expectedMessage: "some repository failure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := internaltests.NewRequestContext(t, http.MethodPost, "/organizations", nil)

			constants.HandleError(tt.err, reqCtx)

			internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
		})
	}
}

// Typing the sentinels must not change how callers compare them. Every service
// returns the bare sentinel and the service tests assert on the value, so
// identity and errors.Is have to keep holding.
func TestSentinelsRemainComparable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             *constants.Error
		expectedMessage string
	}{
		{name: "organizations quota", err: constants.ErrOrganizationsQuotaExceeded, expectedMessage: "organizations quota exceeded"},
		{name: "members quota", err: constants.ErrMembersQuotaExceeded, expectedMessage: "members quota exceeded"},
		{name: "invitations quota", err: constants.ErrInvitationsQuotaExceeded, expectedMessage: "invitations quota exceeded"},
		{name: "invitation email mismatch", err: constants.ErrInvitationEmailMismatch, expectedMessage: "this invitation was sent to a different email address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var asError error = tt.err

			require.Equal(t, tt.expectedMessage, tt.err.Error(), "message must stay byte-identical")
			require.True(t, asError == tt.err, "identity comparison must still hold") //nolint:testifylint // asserting == on purpose
			require.ErrorIs(t, asError, tt.err)
			require.ErrorIs(t, fmt.Errorf("wrapped: %w", asError), tt.err)
		})
	}
}

// Error codes are a public contract: once shipped, a code is never renamed.
// This golden list makes any addition or rename an explicit, reviewed diff.
func TestErrorCodeRegistry(t *testing.T) {
	t.Parallel()

	require.Equal(t, map[string]string{
		"organizations_quota_exceeded": "organizations quota exceeded",
		"members_quota_exceeded":       "members quota exceeded",
		"invitations_quota_exceeded":   "invitations quota exceeded",
		"invitation_email_mismatch":    "this invitation was sent to a different email address",
	}, map[string]string{
		constants.ErrOrganizationsQuotaExceeded.Code: constants.ErrOrganizationsQuotaExceeded.Message,
		constants.ErrMembersQuotaExceeded.Code:       constants.ErrMembersQuotaExceeded.Message,
		constants.ErrInvitationsQuotaExceeded.Code:   constants.ErrInvitationsQuotaExceeded.Message,
		constants.ErrInvitationEmailMismatch.Code:    constants.ErrInvitationEmailMismatch.Message,
	})
}

// No organizations error may use 429: that status means rate limiting in this
// API and is reserved for the rate-limit and api-key plugins, which pair it
// with Retry-After and X-RateLimit-* headers.
func TestNoQuotaErrorUsesTooManyRequests(t *testing.T) {
	t.Parallel()

	for _, err := range []*constants.Error{
		constants.ErrOrganizationsQuotaExceeded,
		constants.ErrMembersQuotaExceeded,
		constants.ErrInvitationsQuotaExceeded,
		constants.ErrInvitationEmailMismatch,
	} {
		require.NotEqual(t, http.StatusTooManyRequests, err.Status, "%s must not use 429", err.Code)
	}
}
