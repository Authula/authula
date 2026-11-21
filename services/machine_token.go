package services

import (
	"context"
	"time"
)

// ClientCredentialsTokenRequest describes a machine token issued for an OAuth
// 2.0 client credentials grant.
type ClientCredentialsTokenRequest struct {
	ClientID       string
	OrganizationID string
	Scopes         []string
	ExpiresIn      time.Duration
}

// ClientCredentialsToken is the result of a client credentials token request.
// ExpiresIn is the effective TTL actually used by the issuer.
type ClientCredentialsToken struct {
	AccessToken string
	ExpiresIn   time.Duration
	TokenType   string
}

// ClientCredentialsTokenIssuer issues JWT access tokens for confidential OAuth
// clients using the client credentials grant.
type ClientCredentialsTokenIssuer interface {
	IssueClientCredentialsToken(
		ctx context.Context,
		req ClientCredentialsTokenRequest,
	) (*ClientCredentialsToken, error)
}
