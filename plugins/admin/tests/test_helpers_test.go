package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

func TestPtrTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		offset int
	}{
		{name: "zero offset returns current time", offset: 0},
		{name: "positive offset returns future time", offset: 3600},
		{name: "negative offset returns past time", offset: -3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := time.Now().Add(time.Duration(tt.offset) * time.Second)
			got := PtrTime(t, tt.offset)
			after := time.Now().Add(time.Duration(tt.offset) * time.Second)

			require.NotNil(t, got)
			assert.Equal(t, time.UTC, got.Location())
			assert.False(t, got.Before(before))
			assert.False(t, got.After(after))
		})
	}
}

func TestNewUsersUseCaseFixture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("repository failure")
	user := &models.User{ID: "user-1"}

	tests := []struct {
		name    string
		userID  string
		setup   func(repo *internaltests.MockUserRepository)
		want    *models.User
		wantErr error
	}{
		{
			name:   "returns user from repository",
			userID: "user-1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", ctx, "user-1").Return(user, nil).Once()
			},
			want: user,
		},
		{
			name:   "returns repository error",
			userID: "user-1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", ctx, "user-1").Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name:    "rejects blank user id before reaching repository",
			userID:  "   ",
			wantErr: adminconstants.ErrUserIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := NewUsersUseCaseFixture()
			if tt.setup != nil {
				tt.setup(repo)
			}

			got, err := useCase.GetByID(ctx, nil, tt.userID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			repo.AssertExpectations(t)
		})
	}
}

func TestNewAccountsUseCaseFixture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("repository failure")
	account := &models.Account{ID: "account-1", UserID: "user-1"}

	tests := []struct {
		name      string
		accountID string
		setup     func(repo *internaltests.MockAccountRepository)
		want      *models.Account
		wantErr   error
	}{
		{
			name:      "returns account from repository",
			accountID: "account-1",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByID", ctx, "account-1").Return(account, nil).Once()
			},
			want: account,
		},
		{
			name:      "returns repository error",
			accountID: "account-1",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByID", ctx, "account-1").Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name:      "rejects blank account id before reaching repository",
			accountID: "",
			wantErr:   coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, service, accountRepo, userRepo, passwordSvc := NewAccountsUseCaseFixture()
			require.NotNil(t, service)
			require.NotNil(t, userRepo)
			require.NotNil(t, passwordSvc)
			if tt.setup != nil {
				tt.setup(accountRepo)
			}

			got, err := useCase.GetByID(ctx, nil, tt.accountID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			accountRepo.AssertExpectations(t)
		})
	}
}

func TestNewAccountsServiceFixture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("repository failure")
	account := &models.Account{ID: "account-1", UserID: "user-1"}

	tests := []struct {
		name      string
		accountID string
		setup     func(repo *internaltests.MockAccountRepository)
		want      *models.Account
		wantErr   error
	}{
		{
			name:      "returns account from repository",
			accountID: "account-1",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByID", ctx, "account-1").Return(account, nil).Once()
			},
			want: account,
		},
		{
			name:      "returns repository error",
			accountID: "account-1",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByID", ctx, "account-1").Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name:      "rejects blank account id before reaching repository",
			accountID: "   ",
			wantErr:   coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, accountRepo, userRepo, passwordSvc := NewAccountsServiceFixture()
			require.NotNil(t, userRepo)
			require.NotNil(t, passwordSvc)
			if tt.setup != nil {
				tt.setup(accountRepo)
			}

			got, err := service.GetByID(ctx, nil, tt.accountID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			accountRepo.AssertExpectations(t)
		})
	}
}

func TestNewStateUseCaseFixture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("repository failure")
	state := &types.AdminUserState{UserID: "user-1"}

	tests := []struct {
		name    string
		userID  string
		setup   func(repo *MockUserStateRepository)
		want    *types.AdminUserState
		wantErr error
	}{
		{
			name:   "returns user state from repository",
			userID: "user-1",
			setup: func(repo *MockUserStateRepository) {
				repo.On("GetByUserID", ctx, "user-1").Return(state, nil).Once()
			},
			want: state,
		},
		{
			name:   "returns repository error",
			userID: "user-1",
			setup: func(repo *MockUserStateRepository) {
				repo.On("GetByUserID", ctx, "user-1").Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name:    "rejects blank user id before reaching repository",
			userID:  "",
			wantErr: coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, sessionStateRepo, impRepo := NewStateUseCaseFixture()
			require.NotNil(t, sessionStateRepo)
			require.NotNil(t, impRepo)
			if tt.setup != nil {
				tt.setup(userStateRepo)
			}

			got, err := useCase.GetUserState(ctx, nil, tt.userID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestNewStateServiceFixture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("repository failure")
	state := &types.AdminUserState{UserID: "user-1"}

	tests := []struct {
		name    string
		userID  string
		setup   func(repo *MockUserStateRepository)
		want    *types.AdminUserState
		wantErr error
	}{
		{
			name:   "returns user state from repository",
			userID: "user-1",
			setup: func(repo *MockUserStateRepository) {
				repo.On("GetByUserID", ctx, "user-1").Return(state, nil).Once()
			},
			want: state,
		},
		{
			name:   "returns repository error",
			userID: "user-1",
			setup: func(repo *MockUserStateRepository) {
				repo.On("GetByUserID", ctx, "user-1").Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name:    "rejects blank user id before reaching repository",
			userID:  "  ",
			wantErr: coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, userStateRepo, sessionStateRepo, impRepo := NewStateServiceFixture()
			require.NotNil(t, sessionStateRepo)
			require.NotNil(t, impRepo)
			if tt.setup != nil {
				tt.setup(userStateRepo)
			}

			got, err := service.GetUserState(ctx, nil, tt.userID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestNewImpersonationUseCaseFixture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("repository failure")
	impersonation := &types.Impersonation{ID: "imp-1", ActorUserID: "admin-1", TargetUserID: "user-1"}

	tests := []struct {
		name            string
		impersonationID string
		setup           func(repo *MockImpersonationRepository)
		want            *types.Impersonation
		wantErr         error
	}{
		{
			name:            "returns impersonation from repository",
			impersonationID: "imp-1",
			setup: func(repo *MockImpersonationRepository) {
				repo.On("GetImpersonationByID", ctx, "imp-1").Return(impersonation, nil).Once()
			},
			want: impersonation,
		},
		{
			name:            "returns not found when repository has no row",
			impersonationID: "imp-1",
			setup: func(repo *MockImpersonationRepository) {
				repo.On("GetImpersonationByID", ctx, "imp-1").Return(nil, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:            "returns repository error",
			impersonationID: "imp-1",
			setup: func(repo *MockImpersonationRepository) {
				repo.On("GetImpersonationByID", ctx, "imp-1").Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name:            "rejects blank impersonation id before reaching repository",
			impersonationID: "   ",
			wantErr:         coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, impRepo, sessionStateRepo, sessionSvc, tokenSvc := NewImpersonationUseCaseFixture(t)
			require.NotNil(t, sessionStateRepo)
			require.NotNil(t, sessionSvc)
			require.NotNil(t, tokenSvc)
			if tt.setup != nil {
				tt.setup(impRepo)
			}

			got, err := useCase.GetImpersonationByID(ctx, nil, tt.impersonationID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			impRepo.AssertExpectations(t)
		})
	}
}

func TestNewImpersonationServiceFixture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("repository failure")
	impersonation := &types.Impersonation{ID: "imp-1", ActorUserID: "admin-1", TargetUserID: "user-1"}

	tests := []struct {
		name            string
		impersonationID string
		setup           func(repo *MockImpersonationRepository)
		want            *types.Impersonation
		wantErr         error
	}{
		{
			name:            "returns impersonation from repository",
			impersonationID: "imp-1",
			setup: func(repo *MockImpersonationRepository) {
				repo.On("GetImpersonationByID", ctx, "imp-1").Return(impersonation, nil).Once()
			},
			want: impersonation,
		},
		{
			name:            "returns not found when repository has no row",
			impersonationID: "imp-1",
			setup: func(repo *MockImpersonationRepository) {
				repo.On("GetImpersonationByID", ctx, "imp-1").Return(nil, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:            "returns repository error",
			impersonationID: "imp-1",
			setup: func(repo *MockImpersonationRepository) {
				repo.On("GetImpersonationByID", ctx, "imp-1").Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name:            "rejects blank impersonation id before reaching repository",
			impersonationID: "",
			wantErr:         coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, impRepo, sessionStateRepo, sessSvc, tokenSvc := NewImpersonationServiceFixture()
			require.NotNil(t, sessionStateRepo)
			require.NotNil(t, sessSvc)
			require.NotNil(t, tokenSvc)
			if tt.setup != nil {
				tt.setup(impRepo)
			}

			got, err := service.GetImpersonationByID(ctx, nil, tt.impersonationID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			impRepo.AssertExpectations(t)
		})
	}
}

func TestNewAccessControlServiceFixture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("lookup failed")

	tests := []struct {
		name    string
		exists  bool
		err     error
		wantErr error
	}{
		{name: "role exists", exists: true},
		{name: "lookup fails", err: errBoom, wantErr: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := NewAccessControlServiceFixture()
			mockSvc.On("RoleExists", ctx, "admin").Return(tt.exists, tt.err).Once()
			var service rootservices.AccessControlService = mockSvc

			got, err := service.RoleExists(ctx, "admin")

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.exists, got)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestMockAccessControlService_RoleExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("lookup failed")

	tests := []struct {
		name   string
		exists bool
		err    error
	}{
		{name: "role exists", exists: true},
		{name: "role does not exist", exists: false},
		{name: "lookup fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewAccessControlServiceFixture()
			m.On("RoleExists", ctx, "admin").Return(tt.exists, tt.err).Once()

			got, err := m.RoleExists(ctx, "admin")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.exists, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockAccessControlService_GetRolePermissionsByName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("lookup failed")

	tests := []struct {
		name   string
		result any
		want   []string
		err    error
	}{
		{name: "returns permissions", result: []string{"users:read", "users:write"}, want: []string{"users:read", "users:write"}},
		{name: "returns nil when repository returns nil", result: nil, want: nil},
		{name: "lookup fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewAccessControlServiceFixture()
			m.On("GetRolePermissionsByName", ctx, "admin").Return(tt.result, tt.err).Once()

			got, err := m.GetRolePermissionsByName(ctx, "admin")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockAccessControlService_GetRoleWeightByName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("lookup failed")

	tests := []struct {
		name   string
		weight int
		err    error
	}{
		{name: "returns role weight", weight: 100},
		{name: "lookup fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewAccessControlServiceFixture()
			m.On("GetRoleWeightByName", ctx, "admin").Return(tt.weight, tt.err).Once()

			got, err := m.GetRoleWeightByName(ctx, "admin")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.weight, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockAccessControlService_ValidatePermissionKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("invalid permission")

	tests := []struct {
		name string
		err  error
	}{
		{name: "valid permission keys", err: nil},
		{name: "invalid permission keys", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewAccessControlServiceFixture()
			keys := []string{"users:read"}
			m.On("ValidatePermissionKeys", ctx, keys).Return(tt.err).Once()

			err := m.ValidatePermissionKeys(ctx, keys)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockAccessControlService_EnsurePermissions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")

	tests := []struct {
		name string
		err  error
	}{
		{name: "permissions ensured", err: nil},
		{name: "permissions fail to persist", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewAccessControlServiceFixture()
			permissions := []rootservices.PermissionDefinition{{Key: "users:read", Description: "Read users"}}
			m.On("EnsurePermissions", ctx, permissions).Return(tt.err).Once()

			err := m.EnsurePermissions(ctx, permissions)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockPasswordService_Hash(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("hashing failed")

	tests := []struct {
		name string
		hash string
		err  error
	}{
		{name: "returns hashed password", hash: "hashed-secret"},
		{name: "hashing fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockPasswordService{}
			m.On("Hash", "secret").Return(tt.hash, tt.err).Once()

			got, err := m.Hash("secret")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.hash, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockPasswordService_Verify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result bool
	}{
		{name: "password matches", result: true},
		{name: "password does not match", result: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockPasswordService{}
			m.On("Verify", "secret", "hashed-secret").Return(tt.result).Once()

			got := m.Verify("secret", "hashed-secret")

			assert.Equal(t, tt.result, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockUserStateRepository_GetByUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	state := &types.AdminUserState{UserID: "user-1", Banned: true}

	tests := []struct {
		name   string
		result any
		want   *types.AdminUserState
		err    error
	}{
		{name: "returns user state", result: state, want: state},
		{name: "returns nil when state is missing", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockUserStateRepository{}
			m.On("GetByUserID", ctx, "user-1").Return(tt.result, tt.err).Once()

			got, err := m.GetByUserID(ctx, "user-1")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockUserStateRepository_Upsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")
	state := &types.AdminUserState{UserID: "user-1"}

	tests := []struct {
		name string
		err  error
	}{
		{name: "upsert succeeds", err: nil},
		{name: "upsert fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockUserStateRepository{}
			m.On("Upsert", ctx, state).Return(tt.err).Once()

			err := m.Upsert(ctx, state)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockUserStateRepository_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")
	state := &types.AdminUserState{UserID: "user-1"}

	tests := []struct {
		name string
		err  error
	}{
		{name: "create succeeds", err: nil},
		{name: "create fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockUserStateRepository{}
			m.On("Create", ctx, state).Return(tt.err).Once()

			err := m.Create(ctx, state)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockUserStateRepository_Update(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")
	state := &types.AdminUserState{UserID: "user-1"}

	tests := []struct {
		name string
		err  error
	}{
		{name: "update succeeds", err: nil},
		{name: "update fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockUserStateRepository{}
			m.On("Update", ctx, state).Return(tt.err).Once()

			err := m.Update(ctx, state)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockUserStateRepository_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("delete failed")

	tests := []struct {
		name string
		err  error
	}{
		{name: "delete succeeds", err: nil},
		{name: "delete fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockUserStateRepository{}
			m.On("Delete", ctx, "user-1").Return(tt.err).Once()

			err := m.Delete(ctx, "user-1")

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockUserStateRepository_GetBanned(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	banned := []types.AdminUserState{{UserID: "user-1", Banned: true}}

	tests := []struct {
		name   string
		result any
		want   []types.AdminUserState
		err    error
	}{
		{name: "returns banned users", result: banned, want: banned},
		{name: "returns nil when there are no banned users", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockUserStateRepository{}
			m.On("GetBanned", ctx).Return(tt.result, tt.err).Once()

			got, err := m.GetBanned(ctx)

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockSessionStateRepository_GetBySessionID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	state := &types.AdminSessionState{SessionID: "session-1"}

	tests := []struct {
		name   string
		result any
		want   *types.AdminSessionState
		err    error
	}{
		{name: "returns session state", result: state, want: state},
		{name: "returns nil when state is missing", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockSessionStateRepository{}
			m.On("GetBySessionID", ctx, "session-1").Return(tt.result, tt.err).Once()

			got, err := m.GetBySessionID(ctx, "session-1")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockSessionStateRepository_Upsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")
	state := &types.AdminSessionState{SessionID: "session-1"}

	tests := []struct {
		name string
		err  error
	}{
		{name: "upsert succeeds", err: nil},
		{name: "upsert fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockSessionStateRepository{}
			m.On("Upsert", ctx, state).Return(tt.err).Once()

			err := m.Upsert(ctx, state)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockSessionStateRepository_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")
	state := &types.AdminSessionState{SessionID: "session-1"}

	tests := []struct {
		name string
		err  error
	}{
		{name: "create succeeds", err: nil},
		{name: "create fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockSessionStateRepository{}
			m.On("Create", ctx, state).Return(tt.err).Once()

			err := m.Create(ctx, state)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockSessionStateRepository_Update(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")
	state := &types.AdminSessionState{SessionID: "session-1"}

	tests := []struct {
		name string
		err  error
	}{
		{name: "update succeeds", err: nil},
		{name: "update fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockSessionStateRepository{}
			m.On("Update", ctx, state).Return(tt.err).Once()

			err := m.Update(ctx, state)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockSessionStateRepository_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("delete failed")

	tests := []struct {
		name string
		err  error
	}{
		{name: "delete succeeds", err: nil},
		{name: "delete fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockSessionStateRepository{}
			m.On("Delete", ctx, "session-1").Return(tt.err).Once()

			err := m.Delete(ctx, "session-1")

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockSessionStateRepository_GetRevoked(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	revoked := []types.AdminSessionState{{SessionID: "session-1"}}

	tests := []struct {
		name   string
		result any
		want   []types.AdminSessionState
		err    error
	}{
		{name: "returns revoked sessions", result: revoked, want: revoked},
		{name: "returns nil when there are no revoked sessions", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockSessionStateRepository{}
			m.On("GetRevoked", ctx).Return(tt.result, tt.err).Once()

			got, err := m.GetRevoked(ctx)

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockSessionStateRepository_SessionExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")

	tests := []struct {
		name   string
		exists bool
		err    error
	}{
		{name: "session exists", exists: true},
		{name: "session does not exist", exists: false},
		{name: "lookup fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockSessionStateRepository{}
			m.On("SessionExists", ctx, "session-1").Return(tt.exists, tt.err).Once()

			got, err := m.SessionExists(ctx, "session-1")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.exists, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockSessionStateRepository_GetByUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	sessions := []types.AdminUserSession{{State: &types.AdminSessionState{SessionID: "session-1"}}}

	tests := []struct {
		name   string
		result any
		want   []types.AdminUserSession
		err    error
	}{
		{name: "returns user sessions", result: sessions, want: sessions},
		{name: "returns nil when user has no sessions", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockSessionStateRepository{}
			m.On("GetByUserID", ctx, "user-1").Return(tt.result, tt.err).Once()

			got, err := m.GetByUserID(ctx, "user-1")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockImpersonationRepository_UserExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")

	tests := []struct {
		name   string
		exists bool
		err    error
	}{
		{name: "user exists", exists: true},
		{name: "user does not exist", exists: false},
		{name: "lookup fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockImpersonationRepository{}
			m.On("UserExists", ctx, "user-1").Return(tt.exists, tt.err).Once()

			got, err := m.UserExists(ctx, "user-1")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.exists, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockImpersonationRepository_CreateImpersonation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")
	impersonation := &types.Impersonation{ID: "imp-1"}

	tests := []struct {
		name string
		err  error
	}{
		{name: "create succeeds", err: nil},
		{name: "create fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockImpersonationRepository{}
			m.On("CreateImpersonation", ctx, impersonation).Return(tt.err).Once()

			err := m.CreateImpersonation(ctx, impersonation)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockImpersonationRepository_GetActiveImpersonationByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	impersonation := &types.Impersonation{ID: "imp-1"}

	tests := []struct {
		name   string
		result any
		want   *types.Impersonation
		err    error
	}{
		{name: "returns active impersonation", result: impersonation, want: impersonation},
		{name: "returns nil when impersonation is not active", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockImpersonationRepository{}
			m.On("GetActiveImpersonationByID", ctx, "imp-1").Return(tt.result, tt.err).Once()

			got, err := m.GetActiveImpersonationByID(ctx, "imp-1")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockImpersonationRepository_GetLatestActiveImpersonationByActor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	impersonation := &types.Impersonation{ID: "imp-1", ActorUserID: "admin-1"}

	tests := []struct {
		name   string
		result any
		want   *types.Impersonation
		err    error
	}{
		{name: "returns latest active impersonation", result: impersonation, want: impersonation},
		{name: "returns nil when actor has no active impersonation", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockImpersonationRepository{}
			m.On("GetLatestActiveImpersonationByActor", ctx, "admin-1").Return(tt.result, tt.err).Once()

			got, err := m.GetLatestActiveImpersonationByActor(ctx, "admin-1")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockImpersonationRepository_EndImpersonation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("write failed")
	endedBy := new("admin-1")

	tests := []struct {
		name string
		err  error
	}{
		{name: "end succeeds", err: nil},
		{name: "end fails", err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockImpersonationRepository{}
			m.On("EndImpersonation", ctx, "imp-1", endedBy).Return(tt.err).Once()

			err := m.EndImpersonation(ctx, "imp-1", endedBy)

			require.ErrorIs(t, err, tt.err)
			m.AssertExpectations(t)
		})
	}
}

func TestMockImpersonationRepository_GetAllImpersonations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	impersonations := []types.Impersonation{{ID: "imp-1"}, {ID: "imp-2"}}

	tests := []struct {
		name   string
		result any
		want   []types.Impersonation
		err    error
	}{
		{name: "returns all impersonations", result: impersonations, want: impersonations},
		{name: "returns nil when there are no impersonations", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockImpersonationRepository{}
			m.On("GetAllImpersonations", ctx).Return(tt.result, tt.err).Once()

			got, err := m.GetAllImpersonations(ctx)

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestMockImpersonationRepository_GetImpersonationByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errBoom := errors.New("read failed")
	impersonation := &types.Impersonation{ID: "imp-1"}

	tests := []struct {
		name   string
		result any
		want   *types.Impersonation
		err    error
	}{
		{name: "returns impersonation", result: impersonation, want: impersonation},
		{name: "returns nil when impersonation is missing", result: nil, want: nil},
		{name: "read fails", result: nil, err: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &MockImpersonationRepository{}
			m.On("GetImpersonationByID", ctx, "imp-1").Return(tt.result, tt.err).Once()

			got, err := m.GetImpersonationByID(ctx, "imp-1")

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, got)
			m.AssertExpectations(t)
		})
	}
}

func TestBuildAccountModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		id         string
		userID     string
		providerID string
		accountID  string
		want       *models.Account
	}{
		{
			name:       "maps all fields",
			id:         "account-1",
			userID:     "user-1",
			providerID: "credential",
			accountID:  "provider-account-1",
			want:       &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "credential", AccountID: "provider-account-1"},
		},
		{
			name: "allows empty fields",
			want: &models.Account{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := BuildAccountModel(tt.id, tt.userID, tt.providerID, tt.accountID)

			assert.Equal(t, tt.want, got)
		})
	}
}
