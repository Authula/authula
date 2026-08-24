package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Authula/authula/models"
)

func PtrString(s string) *string {
	return &s
}

func MarshalToJSON(t *testing.T, payload any) []byte {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	return body
}

func NewHandlerRequest(t *testing.T, method string, path string, body []byte, userID *string) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()

	reader := bytes.NewReader(body)
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	var actor *models.Actor
	if userID != nil {
		actor = &models.Actor{ID: *userID, Type: models.ActorUser}
	}
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Path:           path,
		Method:         method,
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         make(map[string]any),
		Actor:          actor,
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)
	reqCtx.Request = req

	return req, w, reqCtx
}

func NewHandlerRequestWithActor(t *testing.T, method string, path string, body []byte, actor *models.Actor) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()

	reader := bytes.NewReader(body)
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Path:           path,
		Method:         method,
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         make(map[string]any),
		Actor:          actor,
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)
	reqCtx.Request = req

	return req, w, reqCtx
}

func NewRequestContext(t *testing.T, method, path string, headers map[string]string) *models.RequestContext {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return &models.RequestContext{
		Request:        req,
		ResponseWriter: httptest.NewRecorder(),
		Path:           path,
		Method:         method,
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         map[string]any{},
	}
}

func AssertErrorMessage(t *testing.T, reqCtx *models.RequestContext, status int, message string) {
	t.Helper()

	if !reqCtx.Handled {
		t.Fatal("expected request to be marked as handled")
	}
	if reqCtx.ResponseStatus != status {
		t.Fatalf("expected status %d, got %d", status, reqCtx.ResponseStatus)
	}

	payload := DecodeResponseJSON[struct {
		Message string `json:"message"`
	}](t, reqCtx)
	if payload.Message != message {
		t.Fatalf("expected message %q, got %v", message, payload.Message)
	}
}

func AssertErrorResponse(t *testing.T, reqCtx *models.RequestContext, status int, code string, message string) {
	t.Helper()

	if !reqCtx.Handled {
		t.Fatal("expected request to be marked as handled")
	}
	if reqCtx.ResponseStatus != status {
		t.Fatalf("expected status %d, got %d", status, reqCtx.ResponseStatus)
	}

	payload := DecodeResponseJSON[struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}](t, reqCtx)
	if payload.Code != code {
		t.Fatalf("expected code %q, got %q", code, payload.Code)
	}
	if payload.Message != message {
		t.Fatalf("expected message %q, got %q", message, payload.Message)
	}
}

func DecodeResponseJSON[T any](t *testing.T, reqCtx *models.RequestContext) T {
	t.Helper()

	var payload T
	if err := json.Unmarshal(reqCtx.ResponseBody, &payload); err != nil {
		t.Fatalf("failed to decode response json: %v body=%s", err, string(reqCtx.ResponseBody))
	}

	return payload
}
