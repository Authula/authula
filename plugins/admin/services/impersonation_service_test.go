package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func newImpersonationServiceFixture() (*adminservices.ImpersonationService, *admintests.MockImpersonationRepository, *admintests.MockSessionStateRepository, *internaltests.MockSessionService, *internaltests.MockTokenService) {
	svc, impRepo, sessRepo, sessSvc, tokSvc := admintests.NewImpersonationServiceFixture()
	return svc, impRepo, sessRepo, sessSvc, tokSvc
}

func TestImpersonationService_StartImpersonation_validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name  string
		actor string
		req   admintypes.StartImpersonationRequest
		setup func(impRepo *admintests.MockImpersonationRepository)
		want  error
	}{
		{name: "empty actor", actor: "", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, want: coreerrors.ErrBadRequest},
		{name: "empty target", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "  ", Reason: "r"}, want: coreerrors.ErrBadRequest},
		{name: "same user", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "a1", Reason: "r"}, want: coreerrors.ErrBadRequest},
		{name: "empty reason", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "   "}, want: coreerrors.ErrBadRequest},
		{name: "actor not exists", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, setup: func(impRepo *admintests.MockImpersonationRepository) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(false, nil).Once()
		}, want: coreerrors.ErrNotFound},
		{name: "target not exists", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, setup: func(impRepo *admintests.MockImpersonationRepository) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(false, nil).Once()
		}, want: coreerrors.ErrNotFound},
		{name: "expires invalid zero", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r", ExpiresInSeconds: new(0)}, setup: func(impRepo *admintests.MockImpersonationRepository) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(true, nil).Once()
		}, want: coreerrors.ErrBadRequest},
		{name: "expires invalid large", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r", ExpiresInSeconds: new(999999)}, setup: func(impRepo *admintests.MockImpersonationRepository) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(true, nil).Once()
		}, want: coreerrors.ErrBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, impRepo, _, _, _ := newImpersonationServiceFixture()

			if tc.setup != nil {
				tc.setup(impRepo)
			}

			ipAddress := new("127.0.0.1")
			userAgent := new("user-agent")
			_, err := svc.StartImpersonation(ctx, &models.Actor{ID: tc.actor}, nil, ipAddress, userAgent, tc.req, nil, "", 0)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			} else {
				require.NoError(t, err)
			}
			impRepo.AssertExpectations(t)
		})
	}
}

func TestImpersonationService_StartImpersonation_success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name                 string
		withSessionServices  bool
		req                  admintypes.StartImpersonationRequest
		impersonatorScopes   []string
		originalCookieValue  string
		originalCookieMaxAge int
		setup                func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService)
		assertResult         func(t *testing.T, res *admintypes.StartImpersonationResult)
	}{
		{
			name:                 "success with session creation",
			withSessionServices:  true,
			req:                  admintypes.StartImpersonationRequest{TargetUserID: "target", Reason: "reason", ExpiresInSeconds: new(60)},
			impersonatorScopes:   []string{"scope1"},
			originalCookieValue:  "orig-token",
			originalCookieMaxAge: 100,
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				rawToken := "rawtoken"
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				tokSvc.On("Generate").Return(rawToken, nil).Once()
				tokSvc.On("Hash", rawToken).Return("hashed").Once()
				sessSvc.On("Create", mock.Anything, "target", "hashed", mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "sess1"}, nil).Once()
				impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()
				sessRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
			},
			assertResult: func(t *testing.T, res *admintypes.StartImpersonationResult) {
				require.NotNil(t, res.SessionID)
				require.Equal(t, "sess1", *res.SessionID)
				require.NotNil(t, res.SessionToken)
				require.Equal(t, "rawtoken", *res.SessionToken)
				require.Equal(t, "actor", res.ImpersonatorUserID)
				require.Equal(t, []string{"scope1"}, res.ImpersonatorScopes)
				require.Equal(t, "target", res.TargetUserID)
				require.Equal(t, "orig-token", res.OriginalCookieToken)
			},
		},
		{
			name: "no session services skips session creation",
			req:  admintypes.StartImpersonationRequest{TargetUserID: "target", Reason: "reason"},
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()
			},
			assertResult: func(t *testing.T, res *admintypes.StartImpersonationResult) {
				require.Nil(t, res.SessionID)
				require.Nil(t, res.SessionToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var svc *adminservices.ImpersonationService
			var impRepo *admintests.MockImpersonationRepository
			var sessRepo *admintests.MockSessionStateRepository
			var sessSvc *internaltests.MockSessionService
			var tokSvc *internaltests.MockTokenService

			if tt.withSessionServices {
				svc, impRepo, sessRepo, sessSvc, tokSvc = newImpersonationServiceFixture()
			} else {
				impRepo = &admintests.MockImpersonationRepository{}
				sessRepo = &admintests.MockSessionStateRepository{}
				svc = adminservices.NewImpersonationService(impRepo, sessRepo, nil, nil, time.Minute, time.Minute)
			}
			if tt.setup != nil {
				tt.setup(impRepo, sessRepo, sessSvc, tokSvc)
			}

			ipAddress := new("127.0.0.1")
			userAgent := new("user-agent")
			res, err := svc.StartImpersonation(ctx, &models.Actor{ID: "actor"}, nil, ipAddress, userAgent, tt.req, tt.impersonatorScopes, tt.originalCookieValue, tt.originalCookieMaxAge)
			require.NoError(t, err)
			require.NotNil(t, res)
			if tt.assertResult != nil {
				tt.assertResult(t, res)
			}

			impRepo.AssertExpectations(t)
			sessRepo.AssertExpectations(t)
			if sessSvc != nil {
				sessSvc.AssertExpectations(t)
			}
			if tokSvc != nil {
				tokSvc.AssertExpectations(t)
			}
		})
	}
}

func TestImpersonationService_ValidateImpersonationCookie(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService)
		wantErr error
	}{
		{
			name: "session not found returns forbidden",
			setup: func(sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				tokSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
				sessSvc.On("GetByToken", mock.Anything, "hashed-orig").Return((*models.Session)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "success returns session",
			setup: func(sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				tokSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
				sessSvc.On("GetByToken", mock.Anything, "hashed-orig").Return(&models.Session{ID: "orig-sess", UserID: "actor-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, _, _, sessSvc, tokSvc := newImpersonationServiceFixture()
			tt.setup(sessSvc, tokSvc)

			sess, err := svc.ValidateImpersonationCookie(ctx, "orig-token")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, sess)
			} else {
				require.NoError(t, err)
				require.NotNil(t, sess)
				require.Equal(t, "orig-sess", sess.ID)
			}

			tokSvc.AssertExpectations(t)
			sessSvc.AssertExpectations(t)
		})
	}
}

func TestImpersonationService_StopImpersonation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name                string
		actor               *models.Actor
		originalCookieValue string
		setup               func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService)
		wantErr             error
	}{
		{
			name:  "session state lookup error propagates",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser},
			setup: func(_ *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return((*admintypes.AdminSessionState)(nil), coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:  "unauthorized when session state has no impersonator",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser},
			setup: func(_ *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1"}, nil).Once()
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:                "unauthorized when original cookie does not match impersonator session",
			actor:               &models.Actor{ID: "actor-1", Type: models.ActorUser},
			originalCookieValue: "orig-token",
			setup: func(_ *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1"), ImpersonatorSessionID: new("other-sess")}, nil).Once()
				tokSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
				sessSvc.On("GetByToken", mock.Anything, "hashed-orig").Return(&models.Session{ID: "orig-sess", UserID: "actor-1"}, nil).Once()
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:  "unauthorized when claims do not match impersonator",
			actor: &models.Actor{ID: "target-1", Type: models.ActorUser},
			setup: func(_ *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:                "not found when no active impersonation for resolved actor",
			actor:               &models.Actor{ID: "actor-1", Type: models.ActorUser},
			originalCookieValue: "orig-token",
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1"), ImpersonatorSessionID: new("orig-sess")}, nil).Once()
				tokSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
				sessSvc.On("GetByToken", mock.Anything, "hashed-orig").Return(&models.Session{ID: "orig-sess", UserID: "actor-1"}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return((*admintypes.Impersonation)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:                "success with session cleanup",
			actor:               &models.Actor{ID: "actor-1", Type: models.ActorUser},
			originalCookieValue: "orig-token",
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				sessionID := "sess-1"
				imp := &admintypes.Impersonation{ID: "imp-1", ActorUserID: "actor-1", ImpersonationSessionID: &sessionID}
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1"), ImpersonatorSessionID: new("orig-sess")}, nil).Once()
				tokSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
				sessSvc.On("GetByToken", mock.Anything, "hashed-orig").Return(&models.Session{ID: "orig-sess", UserID: "actor-1"}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return(imp, nil).Once()
				sessRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				sessSvc.On("Delete", mock.Anything, "sess-1").Return(nil).Once()
				impRepo.On("EndImpersonation", mock.Anything, "imp-1", mock.AnythingOfType("*string")).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, impRepo, sessRepo, sessSvc, tokSvc := newImpersonationServiceFixture()
			if tt.setup != nil {
				tt.setup(impRepo, sessRepo, sessSvc, tokSvc)
			}

			res, err := svc.StopImpersonation(ctx, tt.actor, "sess-1", tt.originalCookieValue, admintypes.StopImpersonationRequest{})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tt.originalCookieValue, res.OriginalSessionToken)
			}

			impRepo.AssertExpectations(t)
			sessRepo.AssertExpectations(t)
			sessSvc.AssertExpectations(t)
			tokSvc.AssertExpectations(t)
		})
	}
}

func TestImpersonationService_GetAllImpersonations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	someErr := errors.New("some error")

	tests := []struct {
		name    string
		setup   func(impRepo *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetAllImpersonations", mock.Anything).Return([]admintypes.Impersonation{{ID: "i1"}}, nil).Once()
			},
		},
		{
			name: "repo error",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetAllImpersonations", mock.Anything).Return(nil, someErr).Once()
			},
			wantErr: someErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, impRepo, _, _, _ := newImpersonationServiceFixture()
			tt.setup(impRepo)

			res, err := svc.GetAllImpersonations(ctx, internaltests.TestActor())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Len(t, res, 1)
			}
			impRepo.AssertExpectations(t)
		})
	}
}

func TestImpersonationService_GetImpersonationByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name            string
		impersonationID string
		setup           func(impRepo *admintests.MockImpersonationRepository)
		wantErr         error
	}{
		{
			name:            "missing id",
			impersonationID: "   ",
			wantErr:         coreerrors.ErrBadRequest,
		},
		{
			name:            "not found",
			impersonationID: "i1",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetImpersonationByID", mock.Anything, "i1").Return((*admintypes.Impersonation)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:            "success",
			impersonationID: " i1 ",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetImpersonationByID", mock.Anything, "i1").Return(&admintypes.Impersonation{ID: "i1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, impRepo, _, _, _ := newImpersonationServiceFixture()
			if tt.setup != nil {
				tt.setup(impRepo)
			}

			res, err := svc.GetImpersonationByID(ctx, internaltests.TestActor(), tt.impersonationID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, "i1", res.ID)
			}
			impRepo.AssertExpectations(t)
		})
	}
}
