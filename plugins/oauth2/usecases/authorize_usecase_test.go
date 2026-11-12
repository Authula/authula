package usecases

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	corerepos "github.com/Authula/authula/core/repositories"
	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/oauth2/constants"
	oauth2tests "github.com/Authula/authula/plugins/oauth2/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type authorizeFixture struct {
	useCase      *authorizeUseCase
	provider     *oauth2tests.MockProvider
	repo         *oauth2tests.FlowStateRepositoryFake
	tokenService rootservices.TokenService
}

func newAuthorizeFixture(t *testing.T, provider *oauth2tests.MockProvider) *authorizeFixture {
	t.Helper()

	repo := oauth2tests.NewFlowStateRepositoryFake()
	tokenRepo, err := corerepos.NewCryptoTokenRepository(internaltests.MustDecodeKey(internaltests.EncryptionKeyOne), nil)
	require.NoError(t, err)
	tokenService := coreservices.NewTokenService(tokenRepo)
	useCase := NewAuthorizeUseCase(
		oauth2tests.NewRegistry(provider),
		&internaltests.MockLogger{},
		[]string{"https://app.example.com"},
		repo,
		tokenService,
		5*time.Minute,
	)

	return &authorizeFixture{
		useCase:      useCase,
		provider:     provider,
		repo:         repo,
		tokenService: tokenService,
	}
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

func TestOAuthStateHash(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"4b491dfe3fa77c74d7099b3c0dc0c458cfb0de246adcd66f3a13d585d79fbaef",
		util.HashWithSalt("state-1", "binding-1"),
	)
}

func TestAuthorizeUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		pkce            bool
		providerID      string
		redirectTo      string
		actorID         string
		authURLOverride string
		createErr       error
		wantErr         error
		wantChallenge   bool
		wantPurpose     string
		wantRedirect    *string
		wantNoRow       bool
	}{
		{
			name:          "pkce provider includes challenge",
			pkce:          true,
			providerID:    "google",
			redirectTo:    "/welcome",
			wantChallenge: true,
			wantPurpose:   types.OAuthFlowPurposeLogin,
			wantRedirect:  new("/welcome"),
		},
		{
			name:         "non pkce provider omits challenge",
			pkce:         false,
			providerID:   "google",
			redirectTo:   "/welcome",
			wantPurpose:  types.OAuthFlowPurposeLogin,
			wantRedirect: new("/welcome"),
		},
		{
			name:          "link purpose is stored for a signed-in user",
			pkce:          true,
			providerID:    "google",
			redirectTo:    "/account",
			actorID:       "user-1",
			wantChallenge: true,
			wantPurpose:   types.OAuthFlowPurposeLink,
			wantRedirect:  new("/account"),
		},
		{
			name:          "absent redirect is stored as nil",
			pkce:          true,
			providerID:    "google",
			wantChallenge: true,
			wantPurpose:   types.OAuthFlowPurposeLogin,
		},
		{
			name:       "rejects untrusted redirect",
			pkce:       true,
			providerID: "google",
			redirectTo: "https://evil.example/callback",
			wantErr:    constants.ErrInvalidRedirectTo,
			wantNoRow:  true,
		},
		{
			name:       "unknown provider",
			pkce:       true,
			providerID: "missing",
			redirectTo: "/welcome",
			wantErr:    constants.ErrProviderNotFound,
			wantNoRow:  true,
		},
		{
			name:       "repository failure is surfaced",
			pkce:       true,
			providerID: "google",
			redirectTo: "/welcome",
			createErr:  errors.New("insert failed"),
			wantNoRow:  true,
		},
		{
			name:            "invalid authorization URL is not persisted",
			pkce:            false,
			providerID:      "google",
			redirectTo:      "/welcome",
			authURLOverride: "https://provider.example/authorize?response_type=token",
			wantErr:         constants.ErrProviderFailure,
			wantNoRow:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &oauth2tests.MockProvider{
				ProviderName:    "google",
				PKCE:            tt.pkce,
				AuthURLOverride: tt.authURLOverride,
			}
			f := newAuthorizeFixture(t, provider)
			f.repo.CreateErr = tt.createErr

			result, err := f.useCase.Authorize(context.Background(), &types.AuthorizeRequest{
				ProviderID: tt.providerID,
				RedirectTo: tt.redirectTo,
				ActorID:    tt.actorID,
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				assert.Equal(t, 0, f.repo.Count())
				return
			}
			if tt.createErr != nil {
				require.ErrorIs(t, err, tt.createErr)
				assert.Nil(t, result)
				assert.Equal(t, 0, f.repo.Count())
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.Binding)
			assert.Equal(t, 1, f.repo.Count())

			authURL, err := url.Parse(result.AuthorizationURL)
			require.NoError(t, err)
			state := authURL.Query().Get("state")
			require.NotEmpty(t, state)
			assert.Equal(t, "code", authURL.Query().Get("response_type"), "implicit flow must never be used")

			stored, err := f.repo.GetByStateHash(context.Background(), util.HashWithSalt(state, result.Binding))
			require.NoError(t, err)
			require.NotNil(t, stored)
			assert.Equal(t, tt.wantPurpose, stored.Purpose)
			assert.Equal(t, "google", stored.Provider)
			assert.WithinDuration(t, time.Now().Add(5*time.Minute), stored.ExpiresAt, time.Minute)
			if tt.actorID == "" {
				assert.Nil(t, stored.UserID)
			} else {
				require.NotNil(t, stored.UserID)
				assert.Equal(t, tt.actorID, *stored.UserID)
			}
			if tt.wantRedirect == nil {
				assert.Nil(t, stored.RedirectTo)
			} else {
				require.NotNil(t, stored.RedirectTo)
				assert.Equal(t, *tt.wantRedirect, *stored.RedirectTo)
			}

			if tt.wantChallenge {
				challenge := authURL.Query().Get("code_challenge")
				assert.NotEmpty(t, challenge)
				assert.Equal(t, "S256", authURL.Query().Get("code_challenge_method"))
				require.NotNil(t, stored.CodeVerifierEncrypted)
				verifier, err := f.tokenService.Decrypt(*stored.CodeVerifierEncrypted)
				require.NoError(t, err)
				assert.Len(t, verifier, 43)
				assert.Equal(t, oauth2.S256ChallengeFromVerifier(verifier), challenge)
				return
			}

			assert.Empty(t, authURL.Query().Get("code_challenge"))
			assert.Nil(t, stored.CodeVerifierEncrypted)
		})
	}
}
