package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	adminservices "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

func PtrString(t *testing.T, s string) *string {
	t.Helper()
	return &s
}

func PtrTime(t *testing.T, offset int) *time.Time {
	t.Helper()
	tm := time.Now().Add(time.Duration(offset) * time.Second).UTC()
	return &tm
}

func NewUsersUseCaseFixture() (usecases.UsersUseCase, *internaltests.MockUserRepository) {
	mockUserRepo := &internaltests.MockUserRepository{}
	service := adminservices.NewUsersService(mockUserRepo)
	return usecases.NewUsersUseCase(service), mockUserRepo
}

type MockPasswordService struct {
	mock.Mock
}

func (m *MockPasswordService) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordService) Verify(password, encoded string) bool {
	args := m.Called(password, encoded)
	return args.Bool(0)
}

func NewAccountsUseCaseFixture() (usecases.AccountsUseCase, *adminservices.AccountsService, *internaltests.MockAccountRepository, *internaltests.MockUserRepository, *MockPasswordService) {
	accountRepo := &internaltests.MockAccountRepository{}
	userRepo := &internaltests.MockUserRepository{}
	passwordSvc := &MockPasswordService{}
	service := adminservices.NewAccountsService(accountRepo, userRepo, passwordSvc)
	return usecases.NewAccountsUseCase(service), service, accountRepo, userRepo, passwordSvc
}

func NewAccountsServiceFixture() (*adminservices.AccountsService, *internaltests.MockAccountRepository, *internaltests.MockUserRepository, *MockPasswordService) {
	accountRepo := &internaltests.MockAccountRepository{}
	userRepo := &internaltests.MockUserRepository{}
	passwordSvc := &MockPasswordService{}
	return adminservices.NewAccountsService(accountRepo, userRepo, passwordSvc), accountRepo, userRepo, passwordSvc
}

type MockUserStateRepository struct {
	mock.Mock
}

func (m *MockUserStateRepository) GetByUserID(ctx context.Context, userID string) (*types.AdminUserState, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminUserState), args.Error(1)
}

func (m *MockUserStateRepository) Upsert(ctx context.Context, state *types.AdminUserState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockUserStateRepository) Delete(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserStateRepository) GetBanned(ctx context.Context) ([]types.AdminUserState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.AdminUserState), args.Error(1)
}

type MockSessionStateRepository struct {
	mock.Mock
}

func (m *MockSessionStateRepository) GetBySessionID(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminSessionState), args.Error(1)
}

func (m *MockSessionStateRepository) Upsert(ctx context.Context, state *types.AdminSessionState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockSessionStateRepository) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionStateRepository) GetRevoked(ctx context.Context) ([]types.AdminSessionState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.AdminSessionState), args.Error(1)
}

func (m *MockSessionStateRepository) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	args := m.Called(ctx, sessionID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSessionStateRepository) GetByUserID(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.AdminUserSession), args.Error(1)
}

type MockImpersonationRepository struct {
	mock.Mock
}

func (m *MockImpersonationRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockImpersonationRepository) CreateImpersonation(ctx context.Context, impersonation *types.Impersonation) error {
	args := m.Called(ctx, impersonation)
	return args.Error(0)
}

func (m *MockImpersonationRepository) GetActiveImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	args := m.Called(ctx, impersonationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Impersonation), args.Error(1)
}

func (m *MockImpersonationRepository) GetLatestActiveImpersonationByActor(ctx context.Context, actorUserID string) (*types.Impersonation, error) {
	args := m.Called(ctx, actorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Impersonation), args.Error(1)
}

func (m *MockImpersonationRepository) EndImpersonation(ctx context.Context, impersonationID string, endedByUserID *string) error {
	args := m.Called(ctx, impersonationID, endedByUserID)
	return args.Error(0)
}

func (m *MockImpersonationRepository) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Impersonation), args.Error(1)
}

func (m *MockImpersonationRepository) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	args := m.Called(ctx, impersonationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Impersonation), args.Error(1)
}

func NewImpersonationUseCaseFixture(t *testing.T) (usecases.ImpersonationUseCase, *MockImpersonationRepository, *MockSessionStateRepository, *internaltests.MockSessionService, *internaltests.MockTokenService) {
	userStateRepo := &MockUserStateRepository{}
	impRepo := &MockImpersonationRepository{}
	sessionStateRepo := &MockSessionStateRepository{}
	sessionSvc := &internaltests.MockSessionService{}
	tokenSvc := &internaltests.MockTokenService{}
	stateService := adminservices.NewStateService(userStateRepo, sessionStateRepo, impRepo)
	service := adminservices.NewImpersonationService(impRepo, sessionStateRepo, sessionSvc, tokenSvc, 15*time.Minute, 15*time.Minute)
	return usecases.NewImpersonationUseCase(stateService, service), impRepo, sessionStateRepo, sessionSvc, tokenSvc
}

func NewStateUseCaseFixture() (usecases.StateUseCase, *MockUserStateRepository, *MockSessionStateRepository, *MockImpersonationRepository) {
	userStateRepo := &MockUserStateRepository{}
	sessionStateRepo := &MockSessionStateRepository{}
	impRepo := &MockImpersonationRepository{}
	service := adminservices.NewStateService(userStateRepo, sessionStateRepo, impRepo)
	return usecases.NewStateUseCase(service), userStateRepo, sessionStateRepo, impRepo
}

func BuildAccountModel(id, userID, providerID, accountID string) *models.Account {
	return &models.Account{ID: id, UserID: userID, ProviderID: providerID, AccountID: accountID}
}

// helper for constructing service directly with mocks
func NewStateServiceFixture() (*adminservices.StateService, *MockUserStateRepository, *MockSessionStateRepository, *MockImpersonationRepository) {
	userStateRepo := &MockUserStateRepository{}
	sessionStateRepo := &MockSessionStateRepository{}
	impRepo := &MockImpersonationRepository{}
	return adminservices.NewStateService(userStateRepo, sessionStateRepo, impRepo), userStateRepo, sessionStateRepo, impRepo
}

// impersonation service fixture returns service and all repos + helpers
func NewImpersonationServiceFixture() (*adminservices.ImpersonationService, *MockImpersonationRepository, *MockSessionStateRepository, *internaltests.MockSessionService, *internaltests.MockTokenService) {
	impRepo := &MockImpersonationRepository{}
	sessionStateRepo := &MockSessionStateRepository{}
	sessSvc := &internaltests.MockSessionService{}
	tokenSvc := &internaltests.MockTokenService{}
	service := adminservices.NewImpersonationService(impRepo, sessionStateRepo, sessSvc, tokenSvc, 15*time.Minute, 15*time.Minute)
	return service, impRepo, sessionStateRepo, sessSvc, tokenSvc
}
