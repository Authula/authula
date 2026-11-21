package oauth2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/handlers"
	"github.com/Authula/authula/plugins/oauth2/repositories"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type recordingTokenIssuer struct {
	token   *rootservices.ClientCredentialsToken
	err     error
	request rootservices.ClientCredentialsTokenRequest
	calls   int
}

func (i *recordingTokenIssuer) IssueClientCredentialsToken(_ context.Context, req rootservices.ClientCredentialsTokenRequest) (*rootservices.ClientCredentialsToken, error) {
	i.calls++
	i.request = req
	return i.token, i.err
}

type tokenIntegrationFixture struct {
	handler *handlers.TokenHandler
	repo    repositories.OAuthClientRepository
	secrets services.ClientSecretService
	issuer  *recordingTokenIssuer
}

func newTokenIntegrationFixture(t *testing.T, prefix string) *tokenIntegrationFixture {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)
	_, err := db.NewCreateTable().Model((*types.OAuthClient)(nil)).IfNotExists().Exec(context.Background())
	require.NoError(t, err)

	secrets := services.NewSHA256ClientSecretService(prefix)
	repo := repositories.NewBunOAuthClientRepository(db)
	clients := services.NewClientService(repo, secrets)
	issuer := &recordingTokenIssuer{token: &rootservices.ClientCredentialsToken{
		AccessToken: "signed-access-token",
		ExpiresIn:   15 * time.Minute,
		TokenType:   "Bearer",
	}}
	useCase := usecases.NewClientCredentialsUseCase(
		clients,
		issuer,
		[]string{"read", "write", "admin"},
		15*time.Minute,
		&internaltests.MockLogger{},
	)

	return &tokenIntegrationFixture{
		handler: &handlers.TokenHandler{UseCase: useCase, Logger: &internaltests.MockLogger{}},
		repo:    repo,
		secrets: secrets,
		issuer:  issuer,
	}
}

func (f *tokenIntegrationFixture) seedClient(t *testing.T, prefix string, configure func(*types.OAuthClient)) (string, string) {
	t.Helper()

	generated, err := f.secrets.Generate()
	require.NoError(t, err)

	client := &types.OAuthClient{
		ID:                 util.GenerateUUID(),
		ClientID:           "client-" + util.GenerateUUID(),
		ClientSecretHash:   generated.Hash,
		ClientSecretSalt:   generated.Salt,
		ClientSecretPrefix: generated.Prefix,
		ClientSecretStart:  generated.Start,
		ClientSecretLast:   generated.Last,
		Name:               "integration client",
		RedirectURIs:       []string{},
		AllowedGrants:      []string{types.OAuthGrantTypeClientCredentials},
		AllowedScopes:      []string{"read", "write", "admin"},
		OwnerType:          types.OAuthClientOwnerTypeOrganization,
		OwnerID:            "org-1",
		CreatorID:          "user-1",
		IsActive:           true,
	}
	if configure != nil {
		configure(client)
	}

	_, err = f.repo.Create(context.Background(), client)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(generated.Value, prefix))
	require.Equal(t, prefix, client.ClientSecretPrefix)

	return client.ClientID, generated.Value
}

func (f *tokenIntegrationFixture) request(t *testing.T, body string, headers map[string][]string) (*models.RequestContext, map[string]any) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/oauth2/token", strings.NewReader(body))
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	recorder := httptest.NewRecorder()
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: recorder,
		Values:         make(map[string]any),
	}
	req = req.WithContext(models.NewContextWithRequestContext(req.Context(), reqCtx))
	reqCtx.Request = req

	f.handler.Handle()(recorder, req)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &payload))
	return reqCtx, payload
}

func basicHeader(clientID, secret string) string {
	value := base64.StdEncoding.EncodeToString([]byte(url.QueryEscape(clientID) + ":" + url.QueryEscape(secret)))
	return "Basic " + value
}

var formHeaders = map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}}

func assertNoStoreHeaders(t *testing.T, reqCtx *models.RequestContext) {
	t.Helper()
	assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
	assert.Equal(t, "no-cache", reqCtx.ResponseHeaders.Get("Pragma"))
}

func assertClientAuthFailure(t *testing.T, reqCtx *models.RequestContext, payload map[string]any) {
	t.Helper()
	assert.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
	assert.Equal(t, "invalid_client", payload["error"])
	assert.Equal(t, `Basic realm="oauth2/token", charset="UTF-8"`, reqCtx.ResponseHeaders.Get("WWW-Authenticate"))
	assertNoStoreHeaders(t, reqCtx)
}

func assertPostClientAuthFailure(t *testing.T, reqCtx *models.RequestContext, payload map[string]any) {
	t.Helper()
	assert.Equal(t, http.StatusBadRequest, reqCtx.ResponseStatus)
	assert.Equal(t, "invalid_client", payload["error"])
	assert.Empty(t, reqCtx.ResponseHeaders.Get("WWW-Authenticate"))
	assertNoStoreHeaders(t, reqCtx)
}

func TestClientCredentialsTokenEndpoint_Integration(t *testing.T) {
	t.Parallel()

	t.Run("client secret basic", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", nil)

		reqCtx, payload := f.request(t, "grant_type=client_credentials&scope=read+write", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
		assert.Equal(t, "signed-access-token", payload["access_token"])
		assert.Equal(t, "Bearer", payload["token_type"])
		assert.EqualValues(t, 900, payload["expires_in"])
		assert.Equal(t, "read write", payload["scope"])
		assert.NotContains(t, payload, "refresh_token")
		assert.NotContains(t, payload, "id_token")
		assertNoStoreHeaders(t, reqCtx)

		require.Equal(t, 1, f.issuer.calls)
		assert.Equal(t, []string{"read", "write"}, f.issuer.request.Scopes)
		assert.Equal(t, "org-1", f.issuer.request.OrganizationID)
	})

	t.Run("client secret basic with reserved characters", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID := "client id%&+:"
		clientSecret := "secret id%&+:"

		generated, err := f.secrets.Generate()
		require.NoError(t, err)
		hash, salt, err := f.secrets.Hash(clientSecret)
		require.NoError(t, err)

		_, err = f.repo.Create(context.Background(), &types.OAuthClient{
			ID:                 util.GenerateUUID(),
			ClientID:           clientID,
			ClientSecretHash:   hash,
			ClientSecretSalt:   salt,
			ClientSecretPrefix: generated.Prefix,
			ClientSecretStart:  generated.Start,
			ClientSecretLast:   generated.Last,
			Name:               "reserved character client",
			RedirectURIs:       []string{},
			AllowedGrants:      []string{types.OAuthGrantTypeClientCredentials},
			AllowedScopes:      []string{"read"},
			OwnerType:          types.OAuthClientOwnerTypeOrganization,
			OwnerID:            "org-1",
			CreatorID:          "user-1",
			IsActive:           true,
		})
		require.NoError(t, err)

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, clientSecret)},
		})

		require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
		assert.Equal(t, "signed-access-token", payload["access_token"])
	})

	t.Run("client secret post", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", nil)

		reqCtx, payload := f.request(t,
			"grant_type=client_credentials&client_id="+url.QueryEscape(clientID)+"&client_secret="+url.QueryEscape(secret),
			formHeaders,
		)

		require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
		assert.Equal(t, "signed-access-token", payload["access_token"])
		assert.NotContains(t, payload, "scope")
		assert.Equal(t, []string{}, f.issuer.request.Scopes)
	})

	t.Run("wrong secret", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, _ := f.seedClient(t, "authula_cs_", nil)

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, "wrong-secret")},
		})

		assertClientAuthFailure(t, reqCtx, payload)
		assert.Equal(t, 0, f.issuer.calls)
	})

	t.Run("unknown client", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader("missing-client", "secret")},
		})

		assertClientAuthFailure(t, reqCtx, payload)
	})

	t.Run("inactive client", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", func(client *types.OAuthClient) {
			client.IsActive = false
		})

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		assertClientAuthFailure(t, reqCtx, payload)
	})

	t.Run("soft deleted client", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", nil)

		stored, err := f.repo.GetByClientID(context.Background(), clientID)
		require.NoError(t, err)
		require.NotNil(t, stored)
		deleted, err := f.repo.SoftDelete(context.Background(), stored.ID, time.Now().UTC())
		require.NoError(t, err)
		require.True(t, deleted)

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		assertClientAuthFailure(t, reqCtx, payload)
	})

	t.Run("missing post credentials", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		f.seedClient(t, "authula_cs_", nil)

		reqCtx, payload := f.request(t, "grant_type=client_credentials", formHeaders)

		assertPostClientAuthFailure(t, reqCtx, payload)
	})

	t.Run("wrong post secret", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, _ := f.seedClient(t, "authula_cs_", nil)

		reqCtx, payload := f.request(t,
			"grant_type=client_credentials&client_id="+url.QueryEscape(clientID)+"&client_secret=wrong-secret",
			formHeaders,
		)

		assertPostClientAuthFailure(t, reqCtx, payload)
	})

	t.Run("malformed basic credentials", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {"Basic not-base64!"},
		})

		assertClientAuthFailure(t, reqCtx, payload)
	})

	t.Run("multiple authorization headers", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", nil)

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret), basicHeader(clientID, secret)},
		})

		assertClientAuthFailure(t, reqCtx, payload)
	})

	t.Run("grant not allowed", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", func(client *types.OAuthClient) {
			client.AllowedGrants = []string{types.OAuthGrantTypeAuthorizationCode}
		})

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		assert.Equal(t, http.StatusBadRequest, reqCtx.ResponseStatus)
		assert.Equal(t, "unauthorized_client", payload["error"])
		assertNoStoreHeaders(t, reqCtx)
	})

	t.Run("scope escalation", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", func(client *types.OAuthClient) {
			client.AllowedScopes = []string{"read"}
		})

		reqCtx, payload := f.request(t, "grant_type=client_credentials&scope=admin", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		assert.Equal(t, http.StatusBadRequest, reqCtx.ResponseStatus)
		assert.Equal(t, "invalid_scope", payload["error"])
	})

	t.Run("wildcard scope", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", nil)

		reqCtx, payload := f.request(t, "grant_type=client_credentials&scope=%2A", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		assert.Equal(t, http.StatusBadRequest, reqCtx.ResponseStatus)
		assert.Equal(t, "invalid_scope", payload["error"])
	})

	for _, oidcScope := range []string{"openid", "profile", "email", "address", "phone", "offline_access"} {
		t.Run("OIDC scope "+oidcScope, func(t *testing.T) {
			t.Parallel()

			f := newTokenIntegrationFixture(t, "authula_cs_")
			clientID, secret := f.seedClient(t, "authula_cs_", func(client *types.OAuthClient) {
				client.AllowedScopes = append(client.AllowedScopes, oidcScope)
			})

			reqCtx, payload := f.request(t,
				"grant_type=client_credentials&scope="+url.QueryEscape(oidcScope),
				map[string][]string{
					"Content-Type":  {"application/x-www-form-urlencoded"},
					"Authorization": {basicHeader(clientID, secret)},
				},
			)

			assert.Equal(t, http.StatusBadRequest, reqCtx.ResponseStatus)
			assert.Equal(t, "invalid_scope", payload["error"])
		})
	}

	t.Run("present but empty scope", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", nil)

		reqCtx, payload := f.request(t, "grant_type=client_credentials&scope=", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		assert.Equal(t, http.StatusBadRequest, reqCtx.ResponseStatus)
		assert.Equal(t, "invalid_scope", payload["error"])
	})

	t.Run("issuer failure returns project json", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "authula_cs_")
		clientID, secret := f.seedClient(t, "authula_cs_", nil)
		f.issuer.err = assert.AnError

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		assert.Equal(t, http.StatusInternalServerError, reqCtx.ResponseStatus)
		assert.Equal(t, "server_error", payload["error"])
		assert.Equal(t, "The authorization server encountered an unexpected condition.", payload["error_description"])
		assert.NotContains(t, payload, "message")
		assertNoStoreHeaders(t, reqCtx)
	})

	t.Run("prefix override", func(t *testing.T) {
		t.Parallel()

		f := newTokenIntegrationFixture(t, "custom_prefix_")
		clientID, secret := f.seedClient(t, "custom_prefix_", nil)

		assert.True(t, strings.HasPrefix(secret, "custom_prefix_"))

		reqCtx, payload := f.request(t, "grant_type=client_credentials", map[string][]string{
			"Content-Type":  {"application/x-www-form-urlencoded"},
			"Authorization": {basicHeader(clientID, secret)},
		})

		require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
		assert.Equal(t, "signed-access-token", payload["access_token"])
	})
}

func TestClientCredentialsTokenEndpoint_ConfigValidation(t *testing.T) {
	t.Parallel()

	invalidPrefix := types.OAuth2PluginConfig{
		AuthorizationServerEnabled: true,
		AllowedScopes:              []string{"read"},
		ClientSecretValuePrefix:    "bad prefix",
		ClientCredentialsTokenTTL:  time.Minute,
	}
	require.Error(t, invalidPrefix.Validate())

	oidcOnly := types.OAuth2PluginConfig{
		AuthorizationServerEnabled: true,
		AllowedScopes:              []string{"openid", "profile", "email"},
		ClientSecretValuePrefix:    constants.CookieFlowBinding,
		ClientCredentialsTokenTTL:  time.Minute,
	}
	require.Error(t, oidcOnly.Validate())

	valid := types.OAuth2PluginConfig{
		AuthorizationServerEnabled: true,
		AllowedScopes:              []string{"openid", "read"},
		ClientSecretValuePrefix:    "authula_cs_",
		ClientCredentialsTokenTTL:  time.Minute,
	}
	require.NoError(t, valid.Validate())
}
