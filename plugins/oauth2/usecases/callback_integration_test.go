package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	corerepos "github.com/Authula/authula/core/repositories"
	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/services"
	oauth2tests "github.com/Authula/authula/plugins/oauth2/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
)

func TestCallbackUseCase_SQLiteStoresEncryptedTokensForCorrectOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := internaltests.NewSQLiteIntegrationDB(t)

	for _, model := range []any{(*models.User)(nil), (*models.Account)(nil), (*models.Session)(nil)} {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		require.NoError(t, err)
	}

	tokenRepo := corerepos.NewCryptoTokenRepository("integration-secret")
	accountRepo := corerepos.NewBunAccountRepository(db)
	userRepo := corerepos.NewBunUserRepository(db)
	sessionRepo := corerepos.NewBunSessionRepository(db)

	userService := coreservices.NewUserService(userRepo, nil, &internaltests.MockLogger{})
	accountService := coreservices.NewAccountService(nil, accountRepo, tokenRepo, nil, &internaltests.MockLogger{})
	sessionService := coreservices.NewSessionService(sessionRepo, nil, nil, &internaltests.MockLogger{})
	tokenService := coreservices.NewTokenService(tokenRepo)

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		Token: &oauth2.Token{
			AccessToken:  "plain-access-token",
			RefreshToken: "plain-refresh-token",
			Expiry:       time.Now().Add(time.Hour),
		},
		UserInfo: &types.UserInfo{
			ProviderAccountID: "google-1",
			Email:             "new@example.com",
			EmailVerified:     true,
			Name:              "New User",
		},
	}
	hmacKey := services.DeriveOAuthHMACKey("integration-secret")
	useCase := usecases.NewCallbackUseCase(
		&models.Config{Session: models.SessionConfig{ExpiresIn: time.Hour}},
		oauth2tests.NewRegistry(provider),
		&internaltests.MockLogger{},
		hmacKey,
		userService,
		accountService,
		sessionService,
		tokenService,
	)

	stateCookie, err := services.SignState(services.StatePayload{
		State:    "state-1",
		Provider: "google",
		Purpose:  services.StatePurposeLogin,
	}, hmacKey)
	require.NoError(t, err)

	result, err := useCase.Callback(ctx, &types.CallbackRequest{
		ProviderID:  "google",
		Code:        "code-1",
		State:       "state-1",
		StateCookie: stateCookie,
	}, nil, nil)

	require.NoError(t, err)
	require.NotNil(t, result.User)

	stored, err := accountRepo.GetByProviderAndAccountID(ctx, "google", "google-1")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, result.User.ID, stored.UserID)
	require.NotNil(t, stored.AccessToken)
	require.NotNil(t, stored.RefreshToken)
	assert.NotEqual(t, "plain-access-token", *stored.AccessToken)
	assert.NotEqual(t, "plain-refresh-token", *stored.RefreshToken)

	decryptedAccess, err := tokenRepo.Decrypt(*stored.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "plain-access-token", decryptedAccess)

	decryptedRefresh, err := tokenRepo.Decrypt(*stored.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, "plain-refresh-token", decryptedRefresh)
}

func TestCallbackUseCase_SQLiteLinksSecondProviderWhileSignedIn(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := internaltests.NewSQLiteIntegrationDB(t)

	for _, model := range []any{(*models.User)(nil), (*models.Account)(nil), (*models.Session)(nil)} {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		require.NoError(t, err)
	}

	tokenRepo := corerepos.NewCryptoTokenRepository("integration-secret")
	accountRepo := corerepos.NewBunAccountRepository(db)
	userRepo := corerepos.NewBunUserRepository(db)
	sessionRepo := corerepos.NewBunSessionRepository(db)

	userService := coreservices.NewUserService(userRepo, nil, &internaltests.MockLogger{})
	accountService := coreservices.NewAccountService(nil, accountRepo, tokenRepo, nil, &internaltests.MockLogger{})
	sessionService := coreservices.NewSessionService(sessionRepo, nil, nil, &internaltests.MockLogger{})
	tokenService := coreservices.NewTokenService(tokenRepo)

	user, err := userService.Create(ctx, "Existing User", "existing@example.com", true, nil, nil)
	require.NoError(t, err)
	_, err = accountService.Create(ctx, user.ID, "existing@example.com", "email-password", new("hashed-password"))
	require.NoError(t, err)

	session, err := sessionService.Create(ctx, user.ID, tokenService.Hash("session-token"), nil, nil, time.Hour)
	require.NoError(t, err)

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		Token: &oauth2.Token{
			AccessToken:  "linked-access-token",
			RefreshToken: "linked-refresh-token",
			Expiry:       time.Now().Add(time.Hour),
		},
		UserInfo: &types.UserInfo{
			ProviderAccountID: "google-linked",
			Email:             "existing@example.com",
			EmailVerified:     true,
			Name:              "Existing User",
		},
	}
	hmacKey := services.DeriveOAuthHMACKey("integration-secret")
	useCase := usecases.NewCallbackUseCase(
		&models.Config{Session: models.SessionConfig{ExpiresIn: time.Hour}},
		oauth2tests.NewRegistry(provider),
		&internaltests.MockLogger{},
		hmacKey,
		userService,
		accountService,
		sessionService,
		tokenService,
	)

	stateCookie, err := services.SignState(services.StatePayload{
		State:    "state-link",
		Provider: "google",
		UserID:   user.ID,
		Purpose:  services.StatePurposeLink,
	}, hmacKey)
	require.NoError(t, err)

	result, err := useCase.Callback(ctx, &types.CallbackRequest{
		ProviderID:  "google",
		Code:        "code-link",
		State:       "state-link",
		StateCookie: stateCookie,
		ActorID:     user.ID,
		SessionID:   session.ID,
	}, nil, nil)

	require.NoError(t, err)
	assert.True(t, result.ReusedSession)
	assert.Equal(t, session.ID, result.Session.ID)

	accounts, err := accountRepo.GetAllByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)

	linked, err := accountRepo.GetByProviderAndAccountID(ctx, "google", "google-linked")
	require.NoError(t, err)
	require.NotNil(t, linked)
	assert.Equal(t, user.ID, linked.UserID)
}
