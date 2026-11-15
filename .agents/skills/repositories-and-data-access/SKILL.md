---
name: repositories-and-data-access
description: Implement repository interfaces for data persistence and abstraction over database operations using Bun ORM.
---

# Repositories & Data Access

## When to use this skill

- Create data access layer abstractions for entities
- Implement Bun-based repositories for database operations
- Define repository interfaces that services depend on
- Add transaction support (WithTx) to repositories

## Key principles

1. **Interface segregation**: One repository interface per domain model
2. **Bun ORM**: Use Bun for type-safe database operations
3. **Transaction support**: Every repository implements `WithTx(tx bun.IDB)`
4. **Context-aware**: All operations accept `context.Context`
5. **No business logic**: Repositories only handle CRUD; logic lives in services

## Pattern

Repositories are data access only:

- Define interface with CRUD operations (Create, GetByID, Update, Delete)
- Implement with Bun in `bun_*.go` files
- Accept `bun.IDB` at construction (not concrete `*bun.DB`)
- Return interfaces from constructors
- Single-record getters return an exported sentinel error when no row matches (for the core session and verification repositories, `coreerrors.ErrSessionNotFound` and `coreerrors.ErrVerificationNotFound`). Do not return `(nil, nil)`. Batch queries return an empty slice.
- Return `nil, err` for database failures; never return a zero-value record alongside an error.

## Example

See [examples/bun_todo_repository.go](examples/bun_todo_repository.go) for:

- ITodoRepository interface definition
- bunTodoRepository Bun implementation
- CRUD and batch query patterns

## Common mistakes

1. Returning `(nil, nil)` for a missing record and forcing every caller to nil-check
2. Forgetting `WithTx()` method
3. Putting business logic in repositories
4. Not passing context to Scan/Exec
5. Repository-to-repository calls instead of service orchestration
6. Returning a non-nil zero-value record together with an error

## References

- [core/repositories/interfaces.go](../../../core/repositories/interfaces.go) - Core repository interfaces
- [core/repositories/bun_user_repository.go](../../../core/repositories/bun_user_repository.go)
- [core/repositories/bun_account_repository.go](../../../core/repositories/bun_account_repository.go)
