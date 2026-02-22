---
name: repositories-and-data-access
description: Implement repository interfaces for data persistence and abstraction over database operations using Bun ORM.
---

# Repositories & Data Access

## When to use this skill

Use this skill when you need to:
- Create data access layer abstractions for entities (User, Account, Session, etc.)
- Implement Bun-based repositories for database operations
- Define repository interfaces that services depend on
- Add transaction support (WithTx) to repositories
- Implement custom query logic that encapsulates database details

## Pattern overview

Repositories provide a clean abstraction between services and the database. They:
- Define public interfaces in `repositories/interfaces.go`
- Implement operations using Bun ORM in `repositories/bun_*.go` files
- Accept `bun.IDB` database connection at construction
- Support transactions via the `WithTx()` method
- Handle query building, error mapping, and result transformation

### Key principles

1. **Interface segregation**: Each repository interface represents one domain model
2. **Bun ORM usage**: Use Bun for type-safe, performant database operations
3. **Transaction support**: Every repository must support `WithTx(tx bun.IDB)` for transaction propagation
4. **Context-aware**: All operations accept `context.Context`
5. **Error handling**: Map database errors to domain errors appropriately
6. **No business logic**: Repositories handle data access only; logic lives in services

## Implementation checklist

### Step 1: Define the repository interface

In `repositories/interfaces.go`:

```
type YourRepository interface {
    GetByID(ctx context.Context, id string) (*models.YourEntity, error)
    Create(ctx context.Context, entity *models.YourEntity) (*models.YourEntity, error)
    Update(ctx context.Context, entity *models.YourEntity) (*models.YourEntity, error)
    Delete(ctx context.Context, id string) error
    WithTx(tx bun.IDB) YourRepository
}
```

- Methods reflect domain operations (Get, Create, Update, Delete, etc.)
- Every method accepts `context.Context` as the first parameter
- All operations return `(*Model, error)` or `error`
- Include `WithTx(tx bun.IDB)` to support transactions
- Batch operations use slices: `GetByUserID(ctx context.Context, userID string) ([]Model, error)`

**Reference**: See [internal/repositories/interfaces.go](../../../internal/repositories/interfaces.go) for all core repository interfaces

### Step 2: Create the Bun implementation

In `repositories/bun_your_repository.go`:

```
type bunYourRepository struct {
    db bun.IDB
}

func NewBunYourRepository(db bun.IDB) YourRepository {
    return &bunYourRepository{db: db}
}
```

- Struct name follows pattern: `bun<PascalCaseModel>Repository`
- Accept `bun.IDB` interface (not concrete `*bun.DB`)
- Return the interface type from constructor

**Reference**: See [internal/repositories/bun_user_repository.go](../../../internal/repositories/bun_user_repository.go)

### Step 3: Implement basic CRUD operations

```
func (r *bunYourRepository) GetByID(ctx context.Context, id string) (*models.YourEntity, error) {
    entity := &models.YourEntity{}
    err := r.db.NewSelect().Model(entity).Where("id = ?", id).Scan(ctx)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil // Domain convention: missing records return nil, not error
        }
        return nil, err
    }
    return entity, nil
}

func (r *bunYourRepository) Create(ctx context.Context, entity *models.YourEntity) (*models.YourEntity, error) {
    _, err := r.db.NewInsert().Model(entity).Exec(ctx)
    return entity, err
}

func (r *bunYourRepository) Update(ctx context.Context, entity *models.YourEntity) (*models.YourEntity, error) {
    _, err := r.db.NewUpdate().Model(entity).Where("id = ?", entity.ID).Exec(ctx)
    return entity, err
}
```

- Use `r.db.NewSelect()`, `r.db.NewInsert()`, `r.db.NewUpdate()` for operations
- Pass `ctx` to `Scan()` or `Exec()` calls
- Missing records return `nil, nil` (not `ErrNotFound`)
- Modified entities are returned for method chaining (optional but recommended)

**Reference**: See [internal/repositories/bun_account_repository.go](../../../internal/repositories/bun_account_repository.go) for Create, Update, and custom query patterns

### Step 4: Implement the WithTx method

Every repository must support transactions:

```
func (r *bunYourRepository) WithTx(tx bun.IDB) YourRepository {
    return &bunYourRepository{db: tx}
}
```

- Replace the database connection with the transaction
- Return a new instance with the transaction as `db`
- This allows callers to pass transactions for atomicity

**Reference**: See [internal/repositories/bun_user_repository.go](../../../internal/repositories/bun_user_repository.go)

### Step 5: Implement specialized query methods

Add domain-specific queries:

```
func (r *bunYourRepository) GetByUserID(ctx context.Context, userID string) ([]models.YourEntity, error) {
    var entities []models.YourEntity
    err := r.db.NewSelect().Model(&entities).Where("user_id = ?", userID).Scan(ctx)
    return entities, err
}
```

- Use method names that clearly indicate what is being queried
- Return slices for multi-result queries
- Handle `sql.ErrNoRows` appropriately (nil for single results, empty slice for multi results)

**Reference**: See [internal/repositories/bun_session_repository.go](../../../internal/repositories/bun_session_repository.go) for Get, GetByUserID, and batch operation patterns

## Code file references

Study these implementations:

- **Interface definitions**: [internal/repositories/interfaces.go](../../../internal/repositories/interfaces.go)
- **Implementation examples**:
  - [internal/repositories/bun_user_repository.go](../../../internal/repositories/bun_user_repository.go)
  - [internal/repositories/bun_account_repository.go](../../../internal/repositories/bun_account_repository.go)
  - [internal/repositories/bun_session_repository.go](../../../internal/repositories/bun_session_repository.go)
  - [internal/repositories/bun_verification_repository.go](../../../internal/repositories/bun_verification_repository.go)
- **Special case - token repository**: [internal/repositories/crypto_token_repository.go](../../../internal/repositories/crypto_token_repository.go) - Non-Bun implementation for in-memory operations

### Plugin repositories

Plugins also define repositories following the same pattern:

- **Plugin interface**: [plugins/rate-limit/interfaces.go](../../../plugins/rate-limit/interfaces.go)
- **Plugin interface**: [plugins/config-manager/repositories/interfaces.go](../../../plugins/config-manager/repositories/interfaces.go)

## Common mistakes to avoid

1. **Returning errors for missing records**: Use `nil, nil` for single-record queries that find nothing
2. **Forgetting the `WithTx` method**: Every repository requires transaction support
3. **Using concrete `*bun.DB` instead of interface**: Always accept `bun.IDB` for flexibility
4. **Putting business logic in repositories**: Repositories are data access only; logic lives in services
5. **Not passing context properly**: Always pass `ctx` to `Scan()` and `Exec()` calls
6. **Ignoring error types**: Handle `sql.ErrNoRows` distinctly; don't expose it as a service-level error
7. **Repository-to-repository calls**: Repositories should never call other repositories; orchestration is the service's job
