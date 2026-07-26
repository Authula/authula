package types

import (
	"time"

	emailtmpl "github.com/Authula/authula/core/email/template"
)

type OrganizationInvitationContext struct {
	emailtmpl.CommonContext
	InvitationEmail  string
	OrganizationName string
	Role             string
	InviteLink       string
	Expiry           time.Duration
}
