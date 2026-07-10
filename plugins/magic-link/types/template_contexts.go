package types

import (
	"time"

	emailtmpl "github.com/Authula/authula/core/email/template"
)

type MagicLinkSignInContext struct {
	emailtmpl.CommonContext
	UserEmail string
	MagicLink string
	Expiry    time.Duration
}
