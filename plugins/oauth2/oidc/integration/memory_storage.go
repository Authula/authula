// Package integration contains the disposable Phase 0 spike that proves the
// ZITADEL OpenID Provider can be served through Authula's router.
package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

const (
	spikeBasePath  = "/api/auth"
	spikeOIDCPath  = "/oauth2/oidc"
	spikeClientID  = "spike-client"
	spikeSecret    = "spike-secret"
	spikeUserID    = "spike-user"
	spikeUserEmail = "spike@example.com"
	spikeUserName  = "Spike User"
)

var errUnsupported = errors.New("not supported by the Phase 0 memory storage")

var _ op.Storage = (*memoryStorage)(nil)

type memoryUser struct {
	id    string
	name  string
	email string
}

type memoryClient struct {
	id           string
	secret       string
	redirectURIs []string
	loginPath    string
	scopes       []string
}

func (c *memoryClient) GetID() string { return c.id }

func (c *memoryClient) RedirectURIs() []string {
	return slices.Clone(c.redirectURIs)
}

func (c *memoryClient) PostLogoutRedirectURIs() []string { return nil }

func (c *memoryClient) ApplicationType() op.ApplicationType { return op.ApplicationTypeWeb }

func (c *memoryClient) AuthMethod() oidc.AuthMethod { return oidc.AuthMethodPost }

func (c *memoryClient) ResponseTypes() []oidc.ResponseType {
	return []oidc.ResponseType{oidc.ResponseTypeCode}
}

func (c *memoryClient) GrantTypes() []oidc.GrantType {
	return []oidc.GrantType{oidc.GrantTypeCode}
}

func (c *memoryClient) LoginURL(requestID string) string {
	return c.loginPath + "?oauth2_request_id=" + url.QueryEscape(requestID)
}

func (c *memoryClient) AccessTokenType() op.AccessTokenType { return op.AccessTokenTypeJWT }

func (c *memoryClient) IDTokenLifetime() time.Duration { return 5 * time.Minute }

func (c *memoryClient) DevMode() bool { return false }

func (c *memoryClient) RestrictAdditionalIdTokenScopes() func([]string) []string {
	return slices.Clone
}

func (c *memoryClient) RestrictAdditionalAccessTokenScopes() func([]string) []string {
	return slices.Clone
}

func (c *memoryClient) IsScopeAllowed(scope string) bool {
	return slices.Contains(c.scopes, scope)
}

func (c *memoryClient) IDTokenUserinfoClaimsAssertion() bool { return true }

func (c *memoryClient) ClockSkew() time.Duration { return 0 }

type memoryAuthRequest struct {
	id            string
	scopes        []string
	responseType  oidc.ResponseType
	clientID      string
	redirectURI   string
	state         string
	nonce         string
	responseMode  oidc.ResponseMode
	prompt        []string
	maxAge        *uint
	codeChallenge *oidc.CodeChallenge
	subject       string
	authTime      time.Time
	done          bool
}

func (a *memoryAuthRequest) GetID() string                      { return a.id }
func (a *memoryAuthRequest) GetACR() string                     { return "" }
func (a *memoryAuthRequest) GetAMR() []string                   { return nil }
func (a *memoryAuthRequest) GetAudience() []string              { return nil }
func (a *memoryAuthRequest) GetAuthTime() time.Time             { return a.authTime }
func (a *memoryAuthRequest) GetClientID() string                { return a.clientID }
func (a *memoryAuthRequest) GetNonce() string                   { return a.nonce }
func (a *memoryAuthRequest) GetRedirectURI() string             { return a.redirectURI }
func (a *memoryAuthRequest) GetResponseType() oidc.ResponseType { return a.responseType }
func (a *memoryAuthRequest) GetResponseMode() oidc.ResponseMode { return a.responseMode }
func (a *memoryAuthRequest) GetState() string                   { return a.state }
func (a *memoryAuthRequest) GetSubject() string                 { return a.subject }
func (a *memoryAuthRequest) Done() bool                         { return a.done }
func (a *memoryAuthRequest) GetScopes() []string                { return slices.Clone(a.scopes) }

func (a *memoryAuthRequest) GetCodeChallenge() *oidc.CodeChallenge {
	if a.codeChallenge == nil {
		return nil
	}
	challenge := *a.codeChallenge
	return &challenge
}

func (a *memoryAuthRequest) clone() *memoryAuthRequest {
	if a == nil {
		return nil
	}
	cloned := *a
	cloned.scopes = slices.Clone(a.scopes)
	cloned.prompt = slices.Clone(a.prompt)
	if a.maxAge != nil {
		cloned.maxAge = new(*a.maxAge)
	}
	if a.codeChallenge != nil {
		challenge := *a.codeChallenge
		cloned.codeChallenge = &challenge
	}
	return &cloned
}

type memoryStorage struct {
	mu           sync.RWMutex
	client       *memoryClient
	user         memoryUser
	authRequests map[string]*memoryAuthRequest
	codes        map[string]string
	accessTokens map[string]string
	signingKey   *rsa.PrivateKey
	signingKeyID string
}

func newMemoryStorage(redirectURI string) (*memoryStorage, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}

	return &memoryStorage{
		client: &memoryClient{
			id:           spikeClientID,
			secret:       spikeSecret,
			redirectURIs: []string{redirectURI},
			loginPath:    spikeBasePath + spikeOIDCPath + "/login",
			scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail},
		},
		user: memoryUser{
			id:    spikeUserID,
			name:  spikeUserName,
			email: spikeUserEmail,
		},
		authRequests: make(map[string]*memoryAuthRequest),
		codes:        make(map[string]string),
		accessTokens: make(map[string]string),
		signingKey:   privateKey,
		signingKeyID: "spike-key",
	}, nil
}

func (s *memoryStorage) CreateAuthRequest(_ context.Context, request *oidc.AuthRequest, subject string) (op.AuthRequest, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	if subject == "" {
		subject = s.user.id
	}

	authRequest := &memoryAuthRequest{
		id:           id,
		scopes:       slices.Clone(request.Scopes),
		responseType: request.ResponseType,
		clientID:     request.ClientID,
		redirectURI:  request.RedirectURI,
		state:        request.State,
		nonce:        request.Nonce,
		responseMode: request.ResponseMode,
		prompt:       slices.Clone(request.Prompt),
		subject:      subject,
		authTime:     time.Now().UTC(),
	}
	if request.MaxAge != nil {
		authRequest.maxAge = new(*request.MaxAge)
	}
	if request.CodeChallenge != "" {
		authRequest.codeChallenge = &oidc.CodeChallenge{
			Challenge: request.CodeChallenge,
			Method:    request.CodeChallengeMethod,
		}
	}

	s.mu.Lock()
	s.authRequests[id] = authRequest
	s.mu.Unlock()

	return authRequest.clone(), nil
}

func (s *memoryStorage) AuthRequestByID(_ context.Context, id string) (op.AuthRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	authRequest, ok := s.authRequests[id]
	if !ok {
		return nil, fmt.Errorf("auth request %q not found", id)
	}
	return authRequest.clone(), nil
}

// AuthRequestByCode is single-use only because the provider calls
// DeleteAuthRequest after issuing tokens. Phase 1 must consume the code
// atomically at lookup instead.
func (s *memoryStorage) AuthRequestByCode(_ context.Context, code string) (op.AuthRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.codes[code]
	if !ok {
		return nil, fmt.Errorf("code not found")
	}
	authRequest, ok := s.authRequests[id]
	if !ok {
		return nil, fmt.Errorf("auth request for code not found")
	}
	return authRequest.clone(), nil
}

func (s *memoryStorage) SaveAuthCode(_ context.Context, id, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.authRequests[id]; !ok {
		return fmt.Errorf("auth request %q not found", id)
	}
	s.codes[code] = id
	return nil
}

func (s *memoryStorage) DeleteAuthRequest(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.authRequests, id)
	for code, authRequestID := range s.codes {
		if authRequestID == id {
			delete(s.codes, code)
		}
	}
	return nil
}

func (s *memoryStorage) CreateAccessToken(_ context.Context, _ op.TokenRequest) (string, time.Time, error) {
	tokenID, err := randomID()
	if err != nil {
		return "", time.Time{}, err
	}

	s.mu.Lock()
	s.accessTokens[tokenID] = s.user.id
	s.mu.Unlock()

	return tokenID, time.Now().UTC().Add(time.Hour), nil
}

func (s *memoryStorage) CreateAccessAndRefreshTokens(context.Context, op.TokenRequest, string) (string, string, time.Time, error) {
	return "", "", time.Time{}, errUnsupported
}

func (s *memoryStorage) TokenRequestByRefreshToken(context.Context, string) (op.RefreshTokenRequest, error) {
	return nil, op.ErrInvalidRefreshToken
}

func (s *memoryStorage) TerminateSession(context.Context, string, string) error { return nil }

func (s *memoryStorage) RevokeToken(_ context.Context, tokenOrTokenID, _ string, _ string) *oidc.Error {
	s.mu.Lock()
	delete(s.accessTokens, tokenOrTokenID)
	s.mu.Unlock()
	return nil
}

func (s *memoryStorage) GetRefreshTokenInfo(context.Context, string, string) (string, string, error) {
	return "", "", op.ErrInvalidRefreshToken
}

func (s *memoryStorage) SigningKey(context.Context) (op.SigningKey, error) {
	return memorySigningKey{id: s.signingKeyID, key: s.signingKey}, nil
}

func (s *memoryStorage) SignatureAlgorithms(context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}

func (s *memoryStorage) KeySet(context.Context) ([]op.Key, error) {
	return []op.Key{memoryKey{id: s.signingKeyID, key: &s.signingKey.PublicKey}}, nil
}

func (s *memoryStorage) GetClientByClientID(_ context.Context, clientID string) (op.Client, error) {
	if clientID != s.client.id {
		return nil, fmt.Errorf("client %q not found", clientID)
	}
	return s.client, nil
}

func (s *memoryStorage) AuthorizeClientIDSecret(_ context.Context, clientID, clientSecret string) error {
	if clientID != s.client.id || subtle.ConstantTimeCompare([]byte(clientSecret), []byte(s.client.secret)) != 1 {
		return errors.New("invalid client credentials")
	}
	return nil
}

func (s *memoryStorage) SetUserinfoFromScopes(_ context.Context, userinfo *oidc.UserInfo, _, _ string, scopes []string) error {
	s.setUserinfo(userinfo, scopes)
	return nil
}

// SetUserinfoFromToken is a spike simplification: the flow always requests
// profile and email. Phase 3 must derive claims from the access token's
// granted scopes and must not return claims for scopes that were not granted.
func (s *memoryStorage) SetUserinfoFromToken(_ context.Context, userinfo *oidc.UserInfo, _, _ string, _ string) error {
	s.setUserinfo(userinfo, []string{oidc.ScopeProfile, oidc.ScopeEmail})
	return nil
}

func (s *memoryStorage) SetIntrospectionFromToken(_ context.Context, userinfo *oidc.IntrospectionResponse, tokenID, subject, clientID string) error {
	userinfo.Subject = subject
	userinfo.ClientID = clientID
	userinfo.JWTID = tokenID
	userinfo.TokenType = "Bearer"
	return nil
}

func (s *memoryStorage) GetPrivateClaimsFromScopes(context.Context, string, string, []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (s *memoryStorage) GetKeyByIDAndClientID(context.Context, string, string) (*jose.JSONWebKey, error) {
	return nil, errUnsupported
}

func (s *memoryStorage) ValidateJWTProfileScopes(_ context.Context, _ string, scopes []string) ([]string, error) {
	return scopes, nil
}

func (s *memoryStorage) Health(context.Context) error { return nil }

func (s *memoryStorage) CompleteAuthRequest(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	authRequest, ok := s.authRequests[id]
	if !ok {
		return fmt.Errorf("auth request %q not found", id)
	}
	authRequest.done = true
	return nil
}

func (s *memoryStorage) setUserinfo(userinfo *oidc.UserInfo, scopes []string) {
	userinfo.Subject = s.user.id
	if slices.Contains(scopes, oidc.ScopeProfile) {
		userinfo.Name = s.user.name
		// Spike-only placeholder. Phase 3 must define preferred_username explicitly.
		userinfo.PreferredUsername = s.user.id
	}
	if slices.Contains(scopes, oidc.ScopeEmail) {
		userinfo.Email = s.user.email
		userinfo.EmailVerified = true
	}
}

type memorySigningKey struct {
	id  string
	key *rsa.PrivateKey
}

func (k memorySigningKey) SignatureAlgorithm() jose.SignatureAlgorithm { return jose.RS256 }
func (k memorySigningKey) Key() any                                    { return k.key }
func (k memorySigningKey) ID() string                                  { return k.id }

type memoryKey struct {
	id  string
	key *rsa.PublicKey
}

func (k memoryKey) ID() string                         { return k.id }
func (k memoryKey) Algorithm() jose.SignatureAlgorithm { return jose.RS256 }
func (k memoryKey) Use() string                        { return "sig" }
func (k memoryKey) Key() any                           { return k.key }

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
