package types

import (
	"time"

	emailtmpl "github.com/Authula/authula/internal/email/template"
)

type OrganizationInvitationContext struct {
	emailtmpl.CommonContext
	InvitationEmail  string
	OrganizationName string
	Role             string
	AcceptLink       string
	Expiry           time.Duration
}
