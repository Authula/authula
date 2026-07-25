package constants

import (
	"errors"
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
)

var (
	ErrOrganizationsQuotaExceeded    = errors.New("organizations quota exceeded")
	ErrMembersQuotaExceeded          = errors.New("members quota exceeded")
	ErrInvitationsQuotaExceeded      = errors.New("invitations quota exceeded")
	ErrInvitationVerificationFailed  = errors.New("verification token not found or already used")
	ErrInvitationVerificationExpired = errors.New("verification token has expired")
	ErrInvitationEmailMismatch       = errors.New("this invitation was sent to a different email address")
)

func HandleError(err error, reqCtx *models.RequestContext) {
	var status int

	switch err {
	case ErrOrganizationsQuotaExceeded, ErrMembersQuotaExceeded, ErrInvitationsQuotaExceeded:
		status = http.StatusTooManyRequests
	case ErrInvitationVerificationFailed:
		status = http.StatusNotFound
	case ErrInvitationVerificationExpired:
		status = http.StatusGone
	case ErrInvitationEmailMismatch:
		status = http.StatusForbidden
	}

	if status != 0 {
		reqCtx.SetJSONResponse(status, map[string]any{"message": err.Error()})
		reqCtx.Handled = true
		return
	}

	coreerrors.HandleError(err, reqCtx)
}
