---
name: use-cases-and-orchestration
description: Orchestrate services and repositories through use cases to implement application-level workflows and business scenarios.
---

# Use Cases & Orchestration

## When to use this skill

- Implement workflows that span multiple services
- Encapsulate complex authentication or account operations
- Define reusable operations that handlers invoke
- Keep handlers thin by moving logic into use cases

## Key principles

1. **Service orchestration**: Use cases call multiple services
2. **Business-focused**: Methods represent user actions, not HTTP operations
3. **Dependency injection**: Services passed at construction
4. **Pure domain**: No HTTP, no routing, just business logic
5. **Testable**: Can be tested independently by mocking services

## Pattern

Use cases orchestrate services:

- Define request/response types
- Implement as structs with service dependencies
- Single public Execute or action method
- Handle validation such as checking request parameters
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
