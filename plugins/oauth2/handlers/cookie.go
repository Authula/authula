package handlers

import (
	"net/http"
	"time"

	"github.com/Authula/authula/plugins/oauth2/constants"
)

// CookieOptions describes the attributes applied to OAuth2 flow cookies.
type CookieOptions struct {
	Secure   bool
	Domain   string
	SameSite http.SameSite
	MaxAge   time.Duration
}

func (o CookieOptions) set(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   o.Domain,
		Expires:  time.Now().Add(o.effectiveMaxAge()),
		MaxAge:   int(o.effectiveMaxAge().Seconds()),
		HttpOnly: true,
		Secure:   o.Secure,
		SameSite: o.SameSite,
	})
}

func (o CookieOptions) clear(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   o.Domain,
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   o.Secure,
		SameSite: o.SameSite,
	})
}

func (o CookieOptions) effectiveMaxAge() time.Duration {
	if o.MaxAge <= 0 {
		return constants.FlowCookieTTL
	}
	return o.MaxAge
}

func (o CookieOptions) clearFlowCookies(w http.ResponseWriter) {
	o.clear(w, constants.CookieState)
	o.clear(w, constants.CookieRedirectTo)
	o.clear(w, constants.CookieVerifier)
}
