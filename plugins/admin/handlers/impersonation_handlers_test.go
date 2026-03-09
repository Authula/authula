package handlers_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	adminconstants "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	adminhandlers "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/handlers"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestGetAllImpersonationsHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		impRepo.On("GetAllImpersonations", mock.Anything).Return(([]types.Impersonation)(nil), errors.New("internal error")).Once()
		handler := adminhandlers.NewGetAllImpersonationsHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/impersonations", nil)
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusInternalServerError, "internal error")
		impRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		impRepo.On("GetAllImpersonations", mock.Anything).Return([]types.Impersonation{{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(time.Minute)}}, nil).Once()
		handler := adminhandlers.NewGetAllImpersonationsHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/impersonations", nil)
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[[]types.Impersonation](t, reqCtx)
		if payload == nil || len(payload) != 1 {
			t.Fatalf("expected 1 impersonation, got %v", payload)
		}
		impRepo.AssertExpectations(t)
	})
}

func TestGetImpersonationByIDHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		impRepo.On("GetImpersonationByID", mock.Anything, "imp-1").Return((*types.Impersonation)(nil), adminconstants.ErrNotFound).Once()
		handler := adminhandlers.NewGetImpersonationByIDHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/impersonations/imp-1", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		impRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		impRepo.On("GetImpersonationByID", mock.Anything, "imp-1").Return(&types.Impersonation{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(5 * time.Minute)}, nil).Once()
		handler := adminhandlers.NewGetImpersonationByIDHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/impersonations/imp-1", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.GetImpersonationByIDResponse](t, reqCtx)
		if payload.Impersonation == nil {
			t.Fatalf("expected impersonation, got %v", payload)
		}
		impRepo.AssertExpectations(t)
	})
}

func TestStartImpersonationHandler_Unauthorized(t *testing.T) {
	t.Parallel()

	useCase, _, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
	handler := adminhandlers.NewStartImpersonationHandler(useCase)

	req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations", internaltests.MarshalToJSON(t, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}))

	handler.Handler()(w, req)

	internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnauthorized, "Unauthorized")
}

func TestStartImpersonationHandler_InvalidJSON(t *testing.T) {
	t.Parallel()

	useCase, _, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
	handler := adminhandlers.NewStartImpersonationHandler(useCase)

	req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations", []byte("{invalid"))
	userID := "actor-1"
	reqCtx.UserID = &userID

	handler.Handler()(w, req)

	internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
}

func TestStartImpersonationHandler_UseCaseError(t *testing.T) {
	t.Parallel()

	useCase, impRepo, _, _, tokenSvc := admintests.NewImpersonationUseCaseFixture(t)
	impRepo.On("UserExists", mock.Anything, "actor-1").Return(true, nil).Once()
	impRepo.On("UserExists", mock.Anything, "target-1").Return(true, nil).Once()
	tokenSvc.On("Generate").Return("", adminconstants.ErrForbidden).Once()
	handler := adminhandlers.NewStartImpersonationHandler(useCase)

	req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations", internaltests.MarshalToJSON(t, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}))
	userID := "actor-1"
	reqCtx.UserID = &userID

	handler.Handler()(w, req)

	internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
	impRepo.AssertExpectations(t)
	tokenSvc.AssertExpectations(t)
}

func TestStartImpersonationHandler_SuccessSetsContextValues(t *testing.T) {
	t.Parallel()

	sessionID := "session-2"
	sessionToken := "session-token"
	actorSessionID := "session-1"
	ipAddress := "127.0.0.1"
	userAgent := "user-agent"
	useCase, impRepo, sessionStateRepo, sessionSvc, tokenSvc := admintests.NewImpersonationUseCaseFixture(t)
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
	).Return(&models.Session{ID: sessionID, IPAddress: internaltests.PtrString(ipAddress), UserAgent: internaltests.PtrString(userAgent)}, nil).Once()
	impRepo.On("CreateImpersonation", mock.Anything, mock.AnythingOfType("*types.Impersonation")).Return(nil).Once()
	sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
	handler := adminhandlers.NewStartImpersonationHandler(useCase)

	req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations", internaltests.MarshalToJSON(t, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}))
	req.Header.Set("User-Agent", userAgent)
	actorID := "actor-1"
	reqCtx.UserID = &actorID
	reqCtx.Values[models.ContextSessionID.String()] = actorSessionID

	handler.Handler()(w, req)

	if reqCtx.ResponseStatus != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, reqCtx.ResponseStatus)
	}
	if reqCtx.UserID == nil || *reqCtx.UserID != "target-1" {
		t.Fatalf("expected user id to be target-1, got %v", reqCtx.UserID)
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
	impRepo.AssertExpectations(t)
	sessionStateRepo.AssertExpectations(t)
	sessionSvc.AssertExpectations(t)
	tokenSvc.AssertExpectations(t)
}

func TestStopImpersonationHandler(t *testing.T) {
	t.Parallel()

	t.Run("unauthorized when no user ID", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		handler := adminhandlers.NewStopImpersonationHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations/imp-1/stop", nil)
		req.SetPathValue("impersonation_id", "imp-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnauthorized, "Unauthorized")
	})

	t.Run("unauthorized when no session ID", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		handler := adminhandlers.NewStopImpersonationHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations/imp-1/stop", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		actorID := "actor-1"
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnauthorized, "Unauthorized")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, sessionStateRepo, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		actorID := "actor-1"
		sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1", ImpersonatorUserID: &actorID}, nil).Once()
		impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp-1").Return((*types.Impersonation)(nil), nil).Once()
		handler := adminhandlers.NewStopImpersonationHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations/imp-1/stop", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		reqCtx.UserID = &actorID
		reqCtx.Values[models.ContextSessionID.String()] = "session-1"

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		impRepo.AssertExpectations(t)
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		actorID := "actor-1"

		useCase, impRepo, sessionStateRepo, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1", ImpersonatorUserID: &actorID}, nil).Once()
		impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp-1").Return(&types.Impersonation{ID: "imp-1", ActorUserID: "actor-1"}, nil).Once()
		impRepo.On("EndImpersonation", mock.Anything, "imp-1", mock.AnythingOfType("*string")).Return(nil).Once()
		handler := adminhandlers.NewStopImpersonationHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/impersonations/imp-1/stop", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		reqCtx.UserID = &actorID
		reqCtx.Values[models.ContextSessionID.String()] = "session-1"

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.StopImpersonationResponse](t, reqCtx)
		if payload.Message != "Impersonation stopped" {
			t.Fatalf("expected success message, got %v", payload.Message)
		}
		impRepo.AssertExpectations(t)
		sessionStateRepo.AssertExpectations(t)
	})
}
