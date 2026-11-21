package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		want   []string
		wantOK bool
	}{
		{name: "single scope", raw: "read", want: []string{"read"}, wantOK: true},
		{name: "multiple scopes", raw: "read write", want: []string{"read", "write"}, wantOK: true},
		{name: "scope with punctuation", raw: "orders:read orders.write", want: []string{"orders:read", "orders.write"}, wantOK: true},
		{name: "empty value", raw: "", wantOK: false},
		{name: "leading space", raw: " read", wantOK: false},
		{name: "trailing space", raw: "read ", wantOK: false},
		{name: "repeated space", raw: "read  write", wantOK: false},
		{name: "duplicate scope", raw: "read read", wantOK: false},
		{name: "wildcard", raw: "*", wantOK: false},
		{name: "prefixed wildcard", raw: "orders*", wantOK: false},
		{name: "double quote", raw: `read"write`, wantOK: false},
		{name: "backslash", raw: `read\write`, wantOK: false},
		{name: "control character", raw: "read\twrite", wantOK: false},
		{name: "non ascii", raw: "léger", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseScope(tt.raw)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJoinScope(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "read write", JoinScope([]string{"read", "write"}))
	assert.Equal(t, "read", JoinScope([]string{"read"}))
	assert.Equal(t, "", JoinScope(nil))
	assert.Equal(t, "", JoinScope([]string{}))
}

func TestScopeSubset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested []string
		allowed   []string
		want      bool
	}{
		{name: "empty request is a subset", requested: nil, allowed: []string{"read"}, want: true},
		{name: "exact match", requested: []string{"read"}, allowed: []string{"read", "write"}, want: true},
		{name: "all requested scopes allowed", requested: []string{"read", "write"}, allowed: []string{"read", "write"}, want: true},
		{name: "missing scope", requested: []string{"admin"}, allowed: []string{"read", "write"}, want: false},
		{name: "case sensitive", requested: []string{"Read"}, allowed: []string{"read"}, want: false},
		{name: "wildcard in allowed does not match", requested: []string{"read"}, allowed: []string{"*"}, want: false},
		{name: "prefixed wildcard in allowed does not match", requested: []string{"orders:read"}, allowed: []string{"orders:*"}, want: false},
		{name: "wildcard in request is rejected", requested: []string{"*"}, allowed: []string{"*"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ScopeSubset(tt.requested, tt.allowed))
		})
	}
}

func TestIsOIDCScope(t *testing.T) {
	t.Parallel()

	for _, scope := range []string{"openid", "profile", "email", "address", "phone", "offline_access"} {
		assert.True(t, IsOIDCScope(scope), "expected %q to be an OIDC scope", scope)
	}

	for _, scope := range []string{"", "read", "openid:admin", "OpenID"} {
		assert.False(t, IsOIDCScope(scope), "expected %q not to be an OIDC scope", scope)
	}
}
