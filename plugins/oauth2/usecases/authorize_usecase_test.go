package usecases

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/services"
	oauth2tests "github.com/Authula/authula/plugins/oauth2/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
)

func newAuthorizeUseCase(t *testing.T, provider *oauth2tests.MockProvider) *authorizeUseCase {
	t.Helper()

	return NewAuthorizeUseCase(
		oauth2tests.NewRegistry(provider),
		&internaltests.MockLogger{},
		[]string{"https://app.example.com"},
		services.DeriveOAuthHMACKey("test-secret"),
	)
}

func TestValidateAuthorizationURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		url          string
		pkceRequired bool
		wantErr      error
	}{
		{
			name:         "code flow with S256",
			url:          "https://provider.example/authorize?response_type=code&code_challenge=abc&code_challenge_method=S256",
			pkceRequired: true,
		},
		{
			name:         "code flow without PKCE when opted out",
			url:          "https://provider.example/authorize?response_type=code",
			pkceRequired: false,
		},
		{
			name:         "implicit response is rejected",
			url:          "https://provider.example/authorize?response_type=token",
			pkceRequired: false,
			wantErr:      constants.ErrProviderFailure,
		},
		{
			name:         "missing challenge is rejected",
			url:          "https://provider.example/authorize?response_type=code",
			pkceRequired: true,
			wantErr:      constants.ErrPKCEChallengeMissing,
		},
		{
			name:         "plain challenge is rejected",
			url:          "https://provider.example/authorize?response_type=code&code_challenge=abc&code_challenge_method=plain",
			pkceRequired: true,
			wantErr:      constants.ErrPKCEChallengeMissing,
		},
		{
			name:         "relative URL is rejected",
			url:          "/authorize?response_type=code",
			pkceRequired: false,
			wantErr:      constants.ErrProviderFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateAuthorizationURL(tt.url, tt.pkceRequired)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAuthorizeUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		pkce          bool
		providerID    string
		redirectTo    string
		wantErr       error
		wantChallenge bool
	}{
		{
			name:          "pkce provider includes challenge",
			pkce:          true,
			providerID:    "google",
			redirectTo:    "/welcome",
			wantChallenge: true,
		},
		{
			name:       "non pkce provider omits challenge",
			pkce:       false,
			providerID: "google",
			redirectTo: "/welcome",
		},
		{
			name:       "rejects untrusted redirect",
			pkce:       true,
			providerID: "google",
			redirectTo: "https://evil.example/callback",
			wantErr:    constants.ErrInvalidRedirectTo,
		},
		{
			name:       "unknown provider",
			pkce:       true,
			providerID: "missing",
			redirectTo: "/welcome",
			wantErr:    constants.ErrProviderNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &oauth2tests.MockProvider{ProviderName: "google", PKCE: tt.pkce}
			useCase := newAuthorizeUseCase(t, provider)

			result, err := useCase.Authorize(context.Background(), &types.AuthorizeRequest{
				ProviderID: tt.providerID,
				RedirectTo: tt.redirectTo,
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			authURL, err := url.Parse(result.AuthorizationURL)
			require.NoError(t, err)
			assert.NotEmpty(t, authURL.Query().Get("state"))
			assert.Equal(t, "code", authURL.Query().Get("response_type"), "implicit flow must never be used")
			assert.NotEmpty(t, result.StateCookie)
			assert.NotEmpty(t, result.RedirectCookie)

			if tt.wantChallenge {
				assert.NotEmpty(t, authURL.Query().Get("code_challenge"))
				assert.Equal(t, "S256", authURL.Query().Get("code_challenge_method"))
				require.NotNil(t, result.VerifierCookie)

				verifier, err := services.ValidateCookie(*result.VerifierCookie, services.DeriveOAuthHMACKey("test-secret"), constants.FlowCookieTTL)
				require.NoError(t, err)
				require.NotEmpty(t, verifier)
				assert.Equal(t, oauth2.S256ChallengeFromVerifier(verifier), authURL.Query().Get("code_challenge"))
				return
			}

			assert.Empty(t, authURL.Query().Get("code_challenge"))
			assert.Nil(t, result.VerifierCookie)
		})
	}
}
