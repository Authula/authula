package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
	"github.com/Authula/authula/plugins/magic-link/usecases"
)

func TestSignInHandler(t *testing.T) {
	name := "John Doe"

	testCases := []struct {
		name         string
		requestBody  []byte
		setup        func(t *testing.T, reqCtx *models.RequestContext, userService *internaltests.MockUserService, accountService *internaltests.MockAccountService, tokenService *internaltests.MockTokenService, verificationService *internaltests.MockVerificationService)
		assertResult func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name:        "valid request with existing user",
			requestBody: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "test@example.com"}),
			setup: func(t *testing.T, reqCtx *models.RequestContext, userService *internaltests.MockUserService, accountService *internaltests.MockAccountService, tokenService *internaltests.MockTokenService, verificationService *internaltests.MockVerificationService) {
				userService.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-123", Email: "test@example.com"}, nil).Once()
				tokenService.On("Generate").Return("token-123", nil).Once()
				tokenService.On("Hash", "token-123").Return("hashed-token-123").Once()
				verificationService.On("Create", mock.Anything, "user-123", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", 15*time.Minute).
					Return(&models.Verification{ID: "verif-1"}, nil).Once()

				t.Cleanup(func() {
					userService.AssertExpectations(t)
					tokenService.AssertExpectations(t)
					verificationService.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				if reqCtx.ResponseStatus != http.StatusOK {
					t.Fatalf("expected status OK, got %d", reqCtx.ResponseStatus)
				}

				var resp types.SignInResponse
				if err := json.Unmarshal(reqCtx.ResponseBody, &resp); err != nil {
					t.Fatalf("expected valid JSON response, got error: %v", err)
				}
				if resp.Message == "" {
					t.Fatal("expected message in response")
				}
			},
		},
		{
			name:        "invalid json",
			requestBody: []byte("invalid json"),
			setup: func(t *testing.T, reqCtx *models.RequestContext, userService *internaltests.MockUserService, accountService *internaltests.MockAccountService, tokenService *internaltests.MockTokenService, verificationService *internaltests.MockVerificationService) {
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
			},
		},
		{
			name:        "use case error",
			requestBody: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "test@example.com"}),
			setup: func(t *testing.T, reqCtx *models.RequestContext, userService *internaltests.MockUserService, accountService *internaltests.MockAccountService, tokenService *internaltests.MockTokenService, verificationService *internaltests.MockVerificationService) {
				userService.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("database error")).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
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
			},
		},
		{
			name:        "already authenticated returns bad request",
			requestBody: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "test@example.com"}),
			setup: func(t *testing.T, reqCtx *models.RequestContext, userService *internaltests.MockUserService, accountService *internaltests.MockAccountService, tokenService *internaltests.MockTokenService, verificationService *internaltests.MockVerificationService) {
				reqCtx.UserID = &name
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				if reqCtx.ResponseStatus != http.StatusBadRequest {
					t.Fatalf("expected status BadRequest, got %d", reqCtx.ResponseStatus)
				}
				var errResp map[string]any
				_ = json.Unmarshal(reqCtx.ResponseBody, &errResp)
				if errResp["message"] != "you're already authenticated." {
					t.Fatalf("expected authenticated message, got %v", errResp["message"])
				}
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := tt.requestBody
			if reqBody == nil {
				reqBody = []byte(`{}`)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/magic-link/sign-in", reqBody, nil)

			userService := &internaltests.MockUserService{}
			accountService := &internaltests.MockAccountService{}
			tokenService := &internaltests.MockTokenService{}
			verificationService := &internaltests.MockVerificationService{}

			pluginConfig := &types.MagicLinkPluginConfig{ExpiresIn: 15 * time.Minute}
			useCase := &usecases.SignInUseCaseImpl{
				GlobalConfig:        &models.Config{BaseURL: "http://localhost", BasePath: "/auth"},
				PluginConfig:        pluginConfig,
				Logger:              &internaltests.MockLogger{},
				UserService:         userService,
				AccountService:      accountService,
				TokenService:        tokenService,
				VerificationService: verificationService,
				MailerService:       nil,
			}

			tt.setup(t, reqCtx, userService, accountService, tokenService, verificationService)

			handler := &SignInHandler{UseCase: useCase}
			handler.Handler()(w, req)

			tt.assertResult(t, reqCtx)
		})
	}
}
