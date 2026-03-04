package handlers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestStartImpersonationHandler_Unauthorized(t *testing.T) {
	useCase := &mockImpersonationUseCase{}
	handler := NewStartImpersonationHandler(useCase)

	req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations", mustJSON(t, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}))

	handler.Handler()(w, req)

	assertErrorMessage(t, reqCtx, http.StatusUnauthorized, "Unauthorized")
	useCase.AssertExpectations(t)
}

func TestStartImpersonationHandler_InvalidJSON(t *testing.T) {
	useCase := &mockImpersonationUseCase{}
	handler := NewStartImpersonationHandler(useCase)

	req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations", []byte("{invalid"))
	userID := "actor-1"
	reqCtx.UserID = &userID

	handler.Handler()(w, req)

	assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	useCase.AssertExpectations(t)
}

func TestStartImpersonationHandler_UseCaseError(t *testing.T) {
	useCase := &mockImpersonationUseCase{}
	useCase.On("StartImpersonation", mock.Anything, "actor-1", (*string)(nil), types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}).
		Return(nil, errors.New("forbidden")).Once()
	handler := NewStartImpersonationHandler(useCase)

	req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations", mustJSON(t, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}))
	userID := "actor-1"
	reqCtx.UserID = &userID

	handler.Handler()(w, req)

	assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
	useCase.AssertExpectations(t)
}

func TestStartImpersonationHandler_NilResult(t *testing.T) {
	useCase := &mockImpersonationUseCase{}
	useCase.On("StartImpersonation", mock.Anything, "actor-1", (*string)(nil), types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}).
		Return(nil, nil).Once()
	handler := NewStartImpersonationHandler(useCase)

	req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations", mustJSON(t, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}))
	userID := "actor-1"
	reqCtx.UserID = &userID

	handler.Handler()(w, req)

	assertErrorMessage(t, reqCtx, http.StatusInternalServerError, "failed to start impersonation")
	useCase.AssertExpectations(t)
}

func TestStartImpersonationHandler_SuccessSetsContextValues(t *testing.T) {
	now := time.Now().UTC()
	sessionID := "session-2"
	sessionToken := "session-token"
	actorSessionID := "session-1"
	useCase := &mockImpersonationUseCase{}
	useCase.On("StartImpersonation", mock.Anything, "actor-1", &actorSessionID, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}).
		Return(&types.StartImpersonationResult{
			Impersonation: &types.Impersonation{ID: "imp-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(10 * time.Minute)},
			SessionID:     &sessionID,
			SessionToken:  &sessionToken,
		}, nil).Once()
	handler := NewStartImpersonationHandler(useCase)

	req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations", mustJSON(t, types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "support"}))
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

	payload := decodeResponseJSON(t, reqCtx)
	if payload["message"] != "impersonation started" {
		t.Fatalf("expected success message, got %v", payload["message"])
	}
	useCase.AssertExpectations(t)
}

func TestEndImpersonationByIDHandler(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		useCase := &mockImpersonationUseCase{}
		handler := NewEndImpersonationByIDHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations/imp-1/stop", nil)
		req.SetPathValue("impersonation_id", "imp-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnauthorized, "Unauthorized")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockImpersonationUseCase{}
		useCase.On("StopImpersonation", mock.Anything, "actor-1", types.StopImpersonationRequest{ImpersonationID: stringPtr("imp-1")}).
			Return(errors.New("not found")).Once()
		handler := NewEndImpersonationByIDHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations/imp-1/stop", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		actorID := "actor-1"
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockImpersonationUseCase{}
		useCase.On("StopImpersonation", mock.Anything, "actor-1", types.StopImpersonationRequest{ImpersonationID: stringPtr("imp-1")}).Return(nil).Once()
		handler := NewEndImpersonationByIDHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations/imp-1/stop", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		actorID := "actor-1"
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "impersonation ended" {
			t.Fatalf("expected success message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetAllImpersonationsHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockImpersonationUseCase{}
		useCase.On("GetAllImpersonations", mock.Anything).Return(nil, errors.New("internal error")).Once()
		handler := NewGetAllImpersonationsHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/impersonations", nil)
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusInternalServerError, "internal error")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		now := time.Now().UTC()
		useCase := &mockImpersonationUseCase{}
		useCase.On("GetAllImpersonations", mock.Anything).
			Return([]types.Impersonation{{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(time.Minute)}}, nil).Once()
		handler := NewGetAllImpersonationsHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/impersonations", nil)
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetImpersonationByIDHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockImpersonationUseCase{}
		useCase.On("GetImpersonationByID", mock.Anything, "imp-1").Return(nil, errors.New("not found")).Once()
		handler := NewGetImpersonationByIDHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/impersonations/imp-1", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		now := time.Now().UTC()
		useCase := &mockImpersonationUseCase{}
		useCase.On("GetImpersonationByID", mock.Anything, "imp-1").
			Return(&types.Impersonation{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(5 * time.Minute)}, nil).Once()
		handler := NewGetImpersonationByIDHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/impersonations/imp-1", nil)
		req.SetPathValue("impersonation_id", "imp-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func stringPtr(v string) *string {
	return &v
}
