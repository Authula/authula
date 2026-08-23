package constants

import (
	"errors"
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
)

type Error struct {
	Code    string
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrOrganizationsQuotaExceeded = &Error{
		Code:    CodeOrganizationsQuotaExceeded,
		Status:  http.StatusConflict,
		Message: "organizations quota exceeded",
	}
	ErrMembersQuotaExceeded = &Error{
		Code:    CodeMembersQuotaExceeded,
		Status:  http.StatusConflict,
		Message: "members quota exceeded",
	}
	ErrInvitationsQuotaExceeded = &Error{
		Code:    CodeInvitationsQuotaExceeded,
		Status:  http.StatusConflict,
		Message: "invitations quota exceeded",
	}
	ErrInvitationEmailMismatch = &Error{
		Code:    CodeInvitationEmailMismatch,
		Status:  http.StatusForbidden,
		Message: "this invitation was sent to a different email address",
	}
)

const (
	CodeOrganizationsQuotaExceeded = "organizations_quota_exceeded"
	CodeMembersQuotaExceeded       = "members_quota_exceeded"
	CodeInvitationsQuotaExceeded   = "invitations_quota_exceeded"
	CodeInvitationEmailMismatch    = "invitation_email_mismatch"
)

func HandleError(err error, reqCtx *models.RequestContext) {
	if pluginErr, ok := errors.AsType[*Error](err); ok {
		reqCtx.SetJSONResponse(pluginErr.Status, map[string]any{
			"code":    pluginErr.Code,
			"message": pluginErr.Message,
		})
		reqCtx.Handled = true
		return
	}

	coreerrors.HandleError(err, reqCtx)
}
