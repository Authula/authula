package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func TestImpersonationUseCase_GetAllImpersonations(t *testing.T) {
	t.Parallel()

	someErr := errors.New("some error")
	now := time.Now().UTC()
	expected := []admintypes.Impersonation{{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(time.Minute)}}

	tests := []struct {
		name    string
		setup   func(impRepo *admintests.MockImpersonationRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetAllImpersonations", mock.Anything).Return(expected, nil).Once()
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

			useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
			tt.setup(impRepo)

			list, err := useCase.GetAllImpersonations(context.Background(), internaltests.TestActor())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, list)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expected, list)
			}
			impRepo.AssertExpectations(t)
		})
	}
}

func TestImpersonationUseCase_GetImpersonationByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		impersonationID string
		setup           func(impRepo *admintests.MockImpersonationRepository)
		wantErr         error
	}{
		{
			name:            "bad request when empty id",
			impersonationID: "   ",
			wantErr:         coreerrors.ErrBadRequest,
		},
		{
			name:            "forwards trimmed id",
			impersonationID: " imp-1 ",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetImpersonationByID", mock.Anything, "imp-1").Return(&admintypes.Impersonation{ID: "imp-1"}, nil).Once()
			},
		},
		{
			name:            "not found propagates",
			impersonationID: "imp-2",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetImpersonationByID", mock.Anything, "imp-2").Return((*admintypes.Impersonation)(nil), coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
			if tt.setup != nil {
				tt.setup(impRepo)
			}

			res, err := useCase.GetImpersonationByID(context.Background(), internaltests.TestActor(), tt.impersonationID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "imp-1", res.ID)
			}
			impRepo.AssertExpectations(t)
		})
	}
}

func TestImpersonationUseCase_StartImpersonation(t *testing.T) {
	t.Parallel()

	tokenErr := errors.New("fail")

	tests := []struct {
		name               string
		actor              *models.Actor
		req                admintypes.StartImpersonationRequest
		impersonatorScopes []string
		setup              func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService)
		assertResult       func(t *testing.T, res *admintypes.StartImpersonationResult)
		wantErr            error
	}{
		{
			name:    "bad request when actor empty",
			actor:   &models.Actor{ID: ""},
			req:     admintypes.StartImpersonationRequest{TargetUserID: "t", Reason: "r"},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:               "happy path returns result",
			actor:              &models.Actor{ID: "actor-1"},
			req:                admintypes.StartImpersonationRequest{TargetUserID: "target-1", Reason: "reason"},
			impersonatorScopes: []string{"scope1", "scope2"},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) {
				ipAddress := new("127.0.0.1")
				userAgent := new("user-agent")
				impRepo.On("UserExists", mock.Anything, "actor-1").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target-1").Return(true, nil).Once()
				tokenSvc.On("Generate").Return("tok", nil).Once()
				tokenSvc.On("Hash", "tok").Return("hash").Once()
				sessionSvc.On("Create", mock.Anything, "target-1", "hash", ipAddress, userAgent, mock.Anything).Return(&models.Session{ID: "sess"}, nil).Once()
				impRepo.On("CreateImpersonation", mock.Anything, mock.AnythingOfType("*types.Impersonation")).Return(nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
			},
			assertResult: func(t *testing.T, res *admintypes.StartImpersonationResult) {
				assert.Equal(t, "sess", *res.SessionID)
				assert.Equal(t, "tok", *res.SessionToken)
				assert.Equal(t, "actor-1", res.ImpersonatorUserID)
				assert.Equal(t, []string{"scope1", "scope2"}, res.ImpersonatorScopes)
				assert.Equal(t, "target-1", res.TargetUserID)
				assert.Equal(t, "orig-token", res.OriginalCookieToken)
			},
		},
		{
			name:  "error from token service propagates",
			actor: &models.Actor{ID: "actor-1"},
			req:   admintypes.StartImpersonationRequest{TargetUserID: "target-1", Reason: "reason"},
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor-1").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target-1").Return(true, nil).Once()
				tokenSvc.On("Generate").Return("", tokenErr).Once()
			},
			wantErr: tokenErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, impRepo, sessionStateRepo, sessionSvc, tokenSvc := admintests.NewImpersonationUseCaseFixture(t)
			if tt.setup != nil {
				tt.setup(impRepo, sessionStateRepo, sessionSvc, tokenSvc)
			}

			res, err := useCase.StartImpersonation(context.Background(), tt.actor, nil, new("127.0.0.1"), new("user-agent"), tt.req, tt.impersonatorScopes, "orig-token", 100)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				if tt.assertResult != nil {
					tt.assertResult(t, res)
				}
			}

			impRepo.AssertExpectations(t)
			sessionStateRepo.AssertExpectations(t)
			sessionSvc.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}

func TestImpersonationUseCase_StopImpersonation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		actor               *models.Actor
		originalCookieValue string
		setup               func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService)
		wantErr             error
	}{
		{
			name:  "error when session state lookup fails",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser},
			setup: func(_ *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return((*admintypes.AdminSessionState)(nil), coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:  "unauthorized when session state has no impersonator",
			actor: &models.Actor{ID: "actor-1", Type: models.ActorUser},
			setup: func(_ *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, _ *internaltests.MockTokenService) {
				sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return(&admintypes.AdminSessionState{SessionID: "impersonated-session-1"}, nil).Once()
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:                "not found when no active impersonation for resolved actor",
			actor:               &models.Actor{ID: "actor-1", Type: models.ActorUser},
			originalCookieValue: "orig-token",
			setup: func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) {
				sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return(&admintypes.AdminSessionState{SessionID: "impersonated-session-1", ImpersonatorUserID: new("actor-1"), ImpersonatorSessionID: new("orig-sess")}, nil).Once()
				tokenSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed-orig").Return(&models.Session{ID: "orig-sess", UserID: "actor-1"}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return((*admintypes.Impersonation)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:                "session binding mismatch",
			actor:               &models.Actor{ID: "actor-1", Type: models.ActorUser},
			originalCookieValue: "orig-token",
			setup: func(_ *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) {
				sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return(&admintypes.AdminSessionState{SessionID: "impersonated-session-1", ImpersonatorUserID: new("actor-1"), ImpersonatorSessionID: new("other-sess")}, nil).Once()
				tokenSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed-orig").Return(&models.Session{ID: "orig-sess", UserID: "actor-1"}, nil).Once()
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:                "successful stop uses impersonator from session state",
			actor:               &models.Actor{ID: "actor-1", Type: models.ActorUser},
			originalCookieValue: "orig-token",
			setup: func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) {
				sessionID := "impersonated-session-1"
				imp := &admintypes.Impersonation{ID: "imp-1", ActorUserID: "actor-1", ImpersonationSessionID: &sessionID}
				sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return(&admintypes.AdminSessionState{SessionID: "impersonated-session-1", ImpersonatorUserID: new("actor-1"), ImpersonatorSessionID: new("orig-sess")}, nil).Once()
				impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return(imp, nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
				sessionSvc.On("Delete", mock.Anything, "impersonated-session-1").Return(nil).Once()
				impRepo.On("EndImpersonation", mock.Anything, "imp-1", mock.AnythingOfType("*string")).Return(nil).Once()
				tokenSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed-orig").Return(&models.Session{ID: "orig-sess", UserID: "actor-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, impRepo, sessionStateRepo, sessionSvc, tokenSvc := admintests.NewImpersonationUseCaseFixture(t)
			if tt.setup != nil {
				tt.setup(impRepo, sessionStateRepo, sessionSvc, tokenSvc)
			}

			res, err := useCase.StopImpersonation(context.Background(), tt.actor, "target-1", "impersonated-session-1", tt.originalCookieValue, admintypes.StopImpersonationRequest{})
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, tt.originalCookieValue, res.OriginalSessionToken)
			}

			impRepo.AssertExpectations(t)
			sessionStateRepo.AssertExpectations(t)
			sessionSvc.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}
