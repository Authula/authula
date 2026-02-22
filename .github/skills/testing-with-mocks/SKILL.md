---
name: testing-with-mocks
description: Create mock implementations of interfaces for isolated unit testing of services, repositories, and handlers.
---

# Testing with Mocks

## When to use this skill

Use this skill when you need to:
- Test services in isolation by mocking their repositories
- Test handlers by mocking use cases and services
- Create mock implementations that satisfy interfaces
- Control behavior and assertions in unit tests
- Avoid database dependencies in tests
- Test error handling and edge cases

## Pattern overview

The codebase uses hand-written mock implementations that:
- Implement the same interface as the real component
- Allow callers to inject test behavior via function fields
- Track method calls and arguments for assertions
- Can simulate errors or return specific values
- Are simple and explicit (no reflection-based mocking frameworks)

### Key principles

1. **Interface implementation**: Mocks implement the real interface, nothing more
2. **Explicit behavior**: Callers set behavior via fields or closures
3. **Test-friendly**: Mocks are in `*_test.go` files or `internal/tests/`
4. **No magic**: Easy to understand what the mock does
5. **Type-safe**: Mocks are Go code, not generated or dynamic

## Implementation checklist

### Step 1: Create a mock struct

In your test file or `internal/tests/mocks.go`:

```
type MockUserRepository struct {
    GetByIDFn func(ctx context.Context, id string) (*models.User, error)
    CreateFn  func(ctx context.Context, user *models.User) (*models.User, error)
    // ... other methods as function fields
}
```

- Mock struct matches the interface it's implementing
- Each method becomes a function field
- Function field signature matches the real method signature
- Optionally add tracking fields: `GetByIDCalls []struct{ID string}`

**Reference**: See [internal/tests/mocks.go](../../../internal/tests/mocks.go) for comprehensive examples

### Step 2: Implement the interface

```
func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
    if m.GetByIDFn != nil {
        return m.GetByIDFn(ctx, id)
    }
    return nil, errors.New("mock not set up")
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
    if m.CreateFn != nil {
        return m.CreateFn(ctx, user)
    }
    return nil, errors.New("mock not set up")
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) (*models.User, error) {
    if m.UpdateFn != nil {
        return m.UpdateFn(ctx, user)
    }
    return nil, errors.New("mock not set up")
}
```

- Each method delegates to its function field
- Check if function field is nil; return error or nil if not set
- Allows tests to set only the methods they need

**Reference**: See [internal/tests/mocks.go](../../../internal/tests/mocks.go) for patterns

### Step 3: Set up mock behavior in tests

In your test function:

```
func TestUserServiceCreate(t *testing.T) {
    // Create mock repository
    mockRepo := &MockUserRepository{
        CreateFn: func(ctx context.Context, user *models.User) (*models.User, error) {
            // Verify input
            if user.Email == "" {
                return nil, errors.New("email required")
            }
            // Return test data
            user.ID = "test-id-123"
            return user, nil
        },
    }
    
    // Create service with mock
    service := services.NewUserService(mockRepo, mockDbHooks)
    
    // Test the service
    user, err := service.Create(context.Background(), "John", "john@example.com", false, nil)
    
    // Assert results
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.ID != "test-id-123" {
        t.Fatalf("expected user ID test-id-123, got %s", user.ID)
    }
}
```

- Create mock with only the methods you need
- Set function fields to control behavior
- Inject mock into service/handler being tested
- Assert behavior and results

**Reference**: See test files like [internal/services/session_service_test.go](../../../internal/services/session_service_test.go) for full examples

### Step 4: Track calls for assertions

Add tracking fields to mocks:

```
type MockUserRepository struct {
    GetByIDFn    func(ctx context.Context, id string) (*models.User, error)
    GetByIDCalls []struct {
        Ctx context.Context
        ID  string
    }
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
    m.GetByIDCalls = append(m.GetByIDCalls, struct {
        Ctx context.Context
        ID  string
    }{Ctx: ctx, ID: id})
    
    if m.GetByIDFn != nil {
        return m.GetByIDFn(ctx, id)
    }
    return nil, nil
}
```

Then in tests, verify the mock was called correctly:

```
func TestUserServiceGetByID(t *testing.T) {
    mockRepo := &MockUserRepository{
        GetByIDFn: func(ctx context.Context, id string) (*models.User, error) {
            return &models.User{ID: id, Email: "test@example.com"}, nil
        },
    }
    
    service := services.NewUserService(mockRepo, mockDbHooks)
    service.GetByID(context.Background(), "user-123")
    
    // Assert the mock was called
    if len(mockRepo.GetByIDCalls) != 1 {
        t.Fatalf("expected 1 call to GetByID, got %d", len(mockRepo.GetByIDCalls))
    }
    if mockRepo.GetByIDCalls[0].ID != "user-123" {
        t.Fatalf("expected ID user-123, got %s", mockRepo.GetByIDCalls[0].ID)
    }
}
```

**Reference**: See [internal/tests/mocks.go](../../../internal/tests/mocks.go) for call tracking patterns

### Step 5: Mock with strict mode (optional)

Some mocks include strict mode for catching unexpected calls:

```
type MockUserService struct {
    t       *testing.T
    strict  bool
    GetByIDFn func(ctx context.Context, id string) (*models.User, error)
}

func (m *MockUserService) GetByID(ctx context.Context, id string) (*models.User, error) {
    if m.GetByIDFn == nil {
        if m.strict {
            m.t.Fatalf("unexpected call to GetByID")
        }
        return nil, nil
    }
    return m.GetByIDFn(ctx, id)
}
```

- Strict mode fails tests if unexpected methods are called
- Useful for catching unintended service dependencies
- Optional feature; not all mocks need it

**Reference**: See [internal/tests/mocks.go](../../../internal/tests/mocks.go) for strict mode patterns

### Step 6: Test error cases

Mock services can simulate errors:

```
func TestUserServiceCreateWithError(t *testing.T) {
    mockRepo := &MockUserRepository{
        CreateFn: func(ctx context.Context, user *models.User) (*models.User, error) {
            return nil, fmt.Errorf("database error: connection failed")
        },
    }
    
    service := services.NewUserService(mockRepo, mockDbHooks)
    _, err := service.Create(context.Background(), "John", "john@example.com", false, nil)
    
    if err == nil {
        t.Fatal("expected error, got nil")
    }
    if !strings.Contains(err.Error(), "database error") {
        t.Fatalf("expected database error, got %v", err)
    }
}
```

- Set mock function to return errors
- Test that service properly propagates or handles errors
- Test error cases for robustness

## Code file references

Study these mock implementations:

- **Comprehensive mocks**: [internal/tests/mocks.go](../../../internal/tests/mocks.go)
- **MockUserRepository**: [internal/tests/mocks.go](../../../internal/tests/mocks.go)
- **MockAccountService**: [internal/tests/mocks.go](../../../internal/tests/mocks.go)
- **Service tests using mocks**:
  - [internal/services/session_service_test.go](../../../internal/services/session_service_test.go)
  - [plugins/email-password/usecases/sign_up_usecase_test.go](../../../plugins/email-password/usecases/) (if available)

## Testing patterns

### Testing a service

```
func TestUserService(t *testing.T) {
    // Create mock dependency
    mockRepo := &MockUserRepository{
        GetByIDFn: func(ctx context.Context, id string) (*models.User, error) {
            return &models.User{ID: id}, nil
        },
    }
    
    // Create service under test
    service := services.NewUserService(mockRepo, nil)
    
    // Test behavior
    user, err := service.GetByID(context.Background(), "123")
    
    // Assert
    if err != nil || user.ID != "123" {
        t.Fatal("test failed")
    }
}
```

### Testing a handler

```
func TestSignUpHandler(t *testing.T) {
    // Mock the use case
    mockUseCase := &MockSignUpUseCase{
        ExecuteFn: func(ctx context.Context, req *usecases.SignUpRequest) (*usecases.SignUpResponse, error) {
            return &usecases.SignUpResponse{
                User: &models.User{ID: "123", Email: req.Email},
            }, nil
        },
    }
    
    // Create handler
    handler := handlers.NewSignUpHandler(mockUseCase, mockLogger)
    
    // Test with HTTP request
    req := httptest.NewRequest("POST", "/signup", ...)
    w := httptest.NewRecorder()
    
    handler.ServeHTTP(w, req)
    
    // Assert HTTP response
    if w.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d", w.Code)
    }
}
```

**Reference**: See service and handler test files in the codebase

## Common mistakes to avoid

1. **Not implementing all methods**: Mock must implement the entire interface
2. **Ignoring nil checks**: Check if function field is nil before calling
3. **Not tracking calls**: Add call tracking when you need to assert behavior
4. **Complex mock logic**: Keep mocks simple; if they're too complex, test the real component
5. **Sharing mock state between tests**: Create fresh mock for each test
6. **Not handling context**: Mocks should accept context properly
7. **Mocking too much**: Mock only external dependencies; test the real code you wrote
8. **Forgetting error cases**: Test both success and error paths

## Testing guidelines

1. **Test one thing per test**: One interface method being mocked = one behavior being tested
2. **Use clear test names**: Name tests after what they test (TestUserServiceCreateSuccess)
3. **Arrange-Act-Assert**: Set up mocks, call code, verify results
4. **Don't test mocks**: Test real code with mocked dependencies
5. **Integration vs Unit**: Use mocks for unit tests; use real components for integration tests

## Related skills

- See [services-and-interfaces](../services-and-interfaces) for service structure
- See [repositories-and-data-access](../repositories-and-data-access) for repository patterns
- See [use-cases-and-orchestration](../use-cases-and-orchestration) for use case patterns
- See [handlers-and-http](../handlers-and-http) for handler structure
