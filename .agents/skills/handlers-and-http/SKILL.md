---
name: handlers-and-http
description: Implement HTTP handlers that parse requests, invoke use cases, and format responses following REST conventions.
---

# Handlers & HTTP Integration

## When to use this skill

- Create HTTP endpoint handlers for authentication flows
- Parse HTTP requests into domain types
- Handle HTTP-specific concerns (status codes, headers, serialization)
- Return properly formatted JSON responses
- Map domain errors returned from services to HTTP status codes
- Keep handlers thin by delegating to use cases

## Key principles

1. **Thin handlers**: Business logic lives in services, not handlers.
2. **Service coordination**: Handlers invoke services which are interfaces
3. **HTTP boundaries**: Handle only HTTP serialization and status codes, as well as request parsing and validation.
4. **Error mapping**: Map domain errors returned from services to HTTP status codes via `constants/errors.go`.
5. **Context propagation**: Pass request context to services

## Pattern

Handlers are the HTTP boundary:

- Define handler struct with service dependency. If a handler needs multiple unrelated services, create a use case struct instead. For example, a `RegisterHandler` needing both `UserService` and `EmailService` should delegate to a `RegisterUseCase` that imports both interfaces, keeping the handler thin and focused on HTTP concerns.
- Implement `Handle` method with `http.HandlerFunc` type.
- Parse and validate request. Always include a `Trim()` method on request structs to clean up input but do not trim sensitive fields like passwords, tokens, etc. Before trimming fields on the request struct, check that pointer fields are not nil to avoid dereferencing nil pointers. For example, if you have a request struct with a pointer field like `Email *string`, you should check if `Email` is not nil before calling `strings.TrimSpace()` on it within the `Trim()` method. This ensures that you don't encounter a runtime panic due to dereferencing a nil pointer.
- Invoke use case/service
- Map response to HTTP (status code, headers, JSON) using `reqCtx.SetJSONResponse`.
- Handle errors consistently
- Always make use of the code within the `internal/` folder as there are utilities for request parsing, response formatting, and error handling. Avoid reinventing the wheel by utilising these utilities to ensure consistency across handlers.

## Example

See [examples/todo_handlers.go](examples/todo_handlers.go) for:

- CreateTodoHandler parsing POST requests
- MarkTodoCompleteHandler patterns
- Error handling and response formatting

## Common mistakes

1. Putting business logic in handlers
2. Handlers calling multiple services directly (use use cases instead)
3. Not propagating request context to use cases
4. Forgetting error handling
5. Incorrect HTTP status codes and not using `reqCtx.SetJSONResponse` for consistent response formatting or not using it to return the response at all, which can lead to inconsistent responses and missing status codes. Always use `reqCtx.SetJSONResponse` to ensure that responses are consistently formatted and include the appropriate status codes.
6. Logging sensitive data (passwords, tokens)

## References

- [plugins/email-password/handlers/](../../../plugins/email-password/handlers/) - Handler examples
- [internal/handlers/](../../../internal/handlers/) - Core handler examples
- [internal/router/](../../../internal/router/) - HTTP utilities
