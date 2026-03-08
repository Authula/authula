package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	adminservices "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func newStateServiceFixture() (*adminservices.StateService, *admintests.MockUserStateRepository, *admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository) {
	return admintests.NewStateServiceFixture()
}
func TestStateService_GetUserState(t *testing.T) {
	t.Parallel()

	svc, usr, _, _ := newStateServiceFixture()
	ctx := context.Background()

	usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()

	state, err := svc.GetUserState(ctx, "u1")
	assert.NoError(t, err)
	assert.Equal(t, "u1", state.UserID)
	usr.AssertExpectations(t)
}

func TestStateService_UpsertUserState(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	tests := []struct {
		name       string
		userExists bool
		hasRepoErr bool
		request    admintypes.UpsertUserStateRequest
		actor      *string
		expectCall func(*admintypes.AdminUserState) bool
		wantErr    error
	}{
		{
			name:       "user not found",
			userExists: false,
			wantErr:    constants.ErrNotFound,
		},
		{
			name:       "ban with details",
			userExists: true,
			request: admintypes.UpsertUserStateRequest{
				IsBanned:     true,
				BannedUntil:  &now,
				BannedReason: admintests.PtrString(t, "reason"),
			},
			actor: admintests.PtrString(t, "actor"),
			expectCall: func(s *admintypes.AdminUserState) bool {
				return s.IsBanned &&
					*s.BannedByUserID == "actor" &&
					s.BannedReason != nil && *s.BannedReason == "reason" &&
					s.BannedUntil.Equal(now)
			},
		},
		{
			name:       "unban",
			userExists: true,
			request: admintypes.UpsertUserStateRequest{
				IsBanned: false,
			},
			expectCall: func(s *admintypes.AdminUserState) bool {
				return !s.IsBanned
			},
		},
		{
			name:       "repo error",
			userExists: true,
			hasRepoErr: true,
			request: admintypes.UpsertUserStateRequest{
				IsBanned: false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, usr, _, imp := newStateServiceFixture()
			ctx := context.Background()

			imp.On("UserExists", mock.Anything, "u1").Return(tc.userExists, nil).Once()
			if tc.userExists {
				// always prepare an Upsert expectation when user exists
				if tc.hasRepoErr {
					usr.On("Upsert", mock.Anything, mock.Anything).Return(errors.New("boom")).Once()
				} else if tc.expectCall != nil {
					usr.On("Upsert", mock.Anything, mock.MatchedBy(tc.expectCall)).Return(nil).Once()
				} else {
					usr.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				}
				if !tc.hasRepoErr {
					usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
				}
			}

			_, err := svc.UpsertUserState(ctx, "u1", tc.request, tc.actor)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else if tc.hasRepoErr {
				assert.Error(t, err)
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

	svc, usr, _, _ := newStateServiceFixture()
	ctx := context.Background()

	usr.On("Delete", mock.Anything, "u1").Return(nil).Once()
	assert.NoError(t, svc.DeleteUserState(ctx, "u1"))
	usr.AssertExpectations(t)
}

func TestStateService_GetBannedUserStates(t *testing.T) {
	t.Parallel()

	svc, usr, _, _ := newStateServiceFixture()
	ctx := context.Background()

	usr.On("GetBanned", mock.Anything).Return([]admintypes.AdminUserState{{UserID: "u1", IsBanned: true}}, nil).Once()
	list, err := svc.GetBannedUserStates(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	usr.AssertExpectations(t)
}

func TestStateService_GetSessionState(t *testing.T) {
	t.Parallel()

	svc, _, sess, _ := newStateServiceFixture()
	ctx := context.Background()

	sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
	res, err := svc.GetSessionState(ctx, "s1")
	assert.NoError(t, err)
	assert.Equal(t, "s1", res.SessionID)
	sess.AssertExpectations(t)
}

func TestStateService_UpsertSessionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exists     bool
		hasErr     bool
		request    admintypes.UpsertSessionStateRequest
		actor      *string
		expectCall func(*admintypes.AdminSessionState) bool
		wantErr    error
	}{
		{
			name:    "session not found",
			exists:  false,
			wantErr: constants.ErrNotFound,
		},
		{
			name:    "revoke",
			exists:  true,
			request: admintypes.UpsertSessionStateRequest{Revoke: true, RevokedReason: admintests.PtrString(t, "r")},
			actor:   admintests.PtrString(t, "actor"),
			expectCall: func(s *admintypes.AdminSessionState) bool {
				return s.RevokedAt != nil && s.RevokedByUserID != nil && *s.RevokedByUserID == "actor"
			},
		},
		{
			name:    "repo failure",
			exists:  true,
			hasErr:  true,
			request: admintypes.UpsertSessionStateRequest{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, _, sess, _ := newStateServiceFixture()
			ctx := context.Background()

			sess.On("SessionExists", mock.Anything, "s1").Return(tc.exists, nil).Once()
			if tc.exists {
				if tc.hasErr {
					sess.On("Upsert", mock.Anything, mock.Anything).Return(errors.New("fail")).Once()
				} else if tc.expectCall != nil {
					sess.On("Upsert", mock.Anything, mock.MatchedBy(tc.expectCall)).Return(nil).Once()
				} else {
					sess.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				}
				if !tc.hasErr {
					sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()
				}
			}

			_, err := svc.UpsertSessionState(ctx, "s1", tc.request, tc.actor)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			sess.AssertExpectations(t)
		})
	}
}

func TestStateService_DeleteSessionState(t *testing.T) {
	t.Parallel()

	svc, _, sess, _ := newStateServiceFixture()
	ctx := context.Background()

	sess.On("Delete", mock.Anything, "s1").Return(nil).Once()
	assert.NoError(t, svc.DeleteSessionState(ctx, "s1"))
	sess.AssertExpectations(t)
}

func TestStateService_GetUserAdminSessions(t *testing.T) {
	t.Parallel()

	svc, _, sess, imp := newStateServiceFixture()
	ctx := context.Background()

	imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	sess.On("GetByUserID", mock.Anything, "u1").Return([]admintypes.AdminUserSession{}, nil).Once()

	_, err := svc.GetUserAdminSessions(ctx, "u1")
	assert.NoError(t, err)
	imp.AssertExpectations(t)
	sess.AssertExpectations(t)
}

func TestStateService_RevokeSessionAndRevokedList(t *testing.T) {
	t.Parallel()

	svc, _, sess, _ := newStateServiceFixture()
	ctx := context.Background()

	sess.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
	sess.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	sess.On("GetBySessionID", mock.Anything, "s1").Return(&admintypes.AdminSessionState{SessionID: "s1"}, nil).Once()

	_, err := svc.RevokeSession(ctx, "s1", admintests.PtrString(t, "reason"), admintests.PtrString(t, "actor"))
	assert.NoError(t, err)

	sess.On("GetRevoked", mock.Anything).Return([]admintypes.AdminSessionState{{SessionID: "s1"}}, nil).Once()
	list, err := svc.GetRevokedSessionStates(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestStateService_BanAndUnbanUser(t *testing.T) {
	t.Parallel()

	svc, usr, _, imp := newStateServiceFixture()
	ctx := context.Background()

	imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	usr.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
	_, err := svc.BanUser(ctx, "u1", admintypes.BanUserRequest{Reason: admintests.PtrString(t, "r")}, admintests.PtrString(t, "actor"))
	assert.NoError(t, err)

	imp.AssertExpectations(t)
	usr.AssertExpectations(t)

	imp.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	usr.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	usr.On("GetByUserID", mock.Anything, "u1").Return(&admintypes.AdminUserState{UserID: "u1"}, nil).Once()
	_, err = svc.UnbanUser(ctx, "u1")
	assert.NoError(t, err)
}
