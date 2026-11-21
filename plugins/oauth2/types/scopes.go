package types

import "strings"

// ParseScope parses a space-delimited OAuth 2.0 scope value. It returns false
// for leading, trailing or repeated spaces, empty tokens, duplicates, tokens
// with characters outside RFC 6749 section 3.3, and wildcard tokens.
func ParseScope(raw string) ([]string, bool) {
	if raw == "" {
		return nil, false
	}

	tokens := strings.Split(raw, " ")
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if !validScopeToken(token) {
			return nil, false
		}
		if _, exists := seen[token]; exists {
			return nil, false
		}
		seen[token] = struct{}{}
	}

	return tokens, true
}

// JoinScope formats scopes as a single space-delimited OAuth 2.0 scope value.
func JoinScope(scopes []string) string {
	return strings.Join(scopes, " ")
}

// ScopeSubset reports whether every requested scope is present in allowed.
// Matching is exact and case-sensitive. Wildcards are never matched.
func ScopeSubset(requested, allowed []string) bool {
	if len(requested) == 0 {
		return true
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, scope := range allowed {
		allowedSet[scope] = struct{}{}
	}

	for _, scope := range requested {
		if strings.Contains(scope, "*") {
			return false
		}
		if _, ok := allowedSet[scope]; !ok {
			return false
		}
	}

	return true
}

// IsOIDCScope reports whether scope belongs to the OpenID Connect scope set.
func IsOIDCScope(scope string) bool {
	switch scope {
	case ScopeOpenID, ScopeProfile, ScopeEmail, ScopeAddress, ScopePhone, ScopeOfflineAccess:
		return true
	default:
		return false
	}
}

func validScopeToken(token string) bool {
	if token == "" || strings.ContainsRune(token, '*') {
		return false
	}

	for i := 0; i < len(token); i++ {
		b := token[i]
		if b < 0x21 || b > 0x7E || b == '"' || b == '\\' {
			return false
		}
	}

	return true
}
