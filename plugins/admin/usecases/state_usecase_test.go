package usecases_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestStateUseCase_GetUserState(t *testing.T) {
	t.Parallel()

	t.Run("empty id", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		_, err := useCase.GetUserState(context.Background(), "   ")
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards trimmed id", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
		_, err := useCase.GetUserState(context.Background(), "  u1  ")
		assert.NoError(t, err)
		userStateRepo.AssertExpectations(t)
	})
}

func TestStateUseCase_UpsertUserState(t *testing.T) {
	t.Parallel()

	t.Run("empty id", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		_, err := useCase.UpsertUserState(context.Background(), "  ", admintypes.UpsertUserStateRequest{}, nil)
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards trimmed id to service", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		// service will check user exists via impersonation repo
		impRepo.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(s *admintypes.AdminUserState) bool {
			return s.UserID == "u1" && s.IsBanned
		})).Return(nil).Once()
		userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1", IsBanned: true}, nil).Once()

		result, err := useCase.UpsertUserState(context.Background(), " u1 ", admintypes.UpsertUserStateRequest{IsBanned: true}, internaltests.PtrString("actor"))
		assert.NoError(t, err)
		assert.Equal(t, "u1", result.UserID)
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})
}

func TestStateUseCase_DeleteUserState(t *testing.T) {
	t.Parallel()

	t.Run("empty id", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		err := useCase.DeleteUserState(context.Background(), "   ")
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards trimmed id", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("Delete", mock.Anything, "u1").Return(nil).Once()
		err := useCase.DeleteUserState(context.Background(), "  u1 ")
		assert.NoError(t, err)
		userStateRepo.AssertExpectations(t)
	})
}

func TestStateUseCase_GetBannedUserStates(t *testing.T) {
	t.Parallel()

	useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
	userStateRepo.On("GetBanned", mock.Anything).Return([]admintypes.AdminUserState{{UserID: "u1", IsBanned: true}}, nil).Once()

	list, err := useCase.GetBannedUserStates(context.Background())
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	userStateRepo.AssertExpectations(t)
}

func TestStateUseCase_GetSessionState(t *testing.T) {
	t.Parallel()

	t.Run("empty id", func(t *testing.T) {
		t.Parallel()
		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		_, err := useCase.GetSessionState(context.Background(), "")
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards trimmed id", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
		_, err := useCase.GetSessionState(context.Background(), " s1 ")
		assert.NoError(t, err)
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestStateUseCase_UpsertSessionState(t *testing.T) {
	t.Parallel()

	t.Run("empty id", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		_, err := useCase.UpsertSessionState(context.Background(), "", admintypes.UpsertSessionStateRequest{}, nil)
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards trimmed id", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
		sessionStateRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(s *admintypes.AdminSessionState) bool {
			return s.SessionID == "s1" && s.RevokedAt != nil
		})).Return(nil).Once()
		sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()

		_, err := useCase.UpsertSessionState(context.Background(), " s1 ", admintypes.UpsertSessionStateRequest{Revoke: true}, internaltests.PtrString("actor"))
		assert.NoError(t, err)
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestStateUseCase_DeleteSessionState(t *testing.T) {
	t.Parallel()

	t.Run("empty id", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		err := useCase.DeleteSessionState(context.Background(), "   ")
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards trimmed id", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("Delete", mock.Anything, "s1").Return(nil).Once()
		err := useCase.DeleteSessionState(context.Background(), " s1 ")
		assert.NoError(t, err)
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestStateUseCase_GetUserAdminSessions(t *testing.T) {
	t.Parallel()

	t.Run("empty id", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		_, err := useCase.GetUserAdminSessions(context.Background(), "")
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards trimmed id with repo call", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
		impRepo.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
		sessionStateRepo.On("GetByUserID", mock.Anything, "u1").Return([]admintypes.AdminUserSession{}, nil).Once()
		_, err := useCase.GetUserAdminSessions(context.Background(), " u1 ")
		assert.NoError(t, err)
		impRepo.AssertExpectations(t)
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestStateUseCase_RevokeSession(t *testing.T) {
	t.Parallel()

	useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
	sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
	sessionStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()

	_, err := useCase.RevokeSession(context.Background(), "s1", internaltests.PtrString("reason"), internaltests.PtrString("actor"))
	assert.NoError(t, err)
	sessionStateRepo.AssertExpectations(t)
}

func TestStateUseCase_GetRevokedSessionStates(t *testing.T) {
	t.Parallel()

	useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
	sessionStateRepo.On("GetRevoked", mock.Anything).Return([]admintypes.AdminSessionState{{SessionID: "s1"}}, nil).Once()
	list, err := useCase.GetRevokedSessionStates(context.Background())
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	sessionStateRepo.AssertExpectations(t)
}

func TestStateUseCase_BanAndUnbanUser(t *testing.T) {
	t.Parallel()

	t.Run("passes id through without trimming", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		// service doesn't trim, so repository sees the same value provided
		impRepo.On("UserExists", mock.Anything, " u1 ").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
		userStateRepo.On("GetByUserID", mock.Anything, " u1 ").Return(&admintypes.AdminUserState{UserID: " u1 ", IsBanned: true}, nil).Once()
		_, err := useCase.BanUser(context.Background(), " u1 ", admintypes.BanUserRequest{}, internaltests.PtrString("actor"))
		assert.NoError(t, err)
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})

	t.Run("passes id through without trimming", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		impRepo.On("UserExists", mock.Anything, " u1 ").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
		userStateRepo.On("GetByUserID", mock.Anything, " u1 ").Return(&admintypes.AdminUserState{UserID: " u1 ", IsBanned: false}, nil).Once()
		_, err := useCase.UnbanUser(context.Background(), " u1 ")
		assert.NoError(t, err)
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})
}
