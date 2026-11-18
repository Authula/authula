package magiclink

import (
	"context"
	"errors"
	"testing"
	"time"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	types "github.com/Authula/authula/plugins/magic-link/types"
	"github.com/Authula/authula/plugins/magic-link/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSignInUseCase struct {
	mock.Mock
}

func (m *mockSignInUseCase) SignIn(ctx context.Context, name *string, email string, callbackURL *string) (*types.MagicLinkSignInResult, error) {
	args := m.Called(ctx, name, email, callbackURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.MagicLinkSignInResult), args.Error(1)
}

type mockVerifyUseCase struct {
	mock.Mock
}

func (m *mockVerifyUseCase) Verify(ctx context.Context, token string, ipAddress *string, userAgent *string) (string, error) {
	args := m.Called(ctx, token, ipAddress, userAgent)
	return args.String(0), args.Error(1)
}

type mockExchangeUseCase struct {
	mock.Mock
}

func (m *mockExchangeUseCase) Exchange(ctx context.Context, token string, ipAddress *string, userAgent *string) (*types.MagicLinkExchangeResult, error) {
	args := m.Called(ctx, token, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.MagicLinkExchangeResult), args.Error(1)
}

func newWiredPlugin() *MagicLinkPlugin {
	return &MagicLinkPlugin{
		globalConfig:        &models.Config{BaseURL: "http://localhost"},
		pluginConfig:        &types.MagicLinkPluginConfig{ExpiresIn: 15 * time.Minute},
		logger:              &internaltests.MockLogger{},
		userService:         &internaltests.MockUserService{},
		accountService:      &internaltests.MockAccountService{},
		sessionService:      &internaltests.MockSessionService{},
		verificationService: &internaltests.MockVerificationService{},
		tokenService:        &internaltests.MockTokenService{},
		mailerService:       &internaltests.MockMailerService{},
	}
}

func TestBuildUseCases_WiresPluginFields(t *testing.T) {
	p := newWiredPlugin()

	useCases := BuildUseCases(p)

	require.NotNil(t, useCases)

	signIn, ok := useCases.SignInUseCase.(*usecases.SignInUseCaseImpl)
	require.True(t, ok)
	assert.Same(t, p.globalConfig, signIn.GlobalConfig)
	assert.Same(t, p.pluginConfig, signIn.PluginConfig)
	assert.Same(t, p.logger, signIn.Logger)
	assert.Same(t, p.userService, signIn.UserService)
	assert.Same(t, p.accountService, signIn.AccountService)
	assert.Same(t, p.verificationService, signIn.VerificationService)
	assert.Same(t, p.tokenService, signIn.TokenService)
	assert.Same(t, p.mailerService, signIn.MailerService)
	assert.Same(t, p.emailTemplateManager, signIn.EmailTemplateManager)

	verify, ok := useCases.VerifyUseCase.(*usecases.VerifyUseCaseImpl)
	require.True(t, ok)
	assert.Same(t, p.globalConfig, verify.GlobalConfig)
	assert.Same(t, p.pluginConfig, verify.PluginConfig)
	assert.Same(t, p.logger, verify.Logger)
	assert.Same(t, p.userService, verify.UserService)
	assert.Same(t, p.verificationService, verify.VerificationService)
	assert.Same(t, p.tokenService, verify.TokenService)

	exchange, ok := useCases.ExchangeUseCase.(*usecases.ExchangeUseCaseImpl)
	require.True(t, ok)
	assert.Same(t, p.globalConfig, exchange.GlobalConfig)
	assert.Same(t, p.pluginConfig, exchange.PluginConfig)
	assert.Same(t, p.logger, exchange.Logger)
	assert.Same(t, p.userService, exchange.UserService)
	assert.Same(t, p.accountService, exchange.AccountService)
	assert.Same(t, p.sessionService, exchange.SessionService)
	assert.Same(t, p.verificationService, exchange.VerificationService)
	assert.Same(t, p.tokenService, exchange.TokenService)
}

func TestBuildAPI_WiresUseCases(t *testing.T) {
	p := newWiredPlugin()

	api := BuildAPI(p)

	require.NotNil(t, api)
	require.NotNil(t, api.useCases)
	assert.IsType(t, &usecases.SignInUseCaseImpl{}, api.useCases.SignInUseCase)
	assert.IsType(t, &usecases.VerifyUseCaseImpl{}, api.useCases.VerifyUseCase)
	assert.IsType(t, &usecases.ExchangeUseCaseImpl{}, api.useCases.ExchangeUseCase)
}

func TestAPI_SignIn(t *testing.T) {
	signInErr := errors.New("sign-in failed")
	wantResult := &types.MagicLinkSignInResult{Token: "sign-in-token"}
	name := new("Test User")
	callbackURL := new("https://example.com/callback")

	tests := []struct {
		name    string
		result  *types.MagicLinkSignInResult
		err     error
		wantErr bool
	}{
		{
			name:   "delegates and returns the use case result",
			result: wantResult,
		},
		{
			name:    "propagates the use case error",
			err:     signInErr,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			useCase := &mockSignInUseCase{}
			useCase.On("SignIn", ctx, name, "user@example.com", callbackURL).Return(tt.result, tt.err).Once()

			api := &API{useCases: &usecases.UseCases{SignInUseCase: useCase}}

			got, err := api.SignIn(ctx, name, "user@example.com", callbackURL)

			if tt.wantErr {
				require.ErrorIs(t, err, tt.err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Same(t, tt.result, got)
			}
			useCase.AssertExpectations(t)
		})
	}
}

func TestAPI_Verify(t *testing.T) {
	verifyErr := errors.New("verify failed")
	ipAddress := new("203.0.113.10")
	userAgent := new("test-agent")

	tests := []struct {
		name    string
		result  string
		err     error
		wantErr bool
	}{
		{
			name:   "delegates and returns the exchange token",
			result: "exchange-token",
		},
		{
			name:    "propagates the use case error",
			err:     verifyErr,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			useCase := &mockVerifyUseCase{}
			useCase.On("Verify", ctx, "raw-token", ipAddress, userAgent).Return(tt.result, tt.err).Once()

			api := &API{useCases: &usecases.UseCases{VerifyUseCase: useCase}}

			got, err := api.Verify(ctx, "raw-token", ipAddress, userAgent)

			if tt.wantErr {
				require.ErrorIs(t, err, tt.err)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.result, got)
			}
			useCase.AssertExpectations(t)
		})
	}
}

func TestAPI_Exchange(t *testing.T) {
	exchangeErr := errors.New("exchange failed")
	wantResult := &types.MagicLinkExchangeResult{}
	ipAddress := new("203.0.113.10")
	userAgent := new("test-agent")

	tests := []struct {
		name    string
		result  *types.MagicLinkExchangeResult
		err     error
		wantErr bool
	}{
		{
			name:   "delegates and returns the session result",
			result: wantResult,
		},
		{
			name:    "propagates the use case error",
			err:     exchangeErr,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			useCase := &mockExchangeUseCase{}
			useCase.On("Exchange", ctx, "exchange-token", ipAddress, userAgent).Return(tt.result, tt.err).Once()

			api := &API{useCases: &usecases.UseCases{ExchangeUseCase: useCase}}

			got, err := api.Exchange(ctx, "exchange-token", ipAddress, userAgent)

			if tt.wantErr {
				require.ErrorIs(t, err, tt.err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Same(t, tt.result, got)
			}
			useCase.AssertExpectations(t)
		})
	}
}
