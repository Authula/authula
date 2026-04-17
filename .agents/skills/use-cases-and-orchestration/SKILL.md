---
name: use-cases-and-orchestration
description: Orchestrate services and repositories through use cases to implement application-level workflows and business scenarios.
---

# Use Cases & Orchestration

## When to use this skill

- Implement workflows that span multiple services
- Define reusable operations that handlers invoke
- Keep handlers thin by moving logic into use cases if multiple services are needed/used
- Encapsulate business logic and orchestration in a single place
- Coordinate complex operations that require multiple steps and services

## Key principles

1. **Service orchestration**: Use cases call multiple services
2. **Business-focused**: Methods represent user actions, not HTTP operations
3. **Dependency injection**: Services passed at construction
4. **Pure domain**: No HTTP, no routing, just business logic delegated to services. Usecases simply orchestrate services and handle validation, errors, and return domain models
5. **Testable**: Can be tested independently by mocking services as services are interfaces

## Pattern

Use cases orchestrate services:

- Implement as structs with service dependencies
- Single public Execute or action method e.g. `RegisterUseCase` with `Execute(ctx context.Context, reqCtx *models.RequestContext) (res *someStructType, error)`. The `res` can be a domain model that returns the data from each service's method output, but should not be an HTTP response. The handler can convert it to an HTTP response if needed.
- Handle validation such as checking request parameters passed down from the handler.
- Coordinate multiple services sequentially
- Handle errors and return domain models

## Example

See [examples/todo_usecases.go](examples/todo_usecases.go) for:

- CreateTodoUseCase orchestrating TodoService
- MarkTodoCompleteUseCase patterns
- Request/response types and error handling

## Common mistakes

1. Use cases returning HTTP status codes
2. Validation in handlers instead of use cases
3. Use cases calling other use cases
4. Creating new services instead of injecting
5. Use cases with no clear purpose

## References

- [internal/usecases/](../../../internal/usecases/) - Core use cases
- [plugins/email-password/usecases/](../../../plugins/email-password/usecases/) - Plugin use case examples
- [plugins/email-password/types/](../../../plugins/email-password/types/) - Request/response types
