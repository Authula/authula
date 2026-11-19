package integration

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/util"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

const (
	requestContextMarkerKey = "phase0_request_context_marker"
	requestContextMarker    = "request-context-propagation-proven"
)

type spikeHarness struct {
	t           *testing.T
	origin      string
	issuer      string
	redirectURI string
	storage     *memoryStorage
	provider    *op.Provider
	router      *authula.Router
	server      *httptest.Server
	client      *http.Client
	routeMeta   map[string]map[string]any
	hookMu      sync.Mutex
	beforeHooks map[string]int
	afterHooks  map[string]int
}

func newSpikeHarness(t *testing.T) *spikeHarness {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	origin := "http://" + listener.Addr().String()
	h := &spikeHarness{
		t:           t,
		origin:      origin,
		issuer:      origin + spikeBasePath + spikeOIDCPath,
		redirectURI: origin + "/rp/callback",
		beforeHooks: make(map[string]int),
		afterHooks:  make(map[string]int),
	}

	h.storage, err = newMemoryStorage(h.redirectURI)
	require.NoError(t, err)

	cryptoKey := sha256.Sum256([]byte("phase-0-fixed-32-byte-crypto-key"))
	h.provider, err = op.NewProvider(
		&op.Config{
			CryptoKey:             cryptoKey,
			CodeMethodS256:        true,
			AuthMethodPost:        true,
			GrantTypeRefreshToken: false,
			SupportedScopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail},
			SupportedClaims:       []string{"sub", "aud", "exp", "iat", "iss", "auth_time", "nonce", "at_hash", "c_hash", "email", "email_verified", "preferred_username", "name"},
		},
		h.storage,
		op.StaticIssuer(h.issuer),
		op.WithAllowInsecure(),
		op.WithCustomTokenEndpoint(op.NewEndpoint("token")),
		op.WithCustomIntrospectionEndpoint(op.NewEndpoint("introspect")),
	)
	require.NoError(t, err)

	h.router = authula.NewRouter(
		&models.Config{BasePath: spikeBasePath},
		&internaltests.LevelRecordingLogger{},
		nil,
	)

	issuerInterceptor := op.NewIssuerInterceptor(h.provider.IssuerFromRequest)
	withIssuer := func(next http.HandlerFunc) http.Handler {
		return issuerInterceptor.Handler(next)
	}

	loginHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCtx, ok := models.GetRequestContext(r.Context())
		if !ok {
			http.Error(w, "request context missing", http.StatusInternalServerError)
			return
		}
		marker, _ := reqCtx.Values[requestContextMarkerKey].(string)
		if marker != requestContextMarker {
			http.Error(w, "request context marker missing", http.StatusInternalServerError)
			return
		}
		plugins := util.ReadStringSliceFromMetadata(reqCtx.Route.Metadata, "plugins")
		if !slices.Contains(plugins, models.PluginOAuth2.String()) {
			http.Error(w, "route metadata missing oauth2 plugin", http.StatusInternalServerError)
			return
		}
		requestID := r.URL.Query().Get("oauth2_request_id")
		if requestID == "" {
			http.Error(w, "oauth2_request_id missing", http.StatusBadRequest)
			return
		}
		if err := h.storage.CompleteAuthRequest(requestID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("X-Authula-Request-Context", marker)
		w.Header().Set("X-Authula-Route-Plugin", strings.Join(plugins, ","))
		http.Redirect(w, r, op.AuthCallbackURL(h.provider)(r.Context(), requestID), http.StatusFound)
	})

	routes := []models.Route{
		{
			Method: http.MethodGet,
			Path:   spikeOIDCPath + "/.well-known/openid-configuration",
			Handler: withIssuer(func(w http.ResponseWriter, r *http.Request) {
				op.Discover(w, op.CreateDiscoveryConfig(r.Context(), h.provider, h.storage))
			}),
		},
		{
			Method:  http.MethodGet,
			Path:    spikeOIDCPath + "/keys",
			Handler: withIssuer(func(w http.ResponseWriter, r *http.Request) { op.Keys(w, r, h.storage) }),
		},
		{
			Method:  http.MethodGet,
			Path:    spikeOIDCPath + "/authorize",
			Handler: withIssuer(func(w http.ResponseWriter, r *http.Request) { op.Authorize(w, r, h.provider) }),
		},
		{
			Method:  http.MethodGet,
			Path:    spikeOIDCPath + "/authorize/callback",
			Handler: withIssuer(func(w http.ResponseWriter, r *http.Request) { op.AuthorizeCallback(w, r, h.provider) }),
		},
		{
			Method:  http.MethodGet,
			Path:    spikeOIDCPath + "/login",
			Handler: withIssuer(loginHandler),
		},
		{
			Method:  http.MethodPost,
			Path:    spikeOIDCPath + "/token",
			Handler: withIssuer(func(w http.ResponseWriter, r *http.Request) { op.Exchange(w, r, h.provider) }),
		},
		{
			Method:  http.MethodGet,
			Path:    spikeOIDCPath + "/userinfo",
			Handler: withIssuer(func(w http.ResponseWriter, r *http.Request) { op.Userinfo(w, r, h.provider) }),
		},
	}

	h.routeMeta = make(map[string]map[string]any, len(routes))
	for _, route := range routes {
		h.routeMeta[route.Method+":"+route.Path] = map[string]any{
			"plugins": []string{models.PluginOAuth2.String()},
		}
	}
	h.router.SetRouteMetadataFromConfig(h.routeMeta)

	h.router.RegisterHook(models.Hook{
		Stage:    models.HookOnRequest,
		PluginID: models.PluginOAuth2.String(),
		Order:    0,
		Handler: func(ctx *models.RequestContext) error {
			h.recordHook(h.beforeHooks, ctx.Path)
			ctx.Values[requestContextMarkerKey] = requestContextMarker
			return nil
		},
	})
	h.router.RegisterHook(models.Hook{
		Stage:    models.HookAfter,
		PluginID: models.PluginOAuth2.String(),
		Order:    0,
		Handler: func(ctx *models.RequestContext) error {
			h.recordHook(h.afterHooks, ctx.Path)
			if ctx.Path == spikeBasePath+spikeOIDCPath+"/token" {
				ctx.ResponseWriter.Header().Set("Cache-Control", "no-store")
			}
			return nil
		},
	})

	h.router.RegisterRoutes(routes)

	h.server = &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: h.router.Handler()},
	}
	h.server.Start()
	t.Cleanup(h.server.Close)

	h.client = &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return h
}

func (h *spikeHarness) oidcPath(suffix string) string {
	return spikeBasePath + spikeOIDCPath + suffix
}

func (h *spikeHarness) recordHook(counter map[string]int, path string) {
	h.hookMu.Lock()
	defer h.hookMu.Unlock()
	counter[path]++
}

func (h *spikeHarness) hookCount(counter map[string]int, path string) int {
	h.hookMu.Lock()
	defer h.hookMu.Unlock()
	return counter[path]
}

func (h *spikeHarness) do(method, path string, body io.Reader, contentType string, headers http.Header) *http.Response {
	h.t.Helper()

	req, err := http.NewRequest(method, h.origin+path, body)
	require.NoError(h.t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := h.client.Do(req)
	require.NoError(h.t, err)
	return resp
}

func (h *spikeHarness) get(path string) *http.Response {
	h.t.Helper()
	return h.do(http.MethodGet, path, nil, "", nil)
}

func (h *spikeHarness) postForm(path string, form url.Values) *http.Response {
	h.t.Helper()
	return h.do(http.MethodPost, path, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", nil)
}

func (h *spikeHarness) getJSON(path string, target any) {
	h.t.Helper()

	resp := h.get(path)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(h.t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(h.t, err)
	require.NoError(h.t, json.Unmarshal(body, target))
}

func TestDiscoveryPinsProjectPathsAndRecordsOverAdvertising(t *testing.T) {
	h := newSpikeHarness(t)

	var discovery oidc.DiscoveryConfiguration
	h.getJSON(h.oidcPath("/.well-known/openid-configuration"), &discovery)

	assert.Equal(t, h.issuer, discovery.Issuer)
	assert.Equal(t, h.issuer+"/authorize", discovery.AuthorizationEndpoint)
	assert.Equal(t, h.issuer+"/token", discovery.TokenEndpoint)
	assert.Equal(t, h.issuer+"/introspect", discovery.IntrospectionEndpoint)
	assert.Equal(t, h.issuer+"/userinfo", discovery.UserinfoEndpoint)
	assert.Equal(t, h.issuer+"/keys", discovery.JwksURI)
	assert.Equal(t, []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail}, discovery.ScopesSupported)
	assert.Equal(t, []string{"RS256"}, discovery.IDTokenSigningAlgValuesSupported)
	assert.Contains(t, discovery.CodeChallengeMethodsSupported, oidc.CodeChallengeMethodS256)
	assert.Contains(t, discovery.TokenEndpointAuthMethodsSupported, oidc.AuthMethodPost)

	// Phase 3 must filter these capabilities: the library advertises them even
	// though the storage does not implement the corresponding flows.
	assert.Contains(t, discovery.GrantTypesSupported, oidc.GrantTypeImplicit)
	assert.Contains(t, discovery.ResponseTypesSupported, string(oidc.ResponseTypeIDTokenOnly))
	assert.Contains(t, discovery.ResponseTypesSupported, string(oidc.ResponseTypeIDToken))
	assert.NotEmpty(t, discovery.DeviceAuthorizationEndpoint)
	assert.NotEmpty(t, discovery.RevocationEndpoint)
	assert.NotEmpty(t, discovery.EndSessionEndpoint)
	assert.NotContains(t, discovery.ScopesSupported, oidc.ScopeOfflineAccess)

	t.Logf(
		"discovery over-advertising recorded for Phase 3: grants=%v response_types=%v device_authorization=%q revocation=%q end_session=%q scopes=%v",
		discovery.GrantTypesSupported,
		discovery.ResponseTypesSupported,
		discovery.DeviceAuthorizationEndpoint,
		discovery.RevocationEndpoint,
		discovery.EndSessionEndpoint,
		discovery.ScopesSupported,
	)
}

func TestJWKSExposesRS256SigningKey(t *testing.T) {
	h := newSpikeHarness(t)

	var jwks jose.JSONWebKeySet
	h.getJSON(h.oidcPath("/keys"), &jwks)

	require.Len(t, jwks.Keys, 1)
	key := jwks.Keys[0]
	assert.Equal(t, h.storage.signingKeyID, key.KeyID)
	assert.Equal(t, "RS256", key.Algorithm)
	assert.Equal(t, "sig", key.Use)
	publicKey, ok := key.Key.(*rsa.PublicKey)
	require.True(t, ok)
	assert.Equal(t, 2048, publicKey.N.BitLen())
}

func TestAuthorizationCodePKCEFlowThroughAuthulaRouter(t *testing.T) {
	h := newSpikeHarness(t)

	const (
		state    = "spike-state"
		nonce    = "spike-nonce"
		verifier = "spike-code-verifier"
	)

	authorizeQuery := url.Values{
		"response_type":         {string(oidc.ResponseTypeCode)},
		"client_id":             {spikeClientID},
		"redirect_uri":          {h.redirectURI},
		"scope":                 {oidc.ScopeOpenID + " " + oidc.ScopeProfile + " " + oidc.ScopeEmail},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {oidc.NewSHACodeChallenge(verifier)},
		"code_challenge_method": {string(oidc.CodeChallengeMethodS256)},
	}
	authorizeResp := h.get(h.oidcPath("/authorize") + "?" + authorizeQuery.Encode())
	defer func() { _ = authorizeResp.Body.Close() }()
	require.Equal(t, http.StatusFound, authorizeResp.StatusCode)

	loginURL, err := authorizeResp.Location()
	require.NoError(t, err)
	assert.Equal(t, h.oidcPath("/login"), loginURL.Path)
	requestID := loginURL.Query().Get("oauth2_request_id")
	require.NotEmpty(t, requestID)
	assert.Empty(t, loginURL.Query().Get("oauth2_return_to"))

	loginResp := h.get(loginURL.RequestURI())
	defer func() { _ = loginResp.Body.Close() }()
	require.Equal(t, http.StatusFound, loginResp.StatusCode)
	assert.Equal(t, requestContextMarker, loginResp.Header.Get("X-Authula-Request-Context"))
	assert.Contains(t, loginResp.Header.Get("X-Authula-Route-Plugin"), models.PluginOAuth2.String())

	callbackURL, err := loginResp.Location()
	require.NoError(t, err)
	assert.Equal(t, h.oidcPath("/authorize/callback"), callbackURL.Path)
	assert.Equal(t, requestID, callbackURL.Query().Get("id"))

	callbackResp := h.get(callbackURL.RequestURI())
	defer func() { _ = callbackResp.Body.Close() }()
	require.Equal(t, http.StatusFound, callbackResp.StatusCode)

	relyingPartyURL, err := callbackResp.Location()
	require.NoError(t, err)
	assert.Equal(t, h.redirectURI, relyingPartyURL.Scheme+"://"+relyingPartyURL.Host+relyingPartyURL.Path)
	assert.Equal(t, state, relyingPartyURL.Query().Get("state"))
	code := relyingPartyURL.Query().Get("code")
	require.NotEmpty(t, code)

	tokenForm := url.Values{
		"grant_type":    {string(oidc.GrantTypeCode)},
		"code":          {code},
		"redirect_uri":  {h.redirectURI},
		"client_id":     {spikeClientID},
		"client_secret": {spikeSecret},
		"code_verifier": {verifier},
	}
	tokenResp := h.postForm(h.oidcPath("/token"), tokenForm)
	defer func() { _ = tokenResp.Body.Close() }()
	tokenBody, err := io.ReadAll(tokenResp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, tokenResp.StatusCode, string(tokenBody))
	assert.Equal(t, "no-store", tokenResp.Header.Get("Cache-Control"))

	var tokens oidc.AccessTokenResponse
	require.NoError(t, json.Unmarshal(tokenBody, &tokens))
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.IDToken)
	assert.Len(t, strings.Split(tokens.AccessToken, "."), 3, "access token should be a JWT")

	claims := h.verifyIDToken(tokens.IDToken)
	assert.Equal(t, h.issuer, claims.Issuer)
	assert.Equal(t, spikeClientID, claims.ClientID)
	assert.Contains(t, []string(claims.Audience), spikeClientID)
	assert.Equal(t, spikeUserID, claims.Subject)
	assert.Equal(t, nonce, claims.Nonce)

	userinfoResp := h.do(http.MethodGet, h.oidcPath("/userinfo"), nil, "", http.Header{
		"Authorization": {"Bearer " + tokens.AccessToken},
	})
	defer func() { _ = userinfoResp.Body.Close() }()
	userinfoBody, err := io.ReadAll(userinfoResp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, userinfoResp.StatusCode, string(userinfoBody))

	var userinfo oidc.UserInfo
	require.NoError(t, json.Unmarshal(userinfoBody, &userinfo))
	assert.Equal(t, claims.Subject, userinfo.Subject)
	assert.Equal(t, spikeUserEmail, userinfo.Email)

	replayResp := h.postForm(h.oidcPath("/token"), tokenForm)
	defer func() { _ = replayResp.Body.Close() }()
	replayBody, err := io.ReadAll(replayResp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, replayResp.StatusCode, string(replayBody))

	var replayError map[string]any
	require.NoError(t, json.Unmarshal(replayBody, &replayError))
	assert.Equal(t, "invalid_grant", replayError["error"])

	assert.Greater(t, h.hookCount(h.beforeHooks, spikeBasePath+spikeOIDCPath+"/login"), 0)
	assert.Greater(t, h.hookCount(h.afterHooks, spikeBasePath+spikeOIDCPath+"/token"), 0)
}

func TestRouteMetadataGatesOAuth2Hooks(t *testing.T) {
	h := newSpikeHarness(t)

	const unmappedPath = "/plain"
	h.router.RegisterRoute(models.Route{
		Method: http.MethodGet,
		Path:   unmappedPath,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	})

	resp := h.get(spikeBasePath + unmappedPath)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Zero(t, h.hookCount(h.beforeHooks, spikeBasePath+unmappedPath))

	resp = h.get(h.oidcPath("/keys"))
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, h.hookCount(h.beforeHooks, spikeBasePath+spikeOIDCPath+"/keys"))
	assert.Equal(t, 1, h.hookCount(h.afterHooks, spikeBasePath+spikeOIDCPath+"/keys"))
}

func TestTokenRouteMetadataDoesNotIncludeSessionOrCSRF(t *testing.T) {
	h := newSpikeHarness(t)

	tokenMeta, ok := h.routeMeta["POST:"+spikeOIDCPath+"/token"]
	require.True(t, ok)
	plugins := util.ReadStringSliceFromMetadata(tokenMeta, "plugins")
	assert.Equal(t, []string{models.PluginOAuth2.String()}, plugins)
	assert.NotContains(t, plugins, "csrf.protect")
	assert.NotContains(t, plugins, "session.auth")
	assert.Equal(t, h.issuer+"/token", h.provider.TokenEndpoint().Absolute(h.issuer))
	assert.Equal(t, h.issuer+"/introspect", h.provider.IntrospectionEndpoint().Absolute(h.issuer))
}

func (h *spikeHarness) verifyIDToken(rawIDToken string) *oidc.IDTokenClaims {
	h.t.Helper()

	var jwks jose.JSONWebKeySet
	h.getJSON(h.oidcPath("/keys"), &jwks)

	signed, err := jose.ParseSignedCompact(rawIDToken, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(h.t, err)
	require.Len(h.t, signed.Signatures, 1)

	keyID := signed.Signatures[0].Header.KeyID
	keys := jwks.Key(keyID)
	require.Len(h.t, keys, 1)
	payload, err := signed.Verify(keys[0])
	require.NoError(h.t, err)

	claims := new(oidc.IDTokenClaims)
	require.NoError(h.t, json.Unmarshal(payload, claims))
	return claims
}
