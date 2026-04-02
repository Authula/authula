---
name: services-and-interfaces
description: Define and implement services that encapsulate business logic with proper constructor-based dependency injection.
---

# Services & Interface Design

## When to use this skill

- Create services that encapsulate business logic
- Implement business operations that delegate to repositories
- Add authentication or domain-specific features as services

## Key principles

1. **Concrete struct design**: Services are concrete structs
2. **Constructor-based injection**: Dependencies passed at construction
3. **Single responsibility**: Each service handles one domain concern
4. **Repository delegation**: Services delegate all data access to repositories
5. **Context-aware**: All public methods accept `context.Context`

## Pattern

Services handle business logic:
- Define interface (optional but recommended)
- Implement as concrete struct with repository dependencies
- Constructor function injects all dependencies
- Methods are thin wrappers that validate, transform, and delegate to repositories
- Return interfaces from constructors

## Example

See [examples/todo_service.go](examples/todo_service.go) for:
- TodoService interface definition
- todoService implementation
- Validation and repository delegation patterns

## Common mistakes

1. Service-to-service direct calls without interfaces
2. Putting queries in services (repositories own data access)
3. Missing context parameter
4. Exporting implementation struct instead of interface
5. Not testing with mocks

## References

- [services/core.go](../../../services/core.go) - Core service interfaces
- [internal/services/user_service.go](../../../internal/services/user_service.go)
- [internal/services/account_service.go](../../../internal/services/account_service.go)
