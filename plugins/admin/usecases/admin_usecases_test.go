package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
	"github.com/Authula/authula/plugins/admin/usecases"
)

func TestNewAdminUseCases_WiresAllUseCases(t *testing.T) {
	t.Parallel()

	userRepo := &internaltests.MockUserRepository{}
	accountRepo := &internaltests.MockAccountRepository{}
	userStateRepo := &admintests.MockUserStateRepository{}
	sessionStateRepo := &admintests.MockSessionStateRepository{}
	impersonationRepo := &admintests.MockImpersonationRepository{}
	sessionSvc := &internaltests.MockSessionService{}
	tokenSvc := &internaltests.MockTokenService{}
	passwordSvc := &admintests.MockPasswordService{}

	adminUseCases := usecases.NewAdminUseCases(
		adminservices.NewUsersService(userRepo),
		adminservices.NewAccountsService(accountRepo, userRepo, passwordSvc),
		adminservices.NewStateService(userStateRepo, sessionStateRepo, impersonationRepo),
		adminservices.NewImpersonationService(impersonationRepo, sessionStateRepo, sessionSvc, tokenSvc, time.Minute, time.Minute),
		&internaltests.NoopAuthorizer{},
	)
	require.NotNil(t, adminUseCases)

	assert.NotEqual(t, usecases.UsersUseCase{}, adminUseCases.UsersUseCase())
	assert.NotEqual(t, usecases.AccountsUseCase{}, adminUseCases.AccountsUseCase())
	assert.NotEqual(t, usecases.StateUseCase{}, adminUseCases.StateUseCase())
	assert.NotEqual(t, usecases.ImpersonationUseCase{}, adminUseCases.ImpersonationUseCase())

	userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	got, err := adminUseCases.UsersUseCase().GetByID(context.Background(), internaltests.TestActor(), "u1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "u1", got.ID)

	userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&types.AdminUserState{UserID: "u1"}, nil).Once()
	state, err := adminUseCases.StateUseCase().GetUserState(context.Background(), internaltests.TestActor(), "u1")
	require.NoError(t, err)
	require.NotNil(t, state)

	userRepo.AssertExpectations(t)
	userStateRepo.AssertExpectations(t)
}
