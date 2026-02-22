---
name: use-cases-and-orchestration
description: Orchestrate services and repositories through use cases to implement application-level workflows and business scenarios.
---

# Use Cases & Orchestration

## When to use this skill

Use this skill when you need to:
- Implement workflows that span multiple services
- Encapsulate complex authentication or account operations
- Define reusable operations that handlers can invoke
- Keep handlers thin by moving logic into use cases
- Test complex business flows independently from HTTP handlers

## Pattern overview

Use cases are the application's command/query layer. They:
- Coordinate multiple services to accomplish a user-facing operation
- Receive dependencies (services) via constructor injection
- Implement specific business workflows (SignUp, SignIn, ChangePassword, etc.)
- Return domain models or errors, never HTTP responses
- Are called by handlers which handle HTTP concerns

### Key principles

1. **Service orchestration**: Use cases call services to accomplish workflows
2. **Business-focused**: Methods represent user actions, not HTTP operations
3. **Dependency injection**: Services passed at construction time
4. **Pure domain**: No HTTP, no context routing—just business logic
5. **Testable**: Can be tested independently by mocking services
6. **Single operation**: One use case often = one public method (or closely related methods)

## Implementation checklist

### Step 1: Define the use case interface

In `usecases/interfaces.go`:

```
type YourUseCase interface {
    Execute(ctx context.Context, request *YourRequest) (*YourResponse, error)
}
```

Or simply define the struct if it's a simple operation:

```
type SignUpUseCase struct {
    UserService    services.UserService
    AccountService services.AccountService
    MailerService  services.MailerService
}
```

- Method names reflect user actions: Execute, SignUp, ChangePassword, etc.
- Accept domain request/response types, not HTTP types
- Every method takes `context.Context`
- Return domain models, not serialized JSON

**Reference**: See [internal/usecases/interfaces.go](../../../internal/usecases/interfaces.go) for core use case definitions

### Step 2: Create the use case struct

```
type signUpUseCase struct {
    userService    services.UserService
    accountService services.AccountService
    mailerService  services.MailerService
}
```

- Keep struct unexported (lowercase)
- Include all service dependencies as private fields
- Optional: include logger, config for validation and logging

**Reference**: See [plugins/email-password/usecases/sign_up_usecase.go](../../../plugins/email-password/usecases/sign_up_usecase.go)

### Step 3: Implement the constructor

```
func NewSignUpUseCase(
    userService services.UserService,
    accountService services.AccountService,
    mailerService services.MailerService,
) SignUpUseCase {
    return &signUpUseCase{
        userService:    userService,
        accountService: accountService,
        mailerService:  mailerService,
    }
}
```

- Return the interface, not the concrete struct
- Accept only the services needed (no unused dependencies)
- Dependencies are listed in logical order

**Reference**: See plugin use case constructors like [plugins/email-password/usecases/sign_in_usecase.go](../../../plugins/email-password/usecases/sign_in_usecase.go)

### Step 4: Implement the execute method

The use case orchestrates services:

```
func (uc *signUpUseCase) Execute(
    ctx context.Context, 
    req *SignUpRequest,
) (*SignUpResponse, error) {
    // Step 1: Validate request
    if req.Email == "" {
        return nil, errors.New("email required")
    }
    
    // Step 2: Call services in sequence
    user, err := uc.userService.Create(ctx, req.Name, req.Email, false, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    // Step 3: Call dependent service
    account, err := uc.accountService.Create(
        ctx,
        user.ID,
        user.Email,
        "email-password",
        &req.Password,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create account: %w", err)
    }
    
    // Step 4: Return domain response
    return &SignUpResponse{
        User:    user,
        Account: account,
    }, nil
}
```

- Orchestrate multiple services sequentially
- Handle errors at each step (don't swallow errors)
- Return domain models in response structs
- Keep use cases focused; complex multi-step workflows might deserve their own method

**Reference**: See [plugins/email-password/usecases/sign_up_usecase.go](../../../plugins/email-password/usecases/sign_up_usecase.go) for multi-step orchestration

### Step 5: Define request and response types

In `usecases/types.go`:

```
type SignUpRequest struct {
    Name     string
    Email    string
    Password string
}

type SignUpResponse struct {
    User    *models.User
    Account *models.Account
}
```

- Request types capture user input (validated by use case)
- Response types return domain models and computed results
- Optional: Add validation methods on request types

**Reference**: See [plugins/email-password/types/types.go](../../../plugins/email-password/types/types.go)

### Step 6: Use the use case in handlers

Handlers invoke use cases (see handlers-and-http skill):

```
func (h *signUpHandler) Handle(
    w http.ResponseWriter,
    r *http.Request,
) error {
    var req SignUpRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        return err
    }
    
    resp, err := h.useCase.Execute(r.Context(), &req)
    if err != nil {
        return err
    }
    
    return respondJSON(w, http.StatusCreated, resp)
}
```

- Handlers parse HTTP input into use case request types
- Call use case with parsed request
- Convert use case response to HTTP response
- Keep handlers thin; logic lives in use cases

## Code file references

Study these implementations:

- **Core use cases**: [internal/api.go](../../../internal/api.go) - Wires core use cases
- **Core use case example**: [internal/usecases/](../../../internal/usecases/) - GetMeUseCase, SignOutUseCase
- **Plugin use cases**:
  - [plugins/email-password/usecases/](../../../plugins/email-password/usecases/) - Sign up, sign in, password reset workflows
  - [plugins/magic-link/api.go](../../../plugins/magic-link/api.go) - Plugin use case wiring
- **Use case types**: [plugins/email-password/types/types.go](../../../plugins/email-password/types/types.go)

## Common mistakes to avoid

1. **Use cases returning HTTP status codes**: Let handlers handle HTTP concerns
2. **Putting validation in the handler**: Validation belongs in the use case
3. **Use cases calling other use cases**: Refactor to service-based orchestration instead
4. **Creating new services in use cases**: Services must be dependency-injected
5. **Use cases with no clear purpose**: If it's just calling one service, consider moving it to the handler
6. **Not handling errors properly**: Each service call should have error checking and context-aware error messages
7. **Mixing HTTP and domain concerns**: Use case input/output should be domain types, not HTTP types
8. **Use cases doing data transformation only**: If there's no orchestration, move logic to services

## Related skills

- See [services-and-interfaces](../services-and-interfaces) for how services work
- See [handlers-and-http](../handlers-and-http) for how handlers call use cases
- See [dependency-injection](../dependency-injection) for how use cases are wired
