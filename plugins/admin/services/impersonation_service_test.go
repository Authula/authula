package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func newImpersonationServiceFixture() (*adminservices.ImpersonationService, *admintests.MockImpersonationRepository, *admintests.MockSessionStateRepository, *internaltests.MockSessionService, *internaltests.MockTokenService) {
	svc, impRepo, sessRepo, sessSvc, tokSvc := admintests.NewImpersonationServiceFixture()
	return svc, impRepo, sessRepo, sessSvc, tokSvc
}

func TestImpersonationService_New_DefaultsMaxExpiry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		sessionExpiresIn time.Duration
		maxExpiresIn     time.Duration
		wantExpiry       time.Duration
	}{
		{
			name:             "both zero fall back to default",
			sessionExpiresIn: 0,
			maxExpiresIn:     0,
			wantExpiry:       15 * time.Minute,
		},
		{
			name:             "session expiry defaults to max expiry",
			sessionExpiresIn: 0,
			maxExpiresIn:     5 * time.Minute,
			wantExpiry:       5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			impRepo := &admintests.MockImpersonationRepository{}
			sessRepo := &admintests.MockSessionStateRepository{}
			svc := adminservices.NewImpersonationService(impRepo, sessRepo, nil, nil, tt.sessionExpiresIn, tt.maxExpiresIn)

			impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
			impRepo.On("CreateImpersonation", mock.Anything, mock.AnythingOfType("*types.Impersonation")).Return(nil).Once()

			before := time.Now().UTC()
			res, err := svc.StartImpersonation(
				context.Background(),
				&models.Actor{ID: "actor", Type: models.ActorUser},
				nil,
				nil,
				nil,
				admintypes.StartImpersonationRequest{TargetUserID: "target", Reason: "reason"},
				nil,
				"",
				0,
			)
			require.NoError(t, err)
			require.NotNil(t, res)

			got := res.Impersonation.ExpiresAt.Sub(before)
			assert.InDelta(t, tt.wantExpiry.Seconds(), got.Seconds(), 5)

			impRepo.AssertExpectations(t)
			sessRepo.AssertExpectations(t)
		})
	}
}

func TestImpersonationService_StartImpersonation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	generateErr := errors.New("generate failure")
	createErr := errors.New("create failure")
	impCreateErr := errors.New("impersonation create failure")
	stateErr := errors.New("state failure")
	targetReq := admintypes.StartImpersonationRequest{TargetUserID: "target", Reason: "reason"}
	const cookieMaxAge = 1

	tests := []struct {
		name                 string
		actor                string
		req                  admintypes.StartImpersonationRequest
		withSessionServices  bool
		impersonatorScopes   []string
		originalCookieValue  string
		originalCookieMaxAge int
		setup                func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService)
		wantErr              error
		assertResult         func(t *testing.T, res *admintypes.StartImpersonationResult)
	}{
		{name: "empty actor", actor: "", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, wantErr: coreerrors.ErrBadRequest},
		{name: "empty target", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "  ", Reason: "r"}, wantErr: coreerrors.ErrBadRequest},
		{name: "same user", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "a1", Reason: "r"}, wantErr: coreerrors.ErrBadRequest},
		{name: "empty reason", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "   "}, wantErr: coreerrors.ErrBadRequest},
		{name: "actor not exists", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(false, nil).Once()
		}, wantErr: coreerrors.ErrNotFound},
		{name: "target not exists", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(false, nil).Once()
		}, wantErr: coreerrors.ErrNotFound},
		{name: "expires invalid zero", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r", ExpiresInSeconds: new(0)}, setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(true, nil).Once()
		}, wantErr: coreerrors.ErrBadRequest},
		{name: "expires invalid large", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r", ExpiresInSeconds: new(999999)}, setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(true, nil).Once()
		}, wantErr: coreerrors.ErrBadRequest},
		{
			name:                 "success with session creation",
			actor:                "actor",
			req:                  admintypes.StartImpersonationRequest{TargetUserID: "target", Reason: "reason", ExpiresInSeconds: new(60)},
			withSessionServices:  true,
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
			name:  "no session services skips session creation",
			actor: "actor",
			req:   targetReq,
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
		{
			name:                "token generation fails",
			actor:               "actor",
			req:                 targetReq,
			withSessionServices: true,
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				tokSvc.On("Generate").Return("", generateErr).Once()
			},
			wantErr: generateErr,
		},
		{
			name:                "session creation fails",
			actor:               "actor",
			req:                 targetReq,
			withSessionServices: true,
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				tokSvc.On("Generate").Return("raw", nil).Once()
				tokSvc.On("Hash", "raw").Return("hashed").Once()
				sessSvc.On("Create", mock.Anything, "target", "hashed", mock.Anything, mock.Anything, mock.Anything).Return((*models.Session)(nil), createErr).Once()
			},
			wantErr: createErr,
		},
		{
			name:                "impersonation record creation fails",
			actor:               "actor",
			req:                 targetReq,
			withSessionServices: true,
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				tokSvc.On("Generate").Return("raw", nil).Once()
				tokSvc.On("Hash", "raw").Return("hashed").Once()
				sessSvc.On("Create", mock.Anything, "target", "hashed", mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "sess1"}, nil).Once()
				impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(impCreateErr).Once()
			},
			wantErr: impCreateErr,
		},
		{
			name:                "session state upsert fails",
			actor:               "actor",
			req:                 targetReq,
			withSessionServices: true,
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				tokSvc.On("Generate").Return("raw", nil).Once()
				tokSvc.On("Hash", "raw").Return("hashed").Once()
				sessSvc.On("Create", mock.Anything, "target", "hashed", mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "sess1"}, nil).Once()
				impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()
				sessRepo.On("Upsert", mock.Anything, mock.Anything).Return(stateErr).Once()
			},
			wantErr: stateErr,
		},
		{
			name:                 "caps at original cookie max age",
			actor:                "actor",
			req:                  targetReq,
			originalCookieValue:  "orig",
			originalCookieMaxAge: cookieMaxAge,
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()
			},
			assertResult: func(t *testing.T, res *admintypes.StartImpersonationResult) {
				require.Equal(t, "orig", res.OriginalCookieToken)
				require.LessOrEqual(t, res.OriginalCookieMaxAge, cookieMaxAge)
				require.GreaterOrEqual(t, res.OriginalCookieMaxAge, 0)
			},
		},
		{
			name:  "empty cookie yields no token and zero max age",
			actor: "actor",
			req:   targetReq,
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()
			},
			assertResult: func(t *testing.T, res *admintypes.StartImpersonationResult) {
				require.Empty(t, res.OriginalCookieToken)
				require.Zero(t, res.OriginalCookieMaxAge)
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
			res, err := svc.StartImpersonation(ctx, &models.Actor{ID: tt.actor}, nil, ipAddress, userAgent, tt.req, tt.impersonatorScopes, tt.originalCookieValue, tt.originalCookieMaxAge)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				if tt.assertResult != nil {
					tt.assertResult(t, res)
				}
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
	repoErr := errors.New("repo failure")
	upsertErr := errors.New("upsert failure")
	deleteErr := errors.New("delete failure")
	endErr := errors.New("end failure")
	sessionID := "sess-1"

	tests := []struct {
		name                string
		actor               *models.Actor
		originalCookieValue string
		request             admintypes.StopImpersonationRequest
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
		{
			name:  "nil session state is unauthorized",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser},
			setup: func(_ *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return((*admintypes.AdminSessionState)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:  "nil actor with claims path is unauthorized",
			actor: nil,
			setup: func(_ *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:  "claim with non-string impersonator id is unauthorized",
			actor: &models.Actor{ID: "target-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: 42}},
			setup: func(_ *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:  "active impersonation lookup fails by id",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"}},
			request: admintypes.StopImpersonationRequest{
				ImpersonationID: new(" imp-1 "),
			},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
				impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp-1").Return((*admintypes.Impersonation)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:  "active impersonation lookup by id not found",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"}},
			request: admintypes.StopImpersonationRequest{
				ImpersonationID: new("imp-1"),
			},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
				impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp-1").Return((*admintypes.Impersonation)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:  "active impersonation actor mismatch is forbidden",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"}},
			request: admintypes.StopImpersonationRequest{
				ImpersonationID: new("imp-1"),
			},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
				impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp-1").Return(&admintypes.Impersonation{ID: "imp-1", ActorUserID: "other"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "latest active lookup fails",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"}},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return((*admintypes.Impersonation)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:  "session state upsert fails during stop",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"}},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return(&admintypes.Impersonation{ID: "imp-1", ActorUserID: "actor-1", ImpersonationSessionID: &sessionID}, nil).Once()
				sessRepo.On("Upsert", mock.Anything, mock.Anything).Return(upsertErr).Once()
			},
			wantErr: upsertErr,
		},
		{
			name:  "session delete fails during stop",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"}},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return(&admintypes.Impersonation{ID: "imp-1", ActorUserID: "actor-1", ImpersonationSessionID: &sessionID}, nil).Once()
				sessRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				sessSvc.On("Delete", mock.Anything, sessionID).Return(deleteErr).Once()
			},
			wantErr: deleteErr,
		},
		{
			name:  "end impersonation fails during stop",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"}},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return(&admintypes.Impersonation{ID: "imp-1", ActorUserID: "actor-1", ImpersonationSessionID: &sessionID}, nil).Once()
				sessRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				sessSvc.On("Delete", mock.Anything, sessionID).Return(nil).Once()
				impRepo.On("EndImpersonation", mock.Anything, "imp-1", mock.AnythingOfType("*string")).Return(endErr).Once()
			},
			wantErr: endErr,
		},
		{
			name:  "success via claims without impersonation session skips cleanup",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"}},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&admintypes.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor-1")}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return(&admintypes.Impersonation{ID: "imp-1", ActorUserID: "actor-1"}, nil).Once()
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

			res, err := svc.StopImpersonation(ctx, tt.actor, "sess-1", tt.originalCookieValue, tt.request)

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
	repoErr := errors.New("repo failure")

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
			name:            "repo error",
			impersonationID: "i1",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetImpersonationByID", mock.Anything, "i1").Return((*admintypes.Impersonation)(nil), repoErr).Once()
			},
			wantErr: repoErr,
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
