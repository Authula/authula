package usecases_test

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
	"github.com/Authula/authula/plugins/oauth2/usecases"
)

func newAuthorizeUseCase(t *testing.T, provider *oauth2tests.MockProvider) *usecases.AuthorizeUseCase {
	t.Helper()

	return usecases.NewAuthorizeUseCase(
		oauth2tests.NewRegistry(provider),
		&internaltests.MockLogger{},
		[]string{"https://app.example.com"},
		services.DeriveOAuthHMACKey("test-secret"),
	)
}

func TestAuthorizeUseCase_PKCE(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		pkce          bool
		wantChallenge bool
	}{
		{name: "pkce provider includes challenge", pkce: true, wantChallenge: true},
		{name: "non pkce provider omits challenge", pkce: false, wantChallenge: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &oauth2tests.MockProvider{ProviderName: "google", PKCE: tt.pkce}
			useCase := newAuthorizeUseCase(t, provider)

			result, err := useCase.Authorize(context.Background(), &types.AuthorizeRequest{
				ProviderID: "google",
				RedirectTo: "/welcome",
			})
			require.NoError(t, err)

			authURL, err := url.Parse(result.AuthorizationURL)
			require.NoError(t, err)

			if tt.wantChallenge {
				assert.NotEmpty(t, authURL.Query().Get("code_challenge"))
				assert.Equal(t, "S256", authURL.Query().Get("code_challenge_method"))
				require.NotNil(t, result.VerifierCookie)
			} else {
				assert.Empty(t, authURL.Query().Get("code_challenge"))
				assert.Nil(t, result.VerifierCookie)
			}
		})
	}
}

func TestAuthorizeUseCase_RejectsUntrustedRedirect(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{ProviderName: "google", PKCE: true}
	useCase := newAuthorizeUseCase(t, provider)

	_, err := useCase.Authorize(context.Background(), &types.AuthorizeRequest{
		ProviderID: "google",
		RedirectTo: "https://evil.example/callback",
	})

	require.ErrorIs(t, err, constants.ErrInvalidRedirectTo)
}

func TestAuthorizeUseCase_UnknownProvider(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{ProviderName: "google", PKCE: true}
	useCase := newAuthorizeUseCase(t, provider)

	_, err := useCase.Authorize(context.Background(), &types.AuthorizeRequest{ProviderID: "missing"})

	require.ErrorIs(t, err, constants.ErrProviderNotFound)
}
