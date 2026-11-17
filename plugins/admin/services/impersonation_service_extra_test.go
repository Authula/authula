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

func TestImpersonationService_StartImpersonation_SessionErrors(t *testing.T) {
	t.Parallel()

	generateErr := errors.New("generate failure")
	createErr := errors.New("create failure")
	impCreateErr := errors.New("impersonation create failure")
	stateErr := errors.New("state failure")

	tests := []struct {
		name    string
		setup   func(impRepo *admintests.MockImpersonationRepository, sessRepo *admintests.MockSessionStateRepository, sessSvc *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService)
		wantErr error
	}{
		{
			name: "token generation fails",
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, tokSvc *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
				tokSvc.On("Generate").Return("", generateErr).Once()
			},
			wantErr: generateErr,
		},
		{
			name: "session creation fails",
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
			name: "impersonation record creation fails",
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
			name: "session state upsert fails",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, impRepo, sessRepo, sessSvc, tokSvc := newImpersonationServiceFixture()
			tt.setup(impRepo, sessRepo, sessSvc, tokSvc)

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
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, res)

			impRepo.AssertExpectations(t)
			sessRepo.AssertExpectations(t)
			sessSvc.AssertExpectations(t)
			tokSvc.AssertExpectations(t)
		})
	}
}

func TestImpersonationService_StartImpersonation_OriginalCookieMaxAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		originalCookieValue  string
		originalCookieMaxAge int
		wantToken            string
		wantMaxAgeZero       bool
	}{
		{
			name:                 "caps at original cookie max age",
			originalCookieValue:  "orig",
			originalCookieMaxAge: 1,
			wantToken:            "orig",
		},
		{
			name:                "empty cookie yields no token and zero max age",
			originalCookieValue: "",
			wantMaxAgeZero:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, impRepo, _, _, _ := newImpersonationServiceFixture()
			svc := adminservices.NewImpersonationService(impRepo, &admintests.MockSessionStateRepository{}, nil, nil, time.Minute, time.Minute)
			impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
			impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()

			res, err := svc.StartImpersonation(
				context.Background(),
				&models.Actor{ID: "actor", Type: models.ActorUser},
				nil,
				nil,
				nil,
				admintypes.StartImpersonationRequest{TargetUserID: "target", Reason: "reason"},
				nil,
				tt.originalCookieValue,
				tt.originalCookieMaxAge,
			)
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, tt.wantToken, res.OriginalCookieToken)
			if tt.wantMaxAgeZero {
				assert.Zero(t, res.OriginalCookieMaxAge)
			} else {
				assert.LessOrEqual(t, res.OriginalCookieMaxAge, tt.originalCookieMaxAge)
				assert.GreaterOrEqual(t, res.OriginalCookieMaxAge, 0)
			}

			impRepo.AssertExpectations(t)
		})
	}
}

func TestImpersonationService_GetImpersonationByID_RepoError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	svc, impRepo, _, _, _ := newImpersonationServiceFixture()
	impRepo.On("GetImpersonationByID", mock.Anything, "imp-1").Return((*admintypes.Impersonation)(nil), repoErr).Once()

	res, err := svc.GetImpersonationByID(context.Background(), internaltests.TestActor(), "imp-1")
	assert.ErrorIs(t, err, repoErr)
	assert.Nil(t, res)

	impRepo.AssertExpectations(t)
}

func TestImpersonationService_StopImpersonation_ExtraBranches(t *testing.T) {
	t.Parallel()

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
			tt.setup(impRepo, sessRepo, sessSvc, tokSvc)

			res, err := svc.StopImpersonation(context.Background(), tt.actor, "sess-1", tt.originalCookieValue, tt.request)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
			}

			impRepo.AssertExpectations(t)
			sessRepo.AssertExpectations(t)
			sessSvc.AssertExpectations(t)
			tokSvc.AssertExpectations(t)
		})
	}
}
