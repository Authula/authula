package handlers

import (
	"net/http"
	"time"

	"github.com/Authula/authula/plugins/oauth2/constants"
)

// bindingCookieSameSite is forced for the OAuth flow binding cookie. SameSite=Strict
// prevents the browser from sending the cookie on the provider's cross-site
// callback.
const bindingCookieSameSite = http.SameSiteLaxMode

// CookieOptions describes the attributes applied to the OAuth2 flow binding cookie.
type CookieOptions struct {
	Secure bool
	Domain string
	MaxAge time.Duration
}

func (o CookieOptions) setBinding(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     constants.CookieFlowBinding,
		Value:    value,
		Path:     "/",
		Domain:   o.Domain,
		Expires:  time.Now().Add(o.MaxAge),
		MaxAge:   int(o.MaxAge.Seconds()),
		HttpOnly: true,
		Secure:   o.Secure,
		SameSite: bindingCookieSameSite,
	})
}
