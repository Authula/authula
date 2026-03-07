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

1. **Thin handlers**: Business logic lives in use cases, not handlers
2. **Use case coordination**: Handlers invoke use cases, not services
3. **HTTP boundaries**: Handle only HTTP serialization and status codes
4. **Error mapping**: Map domain errors returned from services to HTTP status codes via `constants/errors.go`.
5. **Context propagation**: Pass request context to use cases

## Pattern

Handlers are the HTTP boundary:

- Define handler struct with use case dependency
- Implement Handle method accepting http.ResponseWriter and \*http.Request
- Parse and validate request
- Invoke use case
- Map response to HTTP (status code, headers, JSON)
- Handle errors consistently

## Example

See [examples/todo_handlers.go](examples/todo_handlers.go) for:

- CreateTodoHandler parsing POST requests
- MarkTodoCompleteHandler patterns
- Error handling and response formatting

## Common mistakes

1. Putting business logic in handlers
2. Handlers calling services directly (use use cases)
3. Not propagating request context to use cases
4. Forgetting error handling
5. Incorrect HTTP status codes
6. Logging sensitive data (passwords, tokens)

## References

- [plugins/email-password/handlers/](../../../plugins/email-password/handlers/) - Handler examples
- [internal/handlers/](../../../internal/handlers/) - Core handler examples
- [internal/router/](../../../internal/router/) - HTTP utilities
