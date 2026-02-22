---
name: handlers-and-http
description: Implement HTTP handlers that parse requests, invoke use cases, and format responses following REST conventions.
---

# Handlers & HTTP Integration

## When to use this skill

Use this skill when you need to:
- Create HTTP endpoint handlers for authentication flows
- Parse HTTP requests into domain types
- Handle HTTP-specific concerns (status codes, headers, serialization)
- Return properly formatted JSON responses
- Manage error responses with appropriate status codes
- Keep handlers thin by delegating business logic to use cases

## Pattern overview

Handlers are the HTTP boundary of the application. They:
- Accept HTTP requests and context
- Parse request data (JSON body, URL params, headers)
- Call use cases to execute business logic
- Transform use case responses to HTTP responses
- Return appropriate status codes and headers
- Delegate all business logic to use cases

### Key principles

1. **Thin handlers**: Business logic lives in use cases, not handlers
2. **Use case coordination**: Handlers call use cases, not services directly
3. **HTTP boundaries**: Handlers handle only HTTP serialization and status codes
4. **Error mapping**: Map domain errors to HTTP status codes
5. **Context propagation**: Pass `r.Context()` to use cases for timeout and cancellation

## Implementation checklist

### Step 1: Define the handler struct

```
type signUpHandler struct {
    useCase     usecases.SignUpUseCase
    logger      models.Logger
}

func NewSignUpHandler(
    useCase usecases.SignUpUseCase,
    logger models.Logger,
) *signUpHandler {
    return &signUpHandler{
        useCase: useCase,
        logger:  logger,
    }
}
```

- Include the use case and any needed dependencies (logger, config)
- Constructor function accepts dependencies
- Handler is a receiver for the HTTP handler method

**Reference**: See [plugins/email-password/handlers/sign_up_handler.go](../../../plugins/email-password/handlers/sign_up_handler.go)

### Step 2: Implement the handler method signature

```
func (h *signUpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
    // ...
}
```

Or for use case-based handlers:

```
func (h *signUpHandler) Handle(w http.ResponseWriter, r *http.Request) error {
    // ...
}
```

- Accept `http.ResponseWriter` and `*http.Request`
- Return error (framework handles conversion to HTTP response)
- Use handler context (`r.Context()`) for timeouts and cancellation

**Reference**: See [plugins/email-password/handlers/sign_in_handler.go](../../../plugins/email-password/handlers/sign_in_handler.go)

### Step 3: Parse the request

```
import (
    "github.com/GoBetterAuth/go-better-auth/v2/internal/util"
)

type signUpRequest struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

type signUpResponse struct {
    User    *models.User    `json:"user"`
    Account *models.Account `json:"account"`
}

func (h *signUpHandler) Handle(w http.ResponseWriter, r *http.Request) error {
    ctx := r.Context()
    reqCtx, _ := models.GetRequestContext(ctx)

    var req signUpRequest
		if err := util.ParseJSON(r, &req); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}
    
    // Extract URL parameters
    userID := r.PathValue("user_id")
    
    // Extract query parameters
    page := r.URL.Query().Get("page")
    
    return nil
}
```

- Use `util.ParseJSON()` for JSON request bodies
- Validate parsed input before calling use case
- Return `APIError` with appropriate status code for bad input
- Use routing framework functions to extract URL/path parameters

**Reference**: See [plugins/email-password/handlers/](../../../plugins/email-password/handlers/) for parsing patterns

### Step 4: Call the use case

```
// Call use case with request context
result, err := h.useCase.Execute(r.Context(), req)
if err != nil {
    h.logger.Error("sign up failed", "error", err)
    return err
}
```

- Convert parsed request to use case request type
- Pass `r.Context()` to preserve request timeout and cancellation
- Always log errors for debugging
- Map domain errors to HTTP status codes

**Reference**: See [plugins/email-password/handlers/sign_up_handler.go](../../../plugins/email-password/handlers/sign_up_handler.go) for orchestration pattern

### Step 5: Format the response

```
// Map use case response to HTTP response type
response := signUpResponse{
    User:    result.User,
    Account: result.Account,
}
reqCtx.SetJSONResponse(http.StatusCreated, response)
return nil
```

- Use appropriate status codes (200 OK, 201 Created, 400 Bad Request, 500 Server Error)
- Encode response as JSON using `reqCtx.SetJSONResponse()`
- Map domain models to HTTP response structs as needed

**Reference**: See handler implementations for response formatting

### Step 6: Register the handler in routing

Handlers are registered in the plugin's `Routes()` method:

```
func (p *EmailPasswordPlugin) Routes() []models.Route {
    signUpHandler := handlers.NewSignUpHandler(p.signUpUseCase, p.logger)
    
    return []models.Route{
        {
            Method:  http.MethodPost,
            Path:    "/sign-up",
            Handler: signUpHandler.Handle,
        },
    }
}
```

- Create handler instances with their dependencies
- Return `models.Route` with HTTP method, path, and handler
- Handlers are registered per-route

**Reference**: See [plugins/email-password/api.go](../../../plugins/email-password/api.go) for route registration pattern

## Code file references

Study these implementations:

- **Handler examples**:
  - [plugins/email-password/handlers/sign_up_handler.go](../../../plugins/email-password/handlers/sign_up_handler.go)
  - [plugins/email-password/handlers/sign_in_handler.go](../../../plugins/email-password/handlers/sign_in_handler.go)
  - [plugins/email-password/handlers/verify_email_handler.go](../../../plugins/email-password/handlers/verify_email_handler.go)
- **Core handlers** (simple patterns):
  - [internal/handlers/get_me_handler.go](../../../internal/handlers/get_me_handler.go)
  - [internal/handlers/sign_out_handler.go](../../../internal/handlers/sign_out_handler.go)
- **Handler registration**: [plugins/email-password/api.go](../../../plugins/email-password/api.go)
- **Framework helpers**: [internal/router/](../../../internal/router/) for utilities like deferred response writing

## HTTP error handling patterns

### Success responses

```
reqCtx.SetJSONResponse(http.StatusOK, response)
```

### Error responses

Create domain error types that map to HTTP status:

```
type APIError struct {
    Code       string
    Message    string
    StatusCode int
}

func (e *APIError) Error() string {
    return e.Message
}
```

The HTTP framework (chi or similar) should convert these to appropriate responses.

**Reference**: See error handling patterns in handler implementations

## Common mistakes to avoid

1. **Putting business logic in handlers**: Business logic belongs in use cases
2. **Handlers calling services directly**: Always go through use cases for composition
3. **Not propagating request context**: Always pass `r.Context()` to use cases for timeouts
4. **Forgetting error handling**: Every use case call should check and handle errors
5. **Incorrect HTTP status codes**: Use 400 for client errors, 500 for server errors, 201 for created
6. **Not setting Content-Type header**: Always set `Content-Type: application/json` for JSON responses
7. **Logging sensitive data**: Never log passwords or tokens
8. **Not validating input**: Validate request data before calling use cases

## Related skills

- See [use-cases-and-orchestration](../use-cases-and-orchestration) for how use cases work
- See [services-and-interfaces](../services-and-interfaces) for service patterns
- See [dependency-injection](../dependency-injection) for handler setup
