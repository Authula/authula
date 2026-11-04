package usecases

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			assert.NotEmpty(t, result.StateCookie)
			assert.NotEmpty(t, result.RedirectCookie)

			if tt.wantChallenge {
				assert.NotEmpty(t, authURL.Query().Get("code_challenge"))
				assert.Equal(t, "S256", authURL.Query().Get("code_challenge_method"))
				require.NotNil(t, result.VerifierCookie)
				return
			}

			assert.Empty(t, authURL.Query().Get("code_challenge"))
			assert.Nil(t, result.VerifierCookie)
		})
	}
}
