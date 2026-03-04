# Handler Layer Testing (Request-Response Integration Pattern)

Handler layer tests occupy a middle ground between unit tests (mocked repositories) and full integration tests (real database). These tests exercise HTTP request handling, context mutations, and error mapping without requiring a full fixture setup.

## When to Use This Pattern

- Testing **handler functions** that receive `*models.RequestContext` and `http.ResponseWriter`.
- Testing HTTP request parsing, validation, and response encoding.
- Testing error-to-status-code mapping at the handler layer.
- Avoiding database setup/teardown while still testing realistic request flows.

## Key Concepts

### 1. Shared Test Helpers

Create a `test_helpers_test.go` file with reusable utilities:

```go
// Marshal Go struct to JSON bytes for request bodies
func mustJSON(t *testing.T, payload any) []byte {
    body, err := json.Marshal(payload)
    if err != nil {
        t.Fatalf("failed to marshal payload: %v", err)
    }
    return body
}

// Construct *http.Request + *models.RequestContext + *httptest.ResponseRecorder
func newAdminHandlerRequest(t *testing.T, method, path string, body []byte) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
    var reader *bytes.Reader
    if body == nil {
        reader = bytes.NewReader(nil)
    } else {
        reader = bytes.NewReader(body)
    }

    req := httptest.NewRequest(method, path, reader)
    w := httptest.NewRecorder()
    reqCtx := &models.RequestContext{
        Request:        req,
        ResponseWriter: w,
        Path:           path,
        Method:         method,
        Headers:        req.Header,
        ClientIP:       "127.0.0.1",
        Values:         make(map[string]any),
    }

    ctx := models.SetRequestContext(context.Background(), reqCtx)
    req = req.WithContext(ctx)
    reqCtx.Request = req
    return req, w, reqCtx
}

// Decode response body to map[string]any for assertions
func decodeResponseJSON(t *testing.T, reqCtx *models.RequestContext) map[string]any {
    var payload map[string]any
    if err := json.Unmarshal(reqCtx.ResponseBody, &payload); err != nil {
        t.Fatalf("failed to decode response json: %v body=%s", err, string(reqCtx.ResponseBody))
    }
    return payload
}

// Assert error responses (status + message)
func assertErrorMessage(t *testing.T, reqCtx *models.RequestContext, status int, message string) {
    if !reqCtx.Handled {
        t.Fatal("expected request to be marked as handled")
    }
    if reqCtx.ResponseStatus != status {
        t.Fatalf("expected status %d, got %d", status, reqCtx.ResponseStatus)
    }
    payload := decodeResponseJSON(t, reqCtx)
    if payload["message"] != message {
        t.Fatalf("expected message %q, got %v", message, payload["message"])
    }
}
```

### 2. Interface-Complete Mock Use Cases

For each use case interface, create a mock using `testify/mock`:

```go
type mockUsersUseCase struct {
    mock.Mock
}

func (m *mockUsersUseCase) GetAll(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
    args := m.Called(ctx, cursor, limit)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*types.UsersPage), args.Error(1)
}

func (m *mockUsersUseCase) Create(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
    args := m.Called(ctx, request)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.User), args.Error(1)
}

// ... implement all interface methods
```

**Key Pattern**: Use `.On(...).Return(...).Once()` to configure call expectations, and `.AssertExpectations(t)` to verify mocks were called as expected.

### 3. Handler Test Structure

Each handler test:

1. Creates a mock use case
2. Configures expectations with `.On(...).Return(...).Once()`
3. Constructs a request via `newAdminHandlerRequest`
4. Sets `RequestContext` values (UserID, path params, etc.)
5. Calls the handler
6. Asserts on `RequestContext` mutations (UserID, SessionID, AuthSuccess, etc.) and response status/body

```go
func TestStartImpersonationHandler_Success(t *testing.T) {
    // Arrange
    now := time.Now().UTC()
    sessionID := "session-2"
    sessionToken := "session-token"
    actorSessionID := "session-1"

    useCase := &mockImpersonationUseCase{}
    useCase.On("StartImpersonation", mock.Anything, "actor-1", &actorSessionID,
        types.StartImpersonationRequest{
            TargetUserID: "target-1",
            Reason:       "support",
        }).
        Return(&types.StartImpersonationResult{
            NewUserID:     "target-1",
            NewSessionID:  sessionID,
            SessionToken:  sessionToken,
            ImpersonationStartedAt: &now,
        }, nil).Once()

    handler := NewStartImpersonationHandler(useCase)

    // Act
    req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/impersonations",
        mustJSON(t, types.StartImpersonationRequest{
            TargetUserID: "target-1",
            Reason:       "support",
        }))
    userID := "actor-1"
    reqCtx.UserID = &userID
    sessionID = actorSessionID
    reqCtx.SessionID = &sessionID

    handler.Handler()(w, req)

    // Assert
    if reqCtx.UserID == nil || *reqCtx.UserID != "target-1" {
        t.Fatalf("expected UserID to be set to target-1, got %v", reqCtx.UserID)
    }
    if reqCtx.SessionID == nil || *reqCtx.SessionID != sessionID {
        t.Fatalf("expected SessionID to be updated")
    }
    if !reqCtx.AuthSuccess {
        t.Fatal("expected AuthSuccess to be true")
    }

    payload := decodeResponseJSON(t, reqCtx)
    if payload["data"] == nil {
        t.Fatal("expected data in response")
    }

    useCase.AssertExpectations(t)
}
```

## Test Organization

Structure handler tests with subtests for clarity:

```go
func TestUserHandlers(t *testing.T) {
    t.Run("GetAll_InvalidLimit", func(t *testing.T) { ... })
    t.Run("GetAll_Success", func(t *testing.T) { ... })
    t.Run("GetByID_NotFound", func(t *testing.T) { ... })
    t.Run("GetByID_Success", func(t *testing.T) { ... })
    t.Run("Create_InvalidJSON", func(t *testing.T) { ... })
    t.Run("Create_ValidationError", func(t *test.T) { ... })
    t.Run("Create_Success", func(t *testing.T) { ... })
}
```

Or use separate named test functions per handler:

```go
func TestGetUsersHandler_InvalidLimit(t *testing.T) { ... }
func TestGetUsersHandler_Success(t *testing.T) { ... }
func TestCreateUserHandler_InvalidJSON(t *testing.T) { ... }
func TestCreateUserHandler_Success(t *testing.T) { ... }
```

## Coverage Requirements

- Test **success path**: Handler correctly invokes use case, processes response, sets context values.
- Test **error path**: Handler maps use case errors to correct HTTP status codes.
- Test **validation/parsing**: Malformed JSON, missing fields, invalid query params all return proper error responses.
- Test **context mutations**: Handler correctly updates `UserID`, `SessionID`, `AuthSuccess`, etc. for logged-in operations.

## Best Practices

1. **Avoid Duplication**: Use shared helpers (`mustJSON`, `newAdminHandlerRequest`, `assertErrorMessage`) for all tests.
2. **Clear Expectations**: Use `.On(...).Return(...).Once()` to document expected behavior; makes tests self-documenting.
3. **Minimal Setup**: Don't create fixtures or fixtures. Just configure mocks and create a RequestContext manually.
4. **Verify Calls**: Always call `.AssertExpectations(t)` to ensure the handler invoked the use case as expected.
5. **Test Nil Handling**: Some handlers return `(nil, nil)` for not-found; explicitly test this path and verify 404 response.
6. **Check Status Codes**: Verify handlers map semantic errors (NotFound, Conflict, Forbidden) to correct HTTP codes (404, 409, 403).

## Example File Structure

```
plugins/admin/handlers/
  test_helpers_test.go         # Shared request/response utilities + mock use cases (~400 lines)
  impersonation_handlers_test.go    # 4 handlers, ~13 subtests
  users_handlers_test.go           # 5 handlers, ~15 subtests
  role_permission_handlers_test.go  # 11 handlers, ~25 subtests
  state_handlers_test.go           # 12 handlers, ~28 subtests
  user_roles_handlers_test.go      # 5 handlers, ~13 subtests
```
