package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/usecases"
)

// Mocks

type mockUserServiceWithError struct{}

func (m *mockUserServiceWithError) GetAll(ctx context.Context, cursor *string, limit int) ([]models.User, *string, error) {
	return nil, nil, errors.New("database error")
}

func (m *mockUserServiceWithError) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, errors.New("database error")
}

func (m *mockUserServiceWithError) GetByID(ctx context.Context, id string) (*models.User, error) {
	return &models.User{ID: id, Email: "test@example.com"}, nil
}

func (m *mockUserServiceWithError) Create(ctx context.Context, name string, email string, emailVerified bool, image *string, metadata json.RawMessage) (*models.User, error) {
	return nil, errors.New("database error")
}

func (m *mockUserServiceWithError) Update(ctx context.Context, user *models.User) (*models.User, error) {
	return user, nil
}

func (m *mockUserServiceWithError) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	return nil
}

func (m *mockUserServiceWithError) Delete(ctx context.Context, id string) error {
	return nil
}

// Tests

func TestSignInHandler_ValidRequestWithExistingUser(t *testing.T) {
	payload := SignInPayload{
		Email: "test@example.com",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/magic-link/sign-in", bytes.NewReader(body))
	w := httptest.NewRecorder()

	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Path:           "/magic-link/sign-in",
		Method:         "POST",
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         make(map[string]any),
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)

	useCase := &usecases.SignInUseCaseImpl{
		GlobalConfig: &models.Config{
			BaseURL:  "http://localhost",
			BasePath: "/auth",
		},
		PluginConfig: &types.MagicLinkPluginConfig{
			ExpiresIn: 15 * time.Minute,
		},
		Logger:              &tests.MockLogger{},
		UserService:         &tests.MockUserService{},
		AccountService:      &tests.MockAccountService{},
		TokenService:        &tests.MockTokenService{},
		VerificationService: &tests.MockVerificationService{},
		MailerService:       &tests.MockMailerService{},
	}

	handler := &SignInHandler{UseCase: useCase}
	handlerFunc := handler.Handler()
	handlerFunc(w, req)

	if reqCtx.ResponseStatus != http.StatusOK {
		t.Fatalf("expected status OK, got %d", reqCtx.ResponseStatus)
	}

	var resp types.SignInResponse
	_ = json.Unmarshal(reqCtx.ResponseBody, &resp)
	if resp.Message == "" {
		t.Fatal("expected message in response")
	}
}

func TestSignInHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/magic-link/sign-in", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Path:           "/magic-link/sign-in",
		Method:         "POST",
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         make(map[string]any),
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)

	useCase := &usecases.SignInUseCaseImpl{}

	handler := &SignInHandler{UseCase: useCase}
	handlerFunc := handler.Handler()
	handlerFunc(w, req)

	if !reqCtx.Handled {
		t.Fatal("expected request to be marked as handled")
	}
	if reqCtx.ResponseStatus != http.StatusUnprocessableEntity {
		t.Fatalf("expected status UnprocessableEntity, got %d", reqCtx.ResponseStatus)
	}
	var errResp map[string]any
	if err := json.Unmarshal(reqCtx.ResponseBody, &errResp); err != nil {
		t.Fatalf("expected JSON error response, got %v", err)
	}
	if errResp["message"] != "invalid request body" {
		t.Fatalf("expected invalid request body message, got %v", errResp["message"])
	}
}

func TestSignInHandler_UseCaseError(t *testing.T) {
	payload := SignInPayload{
		Email: "test@example.com",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/magic-link/sign-in", bytes.NewReader(body))
	w := httptest.NewRecorder()

	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Path:           "/magic-link/sign-in",
		Method:         "POST",
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         make(map[string]any),
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)

	userSvc := &mockUserServiceWithError{}

	useCase := &usecases.SignInUseCaseImpl{
		GlobalConfig:        &models.Config{},
		PluginConfig:        &types.MagicLinkPluginConfig{},
		Logger:              &tests.MockLogger{},
		UserService:         userSvc,
		AccountService:      &tests.MockAccountService{},
		TokenService:        &tests.MockTokenService{},
		VerificationService: &tests.MockVerificationService{},
		MailerService:       &tests.MockMailerService{},
	}

	handler := &SignInHandler{UseCase: useCase}
	handlerFunc := handler.Handler()
	handlerFunc(w, req)

	if !reqCtx.Handled {
		t.Fatal("expected request to be marked as handled")
	}
	if reqCtx.ResponseStatus != http.StatusBadRequest {
		t.Fatalf("expected status BadRequest, got %d", reqCtx.ResponseStatus)
	}

	var errResp map[string]any
	_ = json.Unmarshal(reqCtx.ResponseBody, &errResp)
	if errResp["message"] == nil {
		t.Fatal("expected error message in response")
	}
}

func TestSignInHandler_ResponseStructure(t *testing.T) {
	payload := SignInPayload{
		Email: "test@example.com",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/magic-link/sign-in", bytes.NewReader(body))
	w := httptest.NewRecorder()

	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Path:           "/magic-link/sign-in",
		Method:         "POST",
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         make(map[string]any),
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)

	useCase := &usecases.SignInUseCaseImpl{
		GlobalConfig: &models.Config{
			BaseURL:  "http://localhost",
			BasePath: "/auth",
		},
		PluginConfig: &types.MagicLinkPluginConfig{
			ExpiresIn: 15 * time.Minute,
		},
		Logger:              &tests.MockLogger{},
		UserService:         &tests.MockUserService{},
		AccountService:      &tests.MockAccountService{},
		TokenService:        &tests.MockTokenService{},
		VerificationService: &tests.MockVerificationService{},
		MailerService:       &tests.MockMailerService{},
	}

	handler := &SignInHandler{UseCase: useCase}
	handlerFunc := handler.Handler()
	handlerFunc(w, req)

	if reqCtx.ResponseStatus != http.StatusOK {
		t.Fatalf("expected status OK, got %d", reqCtx.ResponseStatus)
	}

	var resp types.SignInResponse
	err := json.Unmarshal(reqCtx.ResponseBody, &resp)
	if err != nil {
		t.Fatalf("expected valid JSON response, got error: %v", err)
	}
	if resp.Message == "" {
		t.Fatal("expected non-empty message in response")
	}
}
