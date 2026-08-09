package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func newStateServiceFixture() (*adminservices.StateService, *admintests.MockUserStateRepository, *admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository) {
	return admintests.NewStateServiceFixture()
}

func TestStateService_GetUserState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(usr *admintests.MockUserStateRepository)
		wantErr error
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(usr *admintests.MockUserStateRepository) {
				usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
			},
		},
		{
			name:    "missing user id",
			userID:  "   ",
			wantErr: coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, usr, _, _ := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(usr)
			}

			state, err := svc.GetUserState(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, state)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, state.UserID)
			}
			usr.AssertExpectations(t)
		})
	}
}

func TestStateService_UpsertUserState(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	upsertErr := errors.New("some error")

	tests := []struct {
		name    string
		userID  string
		request admintypes.UpsertUserStateRequest
		actor   *string
		setup   func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:    "missing user id",
			userID:  "   ",
			request: admintypes.UpsertUserStateRequest{},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "user not found",
			userID:  "u1",
			request: admintypes.UpsertUserStateRequest{},
			setup: func(_ *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "ban with details",
			userID: "u1",
			request: admintypes.UpsertUserStateRequest{
				Banned:       true,
				BannedUntil:  &now,
				BannedReason: new("reason"),
			},
			actor: new("actor"),
			setup: func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				usr.On("Upsert", mock.Anything, mock.MatchedBy(func(s *admintypes.AdminUserState) bool {
					return s.Banned &&
						*s.BannedByUserID == "actor" &&
						s.BannedReason != nil && *s.BannedReason == "reason" &&
						s.BannedUntil.Equal(now)
				})).Return(nil).Once()
				usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
			},
		},
		{
			name:   "unban",
			userID: "u1",
			request: admintypes.UpsertUserStateRequest{
				Banned: false,
			},
			setup: func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				usr.On("Upsert", mock.Anything, mock.MatchedBy(func(s *admintypes.AdminUserState) bool {
					return !s.Banned
				})).Return(nil).Once()
				usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
			},
		},
		{
			name:    "repo error",
			userID:  "u1",
			request: admintypes.UpsertUserStateRequest{Banned: false},
			setup: func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				usr.On("Upsert", mock.Anything, mock.Anything).Return(upsertErr).Once()
			},
			wantErr: upsertErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, usr, _, imp := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(usr, imp)
			}

			_, err := svc.UpsertUserState(context.Background(), internaltests.TestActor(), tt.userID, tt.request, tt.actor)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			imp.AssertExpectations(t)
			usr.AssertExpectations(t)
		})
	}
}

func TestStateService_DeleteUserState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(usr *admintests.MockUserStateRepository)
		wantErr error
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(usr *admintests.MockUserStateRepository) {
				usr.On("Delete", mock.Anything, "u1").Return(nil).Once()
			},
		},
		{
			name:    "missing user id",
			userID:  "   ",
			wantErr: coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, usr, _, _ := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(usr)
			}

			err := svc.DeleteUserState(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			usr.AssertExpectations(t)
		})
	}
}

func TestStateService_GetBannedUserStates(t *testing.T) {
	t.Parallel()

	someErr := errors.New("some error")

	tests := []struct {
		name    string
		setup   func(usr *admintests.MockUserStateRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(usr *admintests.MockUserStateRepository) {
				usr.On("GetBanned", mock.Anything).Return([]admintypes.AdminUserState{{UserID: "u1", Banned: true}}, nil).Once()
			},
		},
		{
			name: "repo error",
			setup: func(usr *admintests.MockUserStateRepository) {
				usr.On("GetBanned", mock.Anything).Return(nil, someErr).Once()
			},
			wantErr: someErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, usr, _, _ := newStateServiceFixture()
			tt.setup(usr)

			list, err := svc.GetBannedUserStates(context.Background(), internaltests.TestActor())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, list)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, 1)
			}
			usr.AssertExpectations(t)
		})
	}
}

func TestStateService_GetSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		setup     func(sess *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "success",
			sessionID: "s1",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
		{
			name:      "missing session id",
			sessionID: "",
			wantErr:   coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, sess, _ := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(sess)
			}

			res, err := svc.GetSessionState(context.Background(), internaltests.TestActor(), tt.sessionID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.sessionID, res.SessionID)
			}
			sess.AssertExpectations(t)
		})
	}
}

func TestStateService_UpsertSessionState(t *testing.T) {
	t.Parallel()

	someErr := errors.New("fail")

	tests := []struct {
		name      string
		sessionID string
		request   admintypes.UpsertSessionStateRequest
		actor     *string
		setup     func(sess *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "missing session id",
			sessionID: "",
			request:   admintypes.UpsertSessionStateRequest{},
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "session not found",
			sessionID: "s1",
			request:   admintypes.UpsertSessionStateRequest{},
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("SessionExists", mock.Anything, "s1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:      "revoke",
			sessionID: "s1",
			request:   admintypes.UpsertSessionStateRequest{Revoke: true, RevokedReason: new("r")},
			actor:     new("actor"),
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
				sess.On("Upsert", mock.Anything, mock.MatchedBy(func(s *admintypes.AdminSessionState) bool {
					return s.RevokedAt != nil && s.RevokedByUserID != nil && *s.RevokedByUserID == "actor"
				})).Return(nil).Once()
				sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
		{
			name:      "repo failure",
			sessionID: "s1",
			request:   admintypes.UpsertSessionStateRequest{},
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
				sess.On("Upsert", mock.Anything, mock.Anything).Return(someErr).Once()
			},
			wantErr: someErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, sess, _ := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(sess)
			}

			_, err := svc.UpsertSessionState(context.Background(), internaltests.TestActor(), tt.sessionID, tt.request, tt.actor)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			sess.AssertExpectations(t)
		})
	}
}

func TestStateService_DeleteSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		setup     func(sess *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "success",
			sessionID: "s1",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("Delete", mock.Anything, "s1").Return(nil).Once()
			},
		},
		{
			name:      "missing session id",
			sessionID: "",
			wantErr:   coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, sess, _ := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(sess)
			}

			err := svc.DeleteSessionState(context.Background(), internaltests.TestActor(), tt.sessionID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			sess.AssertExpectations(t)
		})
	}
}

func TestStateService_GetUserAdminSessions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(sess *admintests.MockSessionStateRepository, imp *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(sess *admintests.MockSessionStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				sess.On("GetByUserID", mock.Anything, "u1").Return([]admintypes.AdminUserSession{}, nil).Once()
			},
		},
		{
			name:   "user not found",
			userID: "u1",
			setup: func(_ *admintests.MockSessionStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:    "missing user id",
			userID:  "   ",
			wantErr: coreerrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, sess, imp := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(sess, imp)
			}

			_, err := svc.GetUserAdminSessions(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			imp.AssertExpectations(t)
			sess.AssertExpectations(t)
		})
	}
}

func TestStateService_RevokeSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		setup     func(sess *admintests.MockSessionStateRepository)
		wantErr   error
	}{
		{
			name:      "success",
			sessionID: "s1",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
				sess.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
			},
		},
		{
			name:      "session not found",
			sessionID: "s1",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("SessionExists", mock.Anything, "s1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, sess, _ := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(sess)
			}

			_, err := svc.RevokeSession(context.Background(), internaltests.TestActor(), tt.sessionID, new("reason"), new("actor"))
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			sess.AssertExpectations(t)
		})
	}
}

func TestStateService_GetRevokedSessionStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(sess *admintests.MockSessionStateRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("GetRevoked", mock.Anything).Return([]admintypes.AdminSessionState{{SessionID: "s1"}}, nil).Once()
			},
		},
		{
			name: "repo error",
			setup: func(sess *admintests.MockSessionStateRepository) {
				sess.On("GetRevoked", mock.Anything).Return(nil, coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, sess, _ := newStateServiceFixture()
			tt.setup(sess)

			list, err := svc.GetRevokedSessionStates(context.Background(), internaltests.TestActor())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, list)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, 1)
			}
			sess.AssertExpectations(t)
		})
	}
}

func TestStateService_BanUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				usr.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1", Banned: true}, nil).Once()
			},
		},
		{
			name:   "user not found",
			userID: "u1",
			setup: func(_ *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, usr, _, imp := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(usr, imp)
			}

			_, err := svc.BanUser(context.Background(), internaltests.TestActor(), tt.userID, admintypes.BanUserRequest{Reason: new("r")}, new("actor"))
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			imp.AssertExpectations(t)
			usr.AssertExpectations(t)
		})
	}
}

func TestStateService_UnbanUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(usr *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
				usr.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1", Banned: false}, nil).Once()
			},
		},
		{
			name:   "user not found",
			userID: "u1",
			setup: func(_ *admintests.MockUserStateRepository, imp *admintests.MockImpersonationRepository) {
				imp.On("UserExists", mock.Anything, "u1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, usr, _, imp := newStateServiceFixture()
			if tt.setup != nil {
				tt.setup(usr, imp)
			}

			_, err := svc.UnbanUser(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			imp.AssertExpectations(t)
			usr.AssertExpectations(t)
		})
	}
}
