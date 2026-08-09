package usecases_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func TestStateUseCase_GetUserState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(userStateRepo *admintests.MockUserStateRepository)
		wantErr error
	}{
		{
			name:    "empty id",
			userID:  "   ",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "forwards trimmed id",
			userID: "  u1  ",
			setup: func(userStateRepo *admintests.MockUserStateRepository) {
				userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(userStateRepo)
			}

			result, err := useCase.GetUserState(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
			}
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_UpsertUserState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		request admintypes.UpsertUserStateRequest
		setup   func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:    "empty id",
			userID:  "  ",
			request: admintypes.UpsertUserStateRequest{},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "forwards trimmed id to service",
			userID:  " u1 ",
			request: admintypes.UpsertUserStateRequest{Banned: true},
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(s *admintypes.AdminUserState) bool {
					return s.UserID == "u1" && s.Banned
				})).Return(nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1", Banned: true}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(userStateRepo, impRepo)
			}

			result, err := useCase.UpsertUserState(context.Background(), internaltests.TestActor(), tt.userID, tt.request, new("actor"))
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "u1", result.UserID)
			}

			impRepo.AssertExpectations(t)
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_DeleteUserState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(userStateRepo *admintests.MockUserStateRepository)
		wantErr error
	}{
		{
			name:    "empty id",
			userID:  "   ",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "forwards trimmed id",
			userID: "  u1 ",
			setup: func(userStateRepo *admintests.MockUserStateRepository) {
				userStateRepo.On("Delete", mock.Anything, "u1").Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(userStateRepo)
			}

			err := useCase.DeleteUserState(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_GetBannedUserStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(userStateRepo *admintests.MockUserStateRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository) {
				userStateRepo.On("GetBanned", mock.Anything).Return([]admintypes.AdminUserState{{UserID: "u1", Banned: true}}, nil).Once()
			},
		},
		{
			name: "repo error",
			setup: func(userStateRepo *admintests.MockUserStateRepository) {
				userStateRepo.On("GetBanned", mock.Anything).Return(nil, coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
			tt.setup(userStateRepo)

			list, err := useCase.GetBannedUserStates(context.Background(), internaltests.TestActor())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, list)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, 1)
			}
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_GetSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		setup     func(sessionStateRepo *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "empty id",
			sessionID: "",
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "forwards trimmed id",
			sessionID: " s1 ",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(sessionStateRepo)
			}

			result, err := useCase.GetSessionState(context.Background(), internaltests.TestActor(), tt.sessionID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
			}
			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_UpsertSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		request   admintypes.UpsertSessionStateRequest
		setup     func(sessionStateRepo *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "empty id",
			sessionID: "",
			request:   admintypes.UpsertSessionStateRequest{},
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "forwards trimmed id",
			sessionID: " s1 ",
			request:   admintypes.UpsertSessionStateRequest{Revoke: true},
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(s *admintypes.AdminSessionState) bool {
					return s.SessionID == "s1" && s.RevokedAt != nil
				})).Return(nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(sessionStateRepo)
			}

			_, err := useCase.UpsertSessionState(context.Background(), internaltests.TestActor(), tt.sessionID, tt.request, new("actor"))
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_DeleteSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		setup     func(sessionStateRepo *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "empty id",
			sessionID: "   ",
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "forwards trimmed id",
			sessionID: " s1 ",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("Delete", mock.Anything, "s1").Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(sessionStateRepo)
			}

			err := useCase.DeleteSessionState(context.Background(), internaltests.TestActor(), tt.sessionID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_GetUserAdminSessions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(sessionStateRepo *admintests.MockSessionStateRepository, impRepo *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:    "empty id",
			userID:  "",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "forwards trimmed id with repo call",
			userID: " u1 ",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				sessionStateRepo.On("GetByUserID", mock.Anything, "u1").Return([]admintypes.AdminUserSession{}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(sessionStateRepo, impRepo)
			}

			_, err := useCase.GetUserAdminSessions(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			impRepo.AssertExpectations(t)
			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_RevokeSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		setup     func(sessionStateRepo *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "success",
			sessionID: "s1",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
		{
			name:      "session not found",
			sessionID: "s1",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(sessionStateRepo)
			}

			_, err := useCase.RevokeSession(context.Background(), internaltests.TestActor(), tt.sessionID, new("reason"), new("actor"))
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_GetRevokedSessionStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(sessionStateRepo *admintests.MockSessionStateRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("GetRevoked", mock.Anything).Return([]admintypes.AdminSessionState{{SessionID: "s1"}}, nil).Once()
			},
		},
		{
			name: "repo error",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("GetRevoked", mock.Anything).Return(nil, coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			tt.setup(sessionStateRepo)

			list, err := useCase.GetRevokedSessionStates(context.Background(), internaltests.TestActor())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, list)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, 1)
			}
			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_BanUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:   "trims id before passing to service",
			userID: " u1 ",
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1", Banned: true}, nil).Once()
			},
		},
		{
			name:   "user not found",
			userID: "u1",
			setup: func(_ *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "u1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(userStateRepo, impRepo)
			}

			result, err := useCase.BanUser(context.Background(), internaltests.TestActor(), tt.userID, admintypes.BanUserRequest{}, new("actor"))
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
			}

			impRepo.AssertExpectations(t)
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_UnbanUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:   "trims id before passing to service",
			userID: " u1 ",
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1", Banned: false}, nil).Once()
			},
		},
		{
			name:   "user not found",
			userID: "u1",
			setup: func(_ *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "u1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(userStateRepo, impRepo)
			}

			result, err := useCase.UnbanUser(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
			}

			impRepo.AssertExpectations(t)
			userStateRepo.AssertExpectations(t)
		})
	}
}
