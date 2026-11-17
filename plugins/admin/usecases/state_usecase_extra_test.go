package usecases_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
	"github.com/Authula/authula/plugins/admin/usecases"
)

func TestStateUseCase_AuthorizerDenies(t *testing.T) {
	t.Parallel()

	denied := coreerrors.ErrForbidden
	ctx := context.Background()

	tests := []struct {
		name  string
		scope string
		call  func(uc usecases.StateUseCase) error
	}{
		{
			name:  "get user state",
			scope: adminconstants.UserStateReadPermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.GetUserState(ctx, internaltests.TestActor(), "u1")
				return err
			},
		},
		{
			name:  "upsert user state",
			scope: adminconstants.UserStateUpdatePermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.UpsertUserState(ctx, internaltests.TestActor(), "u1", admintypes.UpsertUserStateRequest{}, nil)
				return err
			},
		},
		{
			name:  "create user state",
			scope: adminconstants.UserStateCreatePermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.CreateUserState(ctx, internaltests.TestActor(), "u1", admintypes.CreateUserStateRequest{}, nil)
				return err
			},
		},
		{
			name:  "update user state",
			scope: adminconstants.UserStateUpdatePermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.UpdateUserState(ctx, internaltests.TestActor(), "u1", admintypes.UpsertUserStateRequest{}, nil)
				return err
			},
		},
		{
			name:  "delete user state",
			scope: adminconstants.UserStateDeletePermission,
			call: func(uc usecases.StateUseCase) error {
				return uc.DeleteUserState(ctx, internaltests.TestActor(), "u1")
			},
		},
		{
			name:  "get banned user states",
			scope: adminconstants.UserStateListBannedPermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.GetBannedUserStates(ctx, internaltests.TestActor())
				return err
			},
		},
		{
			name:  "get session state",
			scope: adminconstants.SessionStateReadPermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.GetSessionState(ctx, internaltests.TestActor(), "s1")
				return err
			},
		},
		{
			name:  "upsert session state",
			scope: adminconstants.SessionStateUpdatePermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.UpsertSessionState(ctx, internaltests.TestActor(), "s1", admintypes.UpsertSessionStateRequest{}, nil)
				return err
			},
		},
		{
			name:  "create session state",
			scope: adminconstants.SessionStateCreatePermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.CreateSessionState(ctx, internaltests.TestActor(), "s1", admintypes.CreateSessionStateRequest{}, nil)
				return err
			},
		},
		{
			name:  "update session state",
			scope: adminconstants.SessionStateUpdatePermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.UpdateSessionState(ctx, internaltests.TestActor(), "s1", admintypes.UpsertSessionStateRequest{}, nil)
				return err
			},
		},
		{
			name:  "delete session state",
			scope: adminconstants.SessionStateDeletePermission,
			call: func(uc usecases.StateUseCase) error {
				return uc.DeleteSessionState(ctx, internaltests.TestActor(), "s1")
			},
		},
		{
			name:  "get user admin sessions",
			scope: adminconstants.UserStateListSessionsPermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.GetUserAdminSessions(ctx, internaltests.TestActor(), "u1")
				return err
			},
		},
		{
			name:  "revoke session",
			scope: adminconstants.SessionStateRevokePermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.RevokeSession(ctx, internaltests.TestActor(), "s1", nil, nil)
				return err
			},
		},
		{
			name:  "get revoked session states",
			scope: adminconstants.SessionStateListRevokedPermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.GetRevokedSessionStates(ctx, internaltests.TestActor())
				return err
			},
		},
		{
			name:  "ban user",
			scope: adminconstants.UserStateBanPermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.BanUser(ctx, internaltests.TestActor(), "u1", admintypes.BanUserRequest{}, nil)
				return err
			},
		},
		{
			name:  "unban user",
			scope: adminconstants.UserStateUnbanPermission,
			call: func(uc usecases.StateUseCase) error {
				_, err := uc.UnbanUser(ctx, internaltests.TestActor(), "u1")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authorizer := &internaltests.MockAuthorizer{}
			authorizer.On("AuthorizeScope", mock.Anything, mock.Anything, tt.scope).Return(denied).Once()
			uc := usecases.NewStateUseCase(nil, authorizer)

			assert.ErrorIs(t, tt.call(uc), denied)
			authorizer.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_CreateUserState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:    "service validation error propagates",
			setup:   func(_ *admintests.MockUserStateRepository, _ *admintests.MockImpersonationRepository) {},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "success",
			setup: func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				usr.On("GetByUserID", mock.Anything, "u1").Return((*admintypes.AdminUserState)(nil), nil).Once()
				usr.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, usr, _, imp := admintests.NewStateUseCaseFixture()
			tt.setup(usr, imp)

			userID := "u1"
			if tt.wantErr != nil {
				userID = " "
			}

			state, err := uc.CreateUserState(context.Background(), internaltests.TestActor(), userID, admintypes.CreateUserStateRequest{}, nil)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, state)
			} else {
				require.NoError(t, err)
				require.NotNil(t, state)
				assert.Equal(t, "u1", state.UserID)
			}

			imp.AssertExpectations(t)
			usr.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_UpdateUserState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name: "missing id",
			setup: func(_ *admintests.MockUserStateRepository, _ *admintests.MockImpersonationRepository) {
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "not found",
			setup: func(_ *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name: "success",
			setup: func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
				usr.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
				usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1", Banned: true}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, usr, _, imp := admintests.NewStateUseCaseFixture()
			tt.setup(usr, imp)

			userID := "u1"
			if tt.wantErr == coreerrors.ErrBadRequest {
				userID = " "
			}

			state, err := uc.UpdateUserState(context.Background(), internaltests.TestActor(), userID, admintypes.UpsertUserStateRequest{Banned: true}, nil)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, state)
			} else {
				require.NoError(t, err)
				require.NotNil(t, state)
			}

			imp.AssertExpectations(t)
			usr.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_CreateSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(sess *admintests.MockSessionStateRepository)
		wantErr error
	}{
		{
			name:    "missing id",
			setup:   func(_ *admintests.MockSessionStateRepository) {},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "success",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
				sess.On("GetBySessionID", mock.Anything, "s1").Return((*admintypes.AdminSessionState)(nil), nil).Once()
				sess.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, _, sess, _ := admintests.NewStateUseCaseFixture()
			tt.setup(sess)

			sessionID := "s1"
			if tt.wantErr != nil {
				sessionID = ""
			}

			state, err := uc.CreateSessionState(context.Background(), internaltests.TestActor(), sessionID, admintypes.CreateSessionStateRequest{}, nil)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, state)
			} else {
				require.NoError(t, err)
				require.NotNil(t, state)
				assert.Equal(t, "s1", state.SessionID)
			}

			sess.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_UpdateSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(sess *admintests.MockSessionStateRepository)
		wantErr error
	}{
		{
			name:    "missing id",
			setup:   func(_ *admintests.MockSessionStateRepository) {},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "success",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
				sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
				sess.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
				sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, _, sess, _ := admintests.NewStateUseCaseFixture()
			tt.setup(sess)

			sessionID := "s1"
			if tt.wantErr != nil {
				sessionID = " "
			}

			state, err := uc.UpdateSessionState(context.Background(), internaltests.TestActor(), sessionID, admintypes.UpsertSessionStateRequest{Revoke: true}, nil)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, state)
			} else {
				require.NoError(t, err)
				require.NotNil(t, state)
			}

			sess.AssertExpectations(t)
		})
	}
}

func TestStateUseCase_GetSelfSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		setup     func(sess *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "missing id",
			sessionID: "  ",
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "success does not require authorization",
			sessionID: " s1 ",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, _, sess, _ := admintests.NewStateUseCaseFixture()
			if tt.setup != nil {
				tt.setup(sess)
			}

			state, err := uc.GetSelfSessionState(context.Background(), tt.sessionID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, state)
			} else {
				require.NoError(t, err)
				require.NotNil(t, state)
				assert.Equal(t, "s1", state.SessionID)
			}

			sess.AssertExpectations(t)
		})
	}
}
