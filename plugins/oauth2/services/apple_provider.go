package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/util"
)

const (
	appleIssuer  = "https://appleid.apple.com"
	appleJWKSURL = "https://appleid.apple.com/auth/keys"

	appleClientSecretTTL = 5 * time.Minute
	appleJWKSCacheTTL    = time.Hour
	appleMaxJWKSBytes    = 1 << 20
)

var _ types.OAuth2Provider = (*AppleProvider)(nil)

// AppleProvider implements OAuth2Provider for Sign in with Apple
type AppleProvider struct {
	*BaseProvider

	clientID   string
	teamID     string
	keyID      string
	privateKey []byte

	jwksURL string
	now     func() time.Time

	mu        sync.Mutex
	keys      jwk.Set
	keysUntil time.Time
}

var appleDefaults = ProviderDefaults{
	Endpoint: appleEndpoint(),
	Scopes: []string{
		"name",
		"email",
	},
}

func appleEndpoint() oauth2.Endpoint {
	endpoint := endpoints.Apple
	// Apple only advertises client_secret_post. Leaving the auth style unset
	// makes the first exchange probe header auth and fail.
	endpoint.AuthStyle = oauth2.AuthStyleInParams
	return endpoint
}

// NewAppleProvider creates a new Apple OAuth2 provider. private_key is the
// contents of the .p8 key from the Apple Developer portal; literal "\n"
// sequences are accepted so it can be supplied through an environment variable.
func NewAppleProvider(cfg types.ProviderConfig, opts *ProviderOptions) *AppleProvider {
	return &AppleProvider{
		BaseProvider: NewBaseProvider("apple", ResolveOAuth2Config(cfg, appleDefaults), opts, cfg.PKCEEnabled()),
		clientID:     cfg.ClientID,
		teamID:       cfg.TeamID,
		keyID:        cfg.KeyID,
		privateKey:   []byte(normalizePEM(cfg.PrivateKey)),
		jwksURL:      appleJWKSURL,
		now:          time.Now,
	}
}

// GetAuthURL adds response_mode=form_post, which Apple requires whenever any
// scope is requested
func (p *AppleProvider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	authOpts := make([]oauth2.AuthCodeOption, 0, len(opts)+1)
	authOpts = append(authOpts, opts...)
	authOpts = append(authOpts, oauth2.SetAuthURLParam("response_mode", "form_post"))

	return p.BaseProvider.GetAuthURL(state, authOpts...)
}

// Exchange exchanges the authorization code using a freshly minted client
// secret. The secret goes on a copy of the config so concurrent exchanges do
// not race on shared state.
func (p *AppleProvider) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	secret, err := p.clientSecret()
	if err != nil {
		return nil, err
	}

	config := *p.GetConfig()
	config.ClientSecret = secret

	token, err := config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange Apple authorization code: %w", err)
	}

	return token, nil
}

// GetUserInfo verifies the id_token and normalizes its claims. Apple has no
// userinfo endpoint, so no user request is made here.
func (p *AppleProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, fmt.Errorf("failed to get Apple user info: token response is missing id_token")
	}

	claims, err := p.verifyIDToken(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get Apple user info: %w", err)
	}

	subject, ok := claims.Subject()
	if !ok || subject == "" {
		return nil, fmt.Errorf("failed to get Apple user info: id_token is missing sub claim")
	}

	emailVerified, err := claimBool(claims, "email_verified")
	if err != nil {
		return nil, fmt.Errorf("failed to get Apple user info: %w", err)
	}

	raw, err := claimsToMap(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to get Apple user info: %w", err)
	}

	email, _ := claimString(claims, "email")

	return &types.UserInfo{
		ProviderAccountID: subject,
		Email:             util.NormalizeEmail(email),
		EmailVerified:     emailVerified,
		// Apple returns the name out-of-band on first consent only, so it is
		// never available from the id_token.
		Name:    "",
		Picture: "",
		Raw:     raw,
	}, nil
}

func (p *AppleProvider) verifyIDToken(ctx context.Context, rawIDToken string) (jwt.Token, error) {
	keys, err := p.keySet(ctx)
	if err != nil {
		return nil, err
	}

	claims, err := jwt.Parse([]byte(rawIDToken), jwt.WithKeySet(keys))
	if err != nil {
		// Apple rotates its signing keys, so a warm cache can be missing the kid
		// a fresh token was signed with. Refetch once before failing.
		refreshed, refreshErr := p.refreshKeySet(ctx)
		if refreshErr != nil {
			return nil, fmt.Errorf("failed to verify id_token signature: %w", errors.Join(err, refreshErr))
		}

		claims, err = jwt.Parse([]byte(rawIDToken), jwt.WithKeySet(refreshed))
		if err != nil {
			return nil, fmt.Errorf("failed to verify id_token signature: %w", err)
		}
	}

	if err := jwt.Validate(claims, jwt.WithIssuer(appleIssuer), jwt.WithAudience(p.clientID)); err != nil {
		return nil, fmt.Errorf("failed to validate id_token claims: %w", err)
	}

	return claims, nil
}

// keySet returns Apple's signing keys, refreshing them at most once per
// appleJWKSCacheTTL. Apple rotates the set, so keys are matched by kid at parse
// time instead of being pinned.
func (p *AppleProvider) keySet(ctx context.Context) (jwk.Set, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.keys != nil && p.now().Before(p.keysUntil) {
		return p.keys, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := p.opts.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch JWKS: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, appleMaxJWKSBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	keys, err := jwk.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	p.keys = keys
	p.keysUntil = p.now().Add(appleJWKSCacheTTL)

	return keys, nil
}

// refreshKeySet drops the cached keys and fetches them again, so a key rotation
// does not fail verification until the cache happens to expire.
func (p *AppleProvider) refreshKeySet(ctx context.Context) (jwk.Set, error) {
	p.mu.Lock()
	p.keys = nil
	p.keysUntil = time.Time{}
	p.mu.Unlock()

	return p.keySet(ctx)
}

// clientSecret builds the ES256 JWT Apple accepts as a client secret. Apple
// rejects an expiry more than six months out; five minutes keeps a leaked
// secret useless.
func (p *AppleProvider) clientSecret() (string, error) {
	key, err := jwk.ParseKey(p.privateKey, jwk.WithPEM(true))
	if err != nil {
		return "", fmt.Errorf("failed to parse Apple private key: %w", err)
	}

	if err := key.Set(jwk.KeyIDKey, p.keyID); err != nil {
		return "", fmt.Errorf("failed to set Apple key id: %w", err)
	}

	now := p.now()
	claims := jwt.New()

	values := map[string]any{
		jwt.IssuerKey:     p.teamID,
		jwt.SubjectKey:    p.clientID,
		jwt.AudienceKey:   []string{appleIssuer},
		jwt.IssuedAtKey:   now,
		jwt.ExpirationKey: now.Add(appleClientSecretTTL),
	}
	for name, value := range values {
		if err := claims.Set(name, value); err != nil {
			return "", fmt.Errorf("failed to set %s claim on client secret: %w", name, err)
		}
	}

	signed, err := jwt.Sign(claims, jwt.WithKey(jwa.ES256(), key))
	if err != nil {
		return "", fmt.Errorf("failed to sign client secret: %w", err)
	}

	return string(signed), nil
}

// claimBool reads a claim Apple may encode as either a boolean or the strings
// "true"/"false". Decoding into a bool loses these claims silently.
func claimBool(claims jwt.Token, name string) (bool, error) {
	var raw any
	if err := claims.Get(name, &raw); err != nil {
		return false, nil
	}

	switch typed := raw.(type) {
	case nil:
		return false, nil
	case bool:
		return typed, nil
	case string:
		parsed, err := strconv.ParseBool(typed)
		if err != nil {
			return false, fmt.Errorf("claim %s is not a valid boolean: %w", name, err)
		}
		return parsed, nil
	default:
		return false, fmt.Errorf("claim %s has unexpected type %T", name, raw)
	}
}

func claimString(claims jwt.Token, name string) (string, bool) {
	var str string
	if err := claims.Get(name, &str); err != nil {
		return "", false
	}

	return str, true
}

func claimsToMap(claims jwt.Token) (map[string]any, error) {
	encoded, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to encode claims: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	return raw, nil
}

func normalizePEM(pem string) string {
	return strings.TrimSpace(strings.ReplaceAll(pem, `\n`, "\n"))
}
