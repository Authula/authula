package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/types"
)

const (
	testAppleClientID    = "com.example.authula"
	testAppleTeamID      = "TEAM123456"
	testAppleKeyID       = "KEY1234567"
	testAppleRedirectURL = "https://authula.example.com/oauth2/apple/callback"
	testAppleSubject     = "001234.abcdef.1234"
)

func newTestAppleProvider(t *testing.T) *AppleProvider {
	t.Helper()

	return NewAppleProvider(types.ProviderConfig{
		ClientID:    testAppleClientID,
		RedirectURL: testAppleRedirectURL,
		TeamID:      testAppleTeamID,
		KeyID:       testAppleKeyID,
	}, nil)
}

func newRSAKeyPair(t *testing.T, kid string) (*rsa.PrivateKey, jwk.Key) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signingKey, err := jwk.Import(priv)
	require.NoError(t, err)
	require.NoError(t, signingKey.Set(jwk.KeyIDKey, kid))

	return priv, signingKey
}

func marshalJWKS(t *testing.T, kid string, pub *rsa.PublicKey) []byte {
	t.Helper()

	publicKey, err := jwk.Import(pub)
	require.NoError(t, err)
	require.NoError(t, publicKey.Set(jwk.KeyIDKey, kid))
	require.NoError(t, publicKey.Set(jwk.AlgorithmKey, jwa.RS256()))

	set := jwk.NewSet()
	require.NoError(t, set.AddKey(publicKey))

	body, err := json.Marshal(set)
	require.NoError(t, err)

	return body
}

func newJWKSServer(t *testing.T, body []byte, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	hits := new(atomic.Int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return server, hits
}

func appleClaims(now time.Time) map[string]any {
	return map[string]any{
		jwt.IssuerKey:     appleIssuer,
		jwt.SubjectKey:    testAppleSubject,
		jwt.AudienceKey:   []string{testAppleClientID},
		jwt.IssuedAtKey:   now,
		jwt.ExpirationKey: now.Add(time.Hour),
		"email":           "user@example.com",
		"email_verified":  "true",
	}
}

func cloneClaims(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for name, value := range in {
		out[name] = value
	}

	return out
}

func signIDToken(t *testing.T, key jwk.Key, claims map[string]any) string {
	t.Helper()

	token := jwt.New()
	for name, value := range claims {
		require.NoError(t, token.Set(name, value))
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), key))
	require.NoError(t, err)

	return string(signed)
}

func oauth2Token(idToken string) *oauth2.Token {
	return new(oauth2.Token).WithExtra(map[string]any{"id_token": idToken})
}

func TestAppleProvider_GetAuthURL(t *testing.T) {
	t.Parallel()

	provider := newTestAppleProvider(t)

	authURL := provider.GetAuthURL("state-abc", oauth2.SetAuthURLParam("prompt", "consent"))

	is := assert.New(t)
	is.Contains(authURL, "response_mode=form_post")
	is.Contains(authURL, "name")
	is.Contains(authURL, "email")

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)

	query := parsed.Query()
	is.Equal("form_post", query.Get("response_mode"))
	is.Equal("name email", query.Get("scope"))
	is.Equal(testAppleClientID, query.Get("client_id"))
	is.Equal(testAppleRedirectURL, query.Get("redirect_uri"))
	is.Equal("state-abc", query.Get("state"))
	is.Equal("consent", query.Get("prompt"))
	is.True(provider.RequiresPKCE(), "apple must require PKCE by default")
}

// newAppleSigningKey returns a P-256 .p8 key in PEM form and its public half,
// which is what Apple issues in the developer portal.
func newAppleSigningKey(t *testing.T) (string, *ecdsa.PublicKey) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), &key.PublicKey
}

func TestAppleProvider_Exchange(t *testing.T) {
	t.Parallel()

	t.Run("mints an ES256 client secret and posts it to the token endpoint", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)
		must := require.New(t)

		privateKeyPEM, publicKey := newAppleSigningKey(t)

		forms := make(chan url.Values, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			forms <- r.PostForm

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"apple-access","token_type":"Bearer","expires_in":3600,"id_token":"header.payload.signature"}`))
		}))
		t.Cleanup(server.Close)

		provider := NewAppleProvider(types.ProviderConfig{
			ClientID:    testAppleClientID,
			RedirectURL: testAppleRedirectURL,
			TeamID:      testAppleTeamID,
			KeyID:       testAppleKeyID,
			PrivateKey:  privateKeyPEM,
			TokenURL:    server.URL,
		}, nil)

		token, err := provider.Exchange(t.Context(), "auth-code")
		must.NoError(err)
		must.NotNil(token)

		form := <-forms
		is.Equal("authorization_code", form.Get("grant_type"))
		is.Equal("auth-code", form.Get("code"))
		is.Equal(testAppleClientID, form.Get("client_id"))
		is.Empty(form.Get("client_secret_basic"), "Apple only supports client_secret_post")

		secret := form.Get("client_secret")
		must.NotEmpty(secret)

		message, err := jws.Parse([]byte(secret))
		must.NoError(err)
		must.Len(message.Signatures(), 1)

		kid, ok := message.Signatures()[0].ProtectedHeaders().KeyID()
		is.True(ok)
		is.Equal(testAppleKeyID, kid)

		claims, err := jwt.Parse([]byte(secret), jwt.WithKey(jwa.ES256(), publicKey))
		must.NoError(err)

		issuer, ok := claims.Issuer()
		is.True(ok)
		is.Equal(testAppleTeamID, issuer)

		subject, ok := claims.Subject()
		is.True(ok)
		is.Equal(testAppleClientID, subject)

		is.Equal("apple-access", token.AccessToken)
		is.Equal("header.payload.signature", token.Extra("id_token"))
	})

	t.Run("wraps a token endpoint failure", func(t *testing.T) {
		t.Parallel()

		privateKeyPEM, _ := newAppleSigningKey(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
		}))
		t.Cleanup(server.Close)

		provider := NewAppleProvider(types.ProviderConfig{
			ClientID:    testAppleClientID,
			RedirectURL: testAppleRedirectURL,
			TeamID:      testAppleTeamID,
			KeyID:       testAppleKeyID,
			PrivateKey:  privateKeyPEM,
			TokenURL:    server.URL,
		}, nil)

		token, err := provider.Exchange(t.Context(), "auth-code")

		assert.Nil(t, token)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to exchange Apple authorization code")
	})

	t.Run("an unparseable private key fails before the request", func(t *testing.T) {
		t.Parallel()

		hits := new(atomic.Int32)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		provider := NewAppleProvider(types.ProviderConfig{
			ClientID:    testAppleClientID,
			RedirectURL: testAppleRedirectURL,
			TeamID:      testAppleTeamID,
			KeyID:       testAppleKeyID,
			PrivateKey:  "not a pem block",
			TokenURL:    server.URL,
		}, nil)

		token, err := provider.Exchange(t.Context(), "auth-code")

		assert.Nil(t, token)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse Apple private key")
		assert.Equal(t, int32(0), hits.Load())
	})
}

func TestAppleProvider_GetUserInfo(t *testing.T) {
	t.Parallel()

	priv, signingKey := newRSAKeyPair(t, testAppleKeyID)
	jwks := marshalJWKS(t, testAppleKeyID, &priv.PublicKey)
	server, hits := newJWKSServer(t, jwks, http.StatusOK)

	provider := newTestAppleProvider(t)
	provider.jwksURL = server.URL

	idToken := signIDToken(t, signingKey, appleClaims(time.Now()))

	info, err := provider.GetUserInfo(t.Context(), oauth2Token(idToken))
	require.NoError(t, err)
	require.NotNil(t, info)

	is := assert.New(t)
	is.Equal(testAppleSubject, info.ProviderAccountID)
	is.Equal("user@example.com", info.Email)
	is.True(info.EmailVerified)
	is.Empty(info.Name)
	is.Empty(info.Picture)
	is.Equal("true", info.Raw["email_verified"])
	is.Equal(int32(1), hits.Load())
}

func TestAppleProvider_GetUserInfo_EmailVerified(t *testing.T) {
	t.Parallel()

	priv, signingKey := newRSAKeyPair(t, testAppleKeyID)
	server, _ := newJWKSServer(t, marshalJWKS(t, testAppleKeyID, &priv.PublicKey), http.StatusOK)

	now := time.Now()

	withClaim := func(value any) map[string]any {
		claims := cloneClaims(appleClaims(now))
		claims["email_verified"] = value

		return claims
	}
	withoutClaim := func() map[string]any {
		claims := cloneClaims(appleClaims(now))
		delete(claims, "email_verified")

		return claims
	}

	tests := []struct {
		name    string
		claims  map[string]any
		want    bool
		wantErr string
	}{
		{
			name:   "string true is verified",
			claims: withClaim("true"),
			want:   true,
		},
		{
			name:   "bool false is not verified",
			claims: withClaim(false),
		},
		{
			name:   "bool true is verified",
			claims: withClaim(true),
			want:   true,
		},
		{
			name:   "absent claim is not verified",
			claims: withoutClaim(),
		},
		{
			name:    "unparseable string is an error",
			claims:  withClaim("yes"),
			wantErr: "claim email_verified is not a valid boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := newTestAppleProvider(t)
			provider.jwksURL = server.URL

			info, err := provider.GetUserInfo(t.Context(), oauth2Token(signIDToken(t, signingKey, tt.claims)))

			if tt.wantErr != "" {
				assert.Nil(t, info)
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to get Apple user info")
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, tt.want, info.EmailVerified)
		})
	}
}

func TestAppleProvider_GetUserInfo_Errors(t *testing.T) {
	t.Parallel()

	priv, signingKey := newRSAKeyPair(t, testAppleKeyID)
	jwks := marshalJWKS(t, testAppleKeyID, &priv.PublicKey)
	okServer, _ := newJWKSServer(t, jwks, http.StatusOK)
	malformedServer, _ := newJWKSServer(t, []byte("not a jwks"), http.StatusOK)
	failingServer, _ := newJWKSServer(t, nil, http.StatusInternalServerError)

	// Same kid as the published key, so verification fails on the signature
	// rather than on key lookup.
	_, rogueKey := newRSAKeyPair(t, testAppleKeyID)

	okProvider := func(t *testing.T) *AppleProvider {
		p := newTestAppleProvider(t)
		p.jwksURL = okServer.URL

		return p
	}

	now := time.Now()

	tests := []struct {
		name     string
		provider func(*testing.T) *AppleProvider
		token    func(*testing.T) *oauth2.Token
		wantErr  string
	}{
		{
			name:     "missing id_token",
			provider: okProvider,
			token: func(*testing.T) *oauth2.Token {
				return new(oauth2.Token)
			},
			wantErr: "missing id_token",
		},
		{
			name:     "bad signature",
			provider: okProvider,
			token: func(t *testing.T) *oauth2.Token {
				return oauth2Token(signIDToken(t, rogueKey, appleClaims(now)))
			},
			wantErr: "failed to verify id_token signature",
		},
		{
			name:     "wrong issuer",
			provider: okProvider,
			token: func(t *testing.T) *oauth2.Token {
				claims := appleClaims(now)
				claims[jwt.IssuerKey] = "https://evil.example.com"

				return oauth2Token(signIDToken(t, signingKey, claims))
			},
			wantErr: "failed to validate id_token claims",
		},
		{
			name:     "wrong audience",
			provider: okProvider,
			token: func(t *testing.T) *oauth2.Token {
				claims := appleClaims(now)
				claims[jwt.AudienceKey] = []string{"com.example.other"}

				return oauth2Token(signIDToken(t, signingKey, claims))
			},
			wantErr: "failed to validate id_token claims",
		},
		{
			name:     "expired token",
			provider: okProvider,
			token: func(t *testing.T) *oauth2.Token {
				claims := appleClaims(now.Add(-2 * time.Hour))
				claims[jwt.ExpirationKey] = now.Add(-time.Hour)

				return oauth2Token(signIDToken(t, signingKey, claims))
			},
			// jwt.Parse validates exp itself, so the failure surfaces through
			// the signature-verification wrapper rather than the claims one.
			wantErr: "token is expired",
		},
		{
			name:     "missing subject",
			provider: okProvider,
			token: func(t *testing.T) *oauth2.Token {
				claims := cloneClaims(appleClaims(now))
				delete(claims, jwt.SubjectKey)

				return oauth2Token(signIDToken(t, signingKey, claims))
			},
			wantErr: "missing sub claim",
		},
		{
			name: "jwks endpoint error",
			provider: func(t *testing.T) *AppleProvider {
				p := newTestAppleProvider(t)
				p.jwksURL = failingServer.URL

				return p
			},
			token: func(t *testing.T) *oauth2.Token {
				return oauth2Token(signIDToken(t, signingKey, appleClaims(now)))
			},
			wantErr: "failed to fetch JWKS",
		},
		{
			name: "malformed jwks body",
			provider: func(t *testing.T) *AppleProvider {
				p := newTestAppleProvider(t)
				p.jwksURL = malformedServer.URL

				return p
			},
			token: func(t *testing.T) *oauth2.Token {
				return oauth2Token(signIDToken(t, signingKey, appleClaims(now)))
			},
			wantErr: "failed to parse JWKS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info, err := tt.provider(t).GetUserInfo(t.Context(), tt.token(t))

			is := assert.New(t)
			is.Nil(info)
			is.Error(err)
			is.ErrorContains(err, "failed to get Apple user info")
			is.ErrorContains(err, tt.wantErr)
		})
	}
}

func TestAppleProvider_GetUserInfo_CachesJWKS(t *testing.T) {
	t.Parallel()

	priv, signingKey := newRSAKeyPair(t, testAppleKeyID)
	jwks := marshalJWKS(t, testAppleKeyID, &priv.PublicKey)
	server, hits := newJWKSServer(t, jwks, http.StatusOK)

	start := time.Now()
	clock := new(atomic.Int64)
	clock.Store(start.Unix())

	provider := newTestAppleProvider(t)
	provider.jwksURL = server.URL
	provider.now = func() time.Time { return time.Unix(clock.Load(), 0) }

	token := oauth2Token(signIDToken(t, signingKey, appleClaims(start)))

	_, err := provider.GetUserInfo(t.Context(), token)
	require.NoError(t, err)
	_, err = provider.GetUserInfo(t.Context(), token)
	require.NoError(t, err)

	is := assert.New(t)
	is.Equal(int32(1), hits.Load(), "the second verification must reuse the cached key set")

	clock.Store(start.Add(2 * appleJWKSCacheTTL).Unix())

	_, err = provider.GetUserInfo(t.Context(), token)
	require.NoError(t, err)
	is.Equal(int32(2), hits.Load(), "the cache must be refreshed once the TTL has passed")
}

func TestAppleProvider_GetUserInfo_RefreshesKeysOnRotation(t *testing.T) {
	t.Parallel()

	t.Run("an unknown kid is recovered with exactly one refetch", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)
		must := require.New(t)

		const rotatedKeyID = "KEYROTATED"

		stalePriv, _ := newRSAKeyPair(t, "KEYSTALE")
		rotatedPriv, rotatedSigningKey := newRSAKeyPair(t, rotatedKeyID)

		// The first response is the set already in the cache: it does not
		// contain the kid the token was signed with. The second is the rotated
		// set Apple would be serving by then.
		responses := [][]byte{
			marshalJWKS(t, "KEYSTALE", &stalePriv.PublicKey),
			marshalJWKS(t, rotatedKeyID, &rotatedPriv.PublicKey),
		}

		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			n := int(hits.Add(1))
			body := responses[len(responses)-1]
			if n <= len(responses) {
				body = responses[n-1]
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		}))
		t.Cleanup(server.Close)

		provider := newTestAppleProvider(t)
		provider.jwksURL = server.URL

		// Warm the cache with the stale set, the way a previous login would.
		_, err := provider.keySet(t.Context())
		must.NoError(err)
		is.Equal(int32(1), hits.Load())

		info, err := provider.GetUserInfo(t.Context(), oauth2Token(signIDToken(t, rotatedSigningKey, appleClaims(time.Now()))))
		must.NoError(err)
		must.NotNil(info)
		is.Equal(testAppleSubject, info.ProviderAccountID)
		is.Equal(int32(2), hits.Load(), "the unknown kid must trigger exactly one refetch")
	})

	t.Run("a persistent signature failure refetches once and fails", func(t *testing.T) {
		t.Parallel()

		// Same kid as the published key, so the retry fails on the signature
		// rather than on key lookup and cannot be repaired by a refetch.
		priv, _ := newRSAKeyPair(t, testAppleKeyID)
		_, rogueKey := newRSAKeyPair(t, testAppleKeyID)

		server, hits := newJWKSServer(t, marshalJWKS(t, testAppleKeyID, &priv.PublicKey), http.StatusOK)

		provider := newTestAppleProvider(t)
		provider.jwksURL = server.URL

		info, err := provider.GetUserInfo(t.Context(), oauth2Token(signIDToken(t, rogueKey, appleClaims(time.Now()))))

		assert.Nil(t, info)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to verify id_token signature")
		assert.Equal(t, int32(2), hits.Load(), "a failed refetch must not loop")
	})

	t.Run("a refetch failure reports both the verification and fetch errors", func(t *testing.T) {
		t.Parallel()

		stalePriv, _ := newRSAKeyPair(t, "KEYSTALE")
		_, unknownSigningKey := newRSAKeyPair(t, "KEYROTATED")
		staleJWKS := marshalJWKS(t, "KEYSTALE", &stalePriv.PublicKey)

		// The warm-up fetch succeeds with a set that lacks the token's kid; the
		// refetch the rotation would need is unavailable.
		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if hits.Add(1) == 1 {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(staleJWKS)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)

		provider := newTestAppleProvider(t)
		provider.jwksURL = server.URL

		_, err := provider.keySet(t.Context())
		require.NoError(t, err)

		info, err := provider.GetUserInfo(t.Context(), oauth2Token(signIDToken(t, unknownSigningKey, appleClaims(time.Now()))))

		assert.Nil(t, info)
		require.Error(t, err)
		// Both leaves must survive: the kid only appears in the parse error and
		// the status only in the refresh error, so seeing both proves the join.
		assert.ErrorContains(t, err, "failed to verify id_token signature")
		assert.ErrorContains(t, err, "KEYROTATED")
		assert.ErrorContains(t, err, "failed to fetch JWKS")
		assert.ErrorContains(t, err, "status 500")
		assert.Equal(t, int32(2), hits.Load())
	})
}

func TestAppleProvider_clientSecret(t *testing.T) {
	t.Parallel()

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)

	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	now := time.Now().UTC().Truncate(time.Second)

	provider := NewAppleProvider(types.ProviderConfig{
		ClientID:    testAppleClientID,
		RedirectURL: testAppleRedirectURL,
		TeamID:      testAppleTeamID,
		KeyID:       testAppleKeyID,
		PrivateKey:  privateKeyPEM,
	}, nil)
	provider.now = func() time.Time { return now }

	secret, err := provider.clientSecret()
	require.NoError(t, err)
	require.NotEmpty(t, secret)

	message, err := jws.Parse([]byte(secret))
	require.NoError(t, err)
	require.Len(t, message.Signatures(), 1)

	is := assert.New(t)

	headers := message.Signatures()[0].ProtectedHeaders()
	kid, ok := headers.KeyID()
	is.True(ok)
	is.Equal(testAppleKeyID, kid)

	alg, ok := headers.Algorithm()
	is.True(ok)
	is.Equal(jwa.ES256(), alg)

	claims, err := jwt.Parse([]byte(secret), jwt.WithKey(jwa.ES256(), &ecKey.PublicKey))
	require.NoError(t, err)

	issuer, ok := claims.Issuer()
	is.True(ok)
	is.Equal(testAppleTeamID, issuer)

	subject, ok := claims.Subject()
	is.True(ok)
	is.Equal(testAppleClientID, subject)

	audience, ok := claims.Audience()
	is.True(ok)
	is.Equal([]string{appleIssuer}, audience)

	issuedAt, ok := claims.IssuedAt()
	is.True(ok)
	is.True(now.Equal(issuedAt))

	expiration, ok := claims.Expiration()
	is.True(ok)
	is.True(now.Add(appleClientSecretTTL).Equal(expiration))
	is.Equal(5*time.Minute, expiration.Sub(issuedAt))
	is.LessOrEqual(expiration.Sub(now), 180*24*time.Hour, "Apple rejects a client secret valid for more than six months")
}

func TestAppleProvider_clientSecret_RejectsBadKey(t *testing.T) {
	t.Parallel()

	provider := NewAppleProvider(types.ProviderConfig{
		ClientID:    testAppleClientID,
		RedirectURL: testAppleRedirectURL,
		TeamID:      testAppleTeamID,
		KeyID:       testAppleKeyID,
		PrivateKey:  "not a pem block",
	}, nil)

	secret, err := provider.clientSecret()

	require.Error(t, err)
	require.Empty(t, secret)
	require.ErrorContains(t, err, "failed to parse Apple private key")
}

func TestNormalizePEM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "literal backslash n sequences become newlines",
			in:   `-----BEGIN PRIVATE KEY-----\nMIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg\n-----END PRIVATE KEY-----`,
			want: "-----BEGIN PRIVATE KEY-----\nMIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg\n-----END PRIVATE KEY-----",
		},
		{
			name: "real multiline pem passes through trimmed",
			in:   "\n-----BEGIN PRIVATE KEY-----\nMIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg\n-----END PRIVATE KEY-----\n\n",
			want: "-----BEGIN PRIVATE KEY-----\nMIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg\n-----END PRIVATE KEY-----",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			is := assert.New(t)
			is.Equal(tt.want, normalizePEM(tt.in))
		})
	}
}
