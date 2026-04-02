---
name: dependency-injection
description: Wire dependencies using constructor-based dependency injection throughout services, repositories, and handlers.
---

# Dependency Injection & Construction

## When to use this skill

- Wire services, repositories, and handlers with their dependencies
- Initialize core services during application bootstrap
- Pass dependencies to plugins during initialization
- Create dependency graphs that are testable and mockable
- Avoid service locators and global state

## Key principles

1. **Constructor-based**: Dependencies passed via function parameters
2. **Explicit over implicit**: Dependencies visible in constructor signatures
3. **Interface-based**: Depend on interfaces, not concrete implementations
4. **Single responsibility**: Each constructor handles one entity's wiring
5. **Bootstrap concentrated**: Wiring happens in bootstrap.go and plugin Init

## Pattern

Dependency injection flow:
- Repositories accept bun.IDB and return interface
- Services accept repositories and return interface
- Use cases accept services and return struct
- Handlers accept use cases and return struct
- Plugins accept PluginContext, retrieve services, create dependencies
- No global state or service locators

## Example

See [examples/di_todo_example.go](examples/di_todo_example.go) for:
- BuildTodosPlugin demonstrating complete DI chain
- Bottom-up wiring (Database ??? Repository ??? Service ??? UseCase ??? Handler)
- Dependency graph diagram
- Unidirectional dependency flow

## Common mistakes

1. Circular dependencies (Service A ??? Service B ??? Service A)
2. Lazy initialization instead of bootstrap-time
3. Depending on concrete types instead of interfaces
4. Creating service instances multiple times
5. Service retrieval without type assertion
6. Not checking ok flag on type assertion

## References

- [bootstrap.go](../../../bootstrap.go) - Main DI entry point
- [models/services.go](../../../models/services.go) - Service ID constants
- [internal/api.go](../../../internal/api.go) - Use case wiring
- [internal/plugins/service_registry.go](../../../internal/plugins/service_registry.go) - Service registry implementation
