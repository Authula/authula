package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type impersonationUseCaseStub struct {
	startFn  func(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error)
	stopFn   func(ctx context.Context, actorUserID string, req types.StopImpersonationRequest) error
	getAllFn func(ctx context.Context) ([]types.Impersonation, error)
	getByID  func(ctx context.Context, impersonationID string) (*types.Impersonation, error)
}

func (s impersonationUseCaseStub) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	if s.startFn == nil {
		return nil, nil
	}
	return s.startFn(ctx, actorUserID, actorSessionID, req)
}

func (s impersonationUseCaseStub) StopImpersonation(ctx context.Context, actorUserID string, req types.StopImpersonationRequest) error {
	if s.stopFn == nil {
		return nil
	}
	return s.stopFn(ctx, actorUserID, req)
}

func (s impersonationUseCaseStub) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	if s.getAllFn == nil {
		return nil, nil
	}
	return s.getAllFn(ctx)
}

func (s impersonationUseCaseStub) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	if s.getByID == nil {
		return nil, nil
	}
	return s.getByID(ctx, impersonationID)
}

func TestStartImpersonationHandler_Unauthorized(t *testing.T) {
	called := false
	h := NewStartImpersonationHandler(impersonationUseCaseStub{
		startFn: func(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
			called = true
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/impersonations/start", strings.NewReader(`{"target_user_id":"u-2","reason":"support"}`))
	req, rc := withReqCtx(req)

	h.Handler().ServeHTTP(httptest.NewRecorder(), req)

	if called {
		t.Fatal("usecase must not be called for unauthorized requests")
	}
	if rc.ResponseStatus != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rc.ResponseStatus)
	}
	body := decodeJSONBody[map[string]any](t, rc)
	if body["message"] != "Unauthorized" {
		t.Fatalf("expected message Unauthorized, got %v", body["message"])
	}
	if !rc.Handled {
		t.Fatal("expected request context to be marked handled")
	}
}

func TestStartImpersonationHandler_InvalidJSON(t *testing.T) {
	h := NewStartImpersonationHandler(impersonationUseCaseStub{})

	req := httptest.NewRequest(http.MethodPost, "/admin/impersonations/start", strings.NewReader("{"))
	req, rc := withReqCtx(req)
	userID := "actor-1"
	rc.UserID = &userID

	h.Handler().ServeHTTP(httptest.NewRecorder(), req)

	if rc.ResponseStatus != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rc.ResponseStatus)
	}
	body := decodeJSONBody[map[string]any](t, rc)
	if body["message"] != "invalid request body" {
		t.Fatalf("expected invalid request body message, got %v", body["message"])
	}
	if !rc.Handled {
		t.Fatal("expected request context to be marked handled")
	}
}

func TestStartImpersonationHandler_Success(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	actorSessionID := "sess-1"
	var gotActorUserID string
	var gotActorSessionID *string
	var gotReq types.StartImpersonationRequest

	h := NewStartImpersonationHandler(impersonationUseCaseStub{
		startFn: func(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
			gotActorUserID = actorUserID
			gotActorSessionID = actorSessionID
			gotReq = req
			return &types.StartImpersonationResult{
				Impersonation: &types.Impersonation{
					ID:             "imp-1",
					ActorUserID:    actorUserID,
					TargetUserID:   req.TargetUserID,
					Reason:         req.Reason,
					ActorSessionID: actorSessionID,
					StartedAt:      now,
					ExpiresAt:      now.Add(15 * time.Minute),
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				SessionID:    actorSessionID,
				SessionToken: new("raw-impersonation-token"),
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/impersonations/start", strings.NewReader(`{"target_user_id":"u-2","reason":"support"}`))
	req, rc := withReqCtx(req)
	userID := "actor-1"
	rc.UserID = &userID
	rc.Values[models.ContextSessionID.String()] = actorSessionID

	h.Handler().ServeHTTP(httptest.NewRecorder(), req)

	if gotActorUserID != "actor-1" {
		t.Fatalf("expected actor user id actor-1, got %q", gotActorUserID)
	}
	if gotActorSessionID == nil || *gotActorSessionID != actorSessionID {
		t.Fatalf("expected actor session id %q", actorSessionID)
	}
	if gotReq.TargetUserID != "u-2" || gotReq.Reason != "support" {
		t.Fatalf("unexpected start request: %+v", gotReq)
	}
	if gotCtxToken, ok := rc.Values[models.ContextSessionToken.String()].(string); !ok || gotCtxToken != "raw-impersonation-token" {
		t.Fatalf("expected impersonation session token in request context, got %v", rc.Values[models.ContextSessionToken.String()])
	}
	if gotCtxSessionID, ok := rc.Values[models.ContextSessionID.String()].(string); !ok || gotCtxSessionID != actorSessionID {
		t.Fatalf("expected session id %q in context, got %v", actorSessionID, rc.Values[models.ContextSessionID.String()])
	}
	if authSuccess, ok := rc.Values[models.ContextAuthSuccess.String()].(bool); !ok || !authSuccess {
		t.Fatalf("expected auth success marker true, got %v", rc.Values[models.ContextAuthSuccess.String()])
	}
	if rc.UserID == nil || *rc.UserID != "u-2" {
		t.Fatalf("expected context user switched to target user, got %v", rc.UserID)
	}
	if rc.ResponseStatus != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rc.ResponseStatus)
	}
	body := decodeJSONBody[startImpersonationResponse](t, rc)
	if body.Message != "impersonation started" {
		t.Fatalf("expected success message, got %q", body.Message)
	}
	if body.Data.ID != "imp-1" || body.Data.ActorUserID != "actor-1" || body.Data.TargetUserID != "u-2" {
		t.Fatalf("unexpected response data: %+v", body.Data)
	}
}

func TestEndImpersonationByIDHandler_Unauthorized(t *testing.T) {
	h := NewEndImpersonationByIDHandler(impersonationUseCaseStub{})
	req := httptest.NewRequest(http.MethodDelete, "/admin/impersonations/imp-1", nil)
	req.SetPathValue("impersonation_id", "imp-1")
	req, rc := withReqCtx(req)

	h.Handler().ServeHTTP(httptest.NewRecorder(), req)

	if rc.ResponseStatus != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rc.ResponseStatus)
	}
	body := decodeJSONBody[map[string]any](t, rc)
	if body["message"] != "Unauthorized" {
		t.Fatalf("expected message Unauthorized, got %v", body["message"])
	}
}

func TestEndImpersonationByIDHandler_Success(t *testing.T) {
	var gotActorUserID string
	var gotReq types.StopImpersonationRequest

	h := NewEndImpersonationByIDHandler(impersonationUseCaseStub{
		stopFn: func(ctx context.Context, actorUserID string, req types.StopImpersonationRequest) error {
			gotActorUserID = actorUserID
			gotReq = req
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/admin/impersonations/imp-1", nil)
	req.SetPathValue("impersonation_id", "imp-1")
	req, rc := withReqCtx(req)
	userID := "actor-1"
	rc.UserID = &userID

	h.Handler().ServeHTTP(httptest.NewRecorder(), req)

	if gotActorUserID != "actor-1" {
		t.Fatalf("expected actor id actor-1, got %q", gotActorUserID)
	}
	if gotReq.ImpersonationID == nil || *gotReq.ImpersonationID != "imp-1" {
		t.Fatalf("expected impersonation id imp-1, got %+v", gotReq)
	}
	if rc.ResponseStatus != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rc.ResponseStatus)
	}
	body := decodeJSONBody[map[string]any](t, rc)
	if body["message"] != "impersonation ended" {
		t.Fatalf("expected message impersonation ended, got %v", body["message"])
	}
}

func TestGetAllImpersonationsHandler_Success(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	h := NewGetAllImpersonationsHandler(impersonationUseCaseStub{
		getAllFn: func(ctx context.Context) ([]types.Impersonation, error) {
			return []types.Impersonation{{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "u-2", Reason: "support", StartedAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/impersonations", nil)
	req, rc := withReqCtx(req)

	h.Handler().ServeHTTP(httptest.NewRecorder(), req)

	if rc.ResponseStatus != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rc.ResponseStatus)
	}
	body := decodeJSONBody[getAllImpersonationsResponse](t, rc)
	if len(body.Data) != 1 || body.Data[0].ID != "imp-1" {
		t.Fatalf("unexpected data payload: %+v", body.Data)
	}
}

func TestGetImpersonationByIDHandler_NotFound(t *testing.T) {
	h := NewGetImpersonationByIDHandler(impersonationUseCaseStub{
		getByID: func(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
			return nil, errors.New("impersonation not found")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/impersonations/imp-missing", nil)
	req.SetPathValue("impersonation_id", "imp-missing")
	req, rc := withReqCtx(req)

	h.Handler().ServeHTTP(httptest.NewRecorder(), req)

	if rc.ResponseStatus != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rc.ResponseStatus)
	}
	body := decodeJSONBody[map[string]any](t, rc)
	if body["message"] != "impersonation not found" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
	if !rc.Handled {
		t.Fatal("expected request context to be marked handled")
	}
}

func TestMapImpersonationErrorStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil error", err: nil, want: http.StatusInternalServerError},
		{name: "unauthorized", err: errors.New("Unauthorized action"), want: http.StatusUnauthorized},
		{name: "forbidden", err: errors.New("forbidden operation"), want: http.StatusForbidden},
		{name: "not found", err: errors.New("record not found"), want: http.StatusNotFound},
		{name: "bad request marker required", err: errors.New("target user is required"), want: http.StatusBadRequest},
		{name: "bad request marker no active", err: errors.New("no active impersonation found"), want: http.StatusBadRequest},
		{name: "default", err: errors.New("something unexpected"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapImpersonationErrorStatus(tt.err); got != tt.want {
				t.Fatalf("expected status %d, got %d", tt.want, got)
			}
		})
	}
}

type startImpersonationResponse struct {
	Message string              `json:"message"`
	Data    types.Impersonation `json:"data"`
}

type getAllImpersonationsResponse struct {
	Data []types.Impersonation `json:"data"`
}

func decodeJSONBody[T any](t *testing.T, rc *models.RequestContext) T {
	t.Helper()
	var payload T
	if err := json.Unmarshal(rc.ResponseBody, &payload); err != nil {
		t.Fatalf("failed to decode response body %q: %v", string(rc.ResponseBody), err)
	}
	return payload
}
