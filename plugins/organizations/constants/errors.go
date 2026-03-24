package constants

import (
	"errors"
	"net/http"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
)

var (
	ErrOrganizationsQuotaExceeded = errors.New("organizations quota exceeded")
	ErrMembersQuotaExceeded       = errors.New("members quota exceeded")
	ErrInvitationsQuotaExceeded   = errors.New("invitations quota exceeded")
)

func HandleError(err error, reqCtx *models.RequestContext) {
	var status int

	switch err {
	case ErrOrganizationsQuotaExceeded, ErrMembersQuotaExceeded, ErrInvitationsQuotaExceeded:
		status = http.StatusTooManyRequests
	}

	if status != 0 {
		reqCtx.SetJSONResponse(status, map[string]any{"message": err.Error()})
		reqCtx.Handled = true
		return
	}

	internalerrors.HandleError(err, reqCtx)
}
