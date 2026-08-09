package handlers_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	adminhandlers "github.com/Authula/authula/plugins/admin/handlers"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
)

func testGlobalConfig() *models.Config {
	return &models.Config{
		Session: models.SessionConfig{
			CookieName: "authula.session_token",
			HttpOnly:   true,
			Secure:     false,
			SameSite:   "lax",
		},
	}
}

func TestGetAllImpersonationsHandler(t *testing.T) {
	t.Parallel()

	someErr := errors.New("some error")

	tests := []struct {
		name            string
		setup           func(impRepo *admintests.MockImpersonationRepository)
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "error",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetAllImpersonations", mock.Anything).Return(([]types.Impersonation)(nil), someErr).Once()
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "some error",
		},
		{
			name: "success",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				now := time.Now().UTC()
				impRepo.On("GetAllImpersonations", mock.Anything).Return([]types.Impersonation{{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(time.Minute)}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
			tt.setup(impRepo)
			handler := adminhandlers.NewGetAllImpersonationsHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/impersonations", nil, nil)
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, reqCtx.ResponseStatus)
			}
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			impRepo.AssertExpectations(t)
		})
	}
}

func TestGetImpersonationByIDHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(impRepo *admintests.MockImpersonationRepository)
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "error",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("GetImpersonationByID", mock.Anything, "imp-1").Return((*types.Impersonation)(nil), coreerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "success",
			setup: func(impRepo *admintests.MockImpersonationRepository) {
				now := time.Now().UTC()
				impRepo.On("GetImpersonationByID", mock.Anything, "imp-1").Return(&types.Impersonation{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(5 * time.Minute)}, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
			tt.setup(impRepo)
			handler := adminhandlers.NewGetImpersonationByIDHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/impersonations/imp-1", nil, nil)
			req.SetPathValue("impersonation_id", "imp-1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, reqCtx.ResponseStatus)
			}
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			impRepo.AssertExpectations(t)
		})
	}
}

func TestStartImpersonationHandler(t *testing.T) {
	t.Parallel()

	sessionID := "session-2"
	sessionToken := "session-token"
	actorSessionID := "session-1"
	ipAddress := "127.0.0.1"
	userAgent := "user-agent"

	tests := []struct {
		name            string
		body            []byte
		prepare         func(t *testing.T, req *http.Request, reqCtx *models.RequestContext)
		setup           func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name:            "unauthorized when no actor",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name: "invalid json",
			body: []byte("{invalid"),
			prepare: func(t *testing.T, req *http.Request, reqCtx *models.RequestContext) {
				reqCtx.Actor = &models.Actor{ID: "actor-1", Type: models.ActorUser}
			},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "use case error",
			prepare: func(t *testing.T, req *http.Request, reqCtx *models.RequestContext) {
				reqCtx.Actor = &models.Actor{ID: "actor-1", Type: models.ActorUser}
			},
			setup: func(impRepo *admintests.MockImpersonationRepository, _ *admintests.MockSessionStateRepository, _ *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor-1").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target-1").Return(true, nil).Once()
				tokenSvc.On("Generate").Return("", coreerrors.ErrForbidden).Once()
			},
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name: "success sets context values",
			prepare: func(t *testing.T, req *http.Request, reqCtx *models.RequestContext) {
				req.Header.Set("User-Agent", userAgent)
				reqCtx.Actor = &models.Actor{ID: "actor-1", Type: models.ActorUser}
				reqCtx.Values[models.ContextSessionID.String()] = actorSessionID
			},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) {
				impRepo.On("UserExists", mock.Anything, "actor-1").Return(true, nil).Once()
				impRepo.On("UserExists", mock.Anything, "target-1").Return(true, nil).Once()
				tokenSvc.On("Generate").Return(sessionToken, nil).Once()
				tokenSvc.On("Hash", sessionToken).Return("hashed-token").Once()
				sessionSvc.On("Create",
					mock.Anything,
					"target-1",
					"hashed-token",
					mock.MatchedBy(func(ip *string) bool { return ip != nil && *ip == ipAddress }),
					mock.MatchedBy(func(ua *string) bool { return ua != nil && *ua == userAgent }),
					15*time.Minute,
				).Return(&models.Session{ID: sessionID, IPAddress: new(ipAddress), UserAgent: new(userAgent)}, nil).Once()
				impRepo.On("CreateImpersonation", mock.Anything, mock.AnythingOfType("*types.Impersonation")).Return(nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
			},
			expectedStatus: http.StatusCreated,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				if reqCtx.Actor == nil || reqCtx.Actor.ID != "target-1" {
					t.Fatalf("expected user id to be target-1, got %v", reqCtx.Actor.ID)
				}
				if reqCtx.Values[models.ContextSessionID.String()] != sessionID {
					t.Fatalf("expected session id to be updated, got %v", reqCtx.Values[models.ContextSessionID.String()])
				}
				if reqCtx.Values[models.ContextSessionToken.String()] != sessionToken {
					t.Fatalf("expected session token, got %v", reqCtx.Values[models.ContextSessionToken.String()])
				}
				if reqCtx.Values[models.ContextAuthSuccess.String()] != true {
					t.Fatal("expected auth success to be true")
				}
				payload := internaltests.DecodeResponseJSON[types.StartImpersonationResponse](t, reqCtx)
				if payload.Impersonation == nil {
					t.Fatal("expected impersonation, got nil")
				}
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
			handler := adminhandlers.NewStartImpersonationHandler(useCase, testGlobalConfig())

			body := tt.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"})
			}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations", body, nil)
			if tt.prepare != nil {
				tt.prepare(t, req, reqCtx)
			}

			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, reqCtx.ResponseStatus)
			}
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			if tt.assertResponse != nil {
				tt.assertResponse(t, reqCtx)
			}

			impRepo.AssertExpectations(t)
			sessionStateRepo.AssertExpectations(t)
			sessionSvc.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}

func TestStopImpersonationHandler(t *testing.T) {
	t.Parallel()

	originalSessionToken := "orig-session-token"

	tests := []struct {
		name            string
		prepare         func(t *testing.T, req *http.Request, reqCtx *models.RequestContext)
		setup           func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService)
		expectedStatus  int
		expectedMessage string
	}{
		{
			name:            "unauthorized when no user ID",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name: "unauthorized when no session ID",
			prepare: func(t *testing.T, req *http.Request, reqCtx *models.RequestContext) {
				reqCtx.Actor = &models.Actor{ID: "actor-1", Type: models.ActorUser}
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name: "no original cookie",
			prepare: func(t *testing.T, req *http.Request, reqCtx *models.RequestContext) {
				reqCtx.Actor = &models.Actor{ID: "actor-1", Type: models.ActorUser}
				reqCtx.Values[models.ContextSessionID.String()] = "session-1"
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "no original session found",
		},
		{
			name: "success",
			prepare: func(t *testing.T, req *http.Request, reqCtx *models.RequestContext) {
				reqCtx.Actor = &models.Actor{ID: "actor-1", Type: models.ActorUser}
				reqCtx.Values[models.ContextSessionID.String()] = "session-1"
				cn := testGlobalConfig().Session.CookieName
				req.AddCookie(&http.Cookie{Name: cn + adminconstants.OriginalSessionCookieSuffix, Value: originalSessionToken})
			},
			setup: func(impRepo *admintests.MockImpersonationRepository, sessionStateRepo *admintests.MockSessionStateRepository, sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) {
				actorID := "actor-1"
				origSessionID := "orig-session"
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1", ImpersonatorUserID: &actorID, ImpersonatorSessionID: &origSessionID}, nil).Once()
				impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp-1").Return(&types.Impersonation{ID: "imp-1", ActorUserID: "actor-1", ImpersonationSessionID: new("session-1")}, nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
				sessionSvc.On("Delete", mock.Anything, "session-1").Return(nil).Once()
				impRepo.On("EndImpersonation", mock.Anything, "imp-1", mock.AnythingOfType("*string")).Return(nil).Once()
				tokenSvc.On("Hash", originalSessionToken).Return("hashed-original").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed-original").Return(&models.Session{ID: origSessionID, UserID: actorID}, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, impRepo, sessionStateRepo, sessionSvc, tokenSvc := admintests.NewImpersonationUseCaseFixture(t)
			if tt.setup != nil {
				tt.setup(impRepo, sessionStateRepo, sessionSvc, tokenSvc)
			}
			handler := adminhandlers.NewStopImpersonationHandler(useCase, testGlobalConfig())

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations/imp-1/stop", nil, nil)
			req.SetPathValue("impersonation_id", "imp-1")
			if tt.prepare != nil {
				tt.prepare(t, req, reqCtx)
			}

			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, reqCtx.ResponseStatus)
			}
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}

			impRepo.AssertExpectations(t)
			sessionStateRepo.AssertExpectations(t)
			sessionSvc.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}
