---
name: services-and-interfaces
description: Define and implement service interfaces that encapsulate business logic with proper constructor-based dependency injection.
---

# Services & Interface Design

## When to use this skill

Use this skill when you need to:
- Create a new service that encapsulates business logic (e.g., UserService, AccountService)
- Define reusable service contracts across the application
- Implement business operations that delegate to repositories for data access
- Add a new authentication or domain-specific feature as a service

## Pattern overview

Services are the heart of GoBetterAuth's domain logic. They:
- Define public interfaces in a `services/interfaces.go` file
- Implement logic in concrete structs (unexported, e.g., `userService`)
- Accept dependencies via constructor functions (e.g., `NewUserService()`)
- Delegate data operations to repositories
- Handle cross-cutting concerns like database hooks and error handling

### Key principles

1. **Interface-first design**: Service contracts are defined as interfaces, allowing easy mocking and substitution
2. **Constructor-based injection**: All dependencies passed at construction time
3. **Single responsibility**: Each service handles one domain concern (users, accounts, sessions, etc.)
4. **Repository delegation**: Services never query the database directly; they use repositories
5. **Context-aware**: All public methods accept `context.Context` as first parameter

## Implementation checklist

### Step 1: Define the interface

Create or update `services/interfaces.go` with your service interface:

```
type YourService interface {
    DoSomething(ctx context.Context, param string) (*Result, error)
    GetByID(ctx context.Context, id string) (*Model, error)
}
```

- Method names should start with verbs (Create, Get, Update, Delete, etc.)
- Every method takes `context.Context` as the first parameter after the receiver
- Return types should be `(*Model, error)` or `([]Model, error)` or just `error`

**Reference**: See [services/core.go](../../../services/core.go) for interfaces like `UserService`, `AccountService`, `SessionService`

### Step 2: Create the implementation struct

In a new file (e.g., `services/your_service.go`):

```
type yourService struct {
    repo    repositories.YourRepository
    signer  security.TokenSigner
    dbHooks *models.CoreDatabaseHooksConfig
}
```

- Struct name is unexported (lowercase)
- Include all dependencies as private fields
- Common dependencies: repositories, signers, loggers, config, database hooks

**Reference**: See [internal/services/user_service.go](../../../internal/services/user_service.go) for example

### Step 3: Implement the constructor

The constructor function signature follows a consistent pattern:

```
func NewYourService(
    repo repositories.YourRepository,
    signer security.TokenSigner,
    dbHooks *models.CoreDatabaseHooksConfig,
) services.YourService {
    return &yourService{
        repo:    repo,
        signer:  signer,
        dbHooks: dbHooks,
    }
}
```

- Function name is `New` + interface name (e.g., `NewUserService`)
- Accept all dependencies as parameters
- Return the interface type, not the concrete struct
- Dependencies are listed in a consistent order: repositories first, then specialized dependencies, then config

**Reference**: See [internal/services/account_service.go](../../../internal/services/account_service.go#L21) constructor

### Step 4: Implement methods

Each method delegates to repositories and handles cross-cutting concerns:

```
func (s *yourService) DoSomething(ctx context.Context, param string) (*Model, error) {
    // Fetch from repository
    item, err := s.repo.GetByID(ctx, param)
    if err != nil {
        return nil, err
    }
    
    // Apply business logic
    item.Name = processName(param)
    
    // Update via repository
    return s.repo.Update(ctx, item)
}
```

- Delegate data access to repositories
- Use database hooks for pre/post operation callbacks
- Handle errors appropriately
- Apply business rules, validation, and transformations

**Reference**: See [internal/services/verification_service.go](../../../internal/services/verification_service.go) for patterns

## Code file references

Study these files to understand the service pattern:

- **Interface definitions**: [services/core.go](../../../services/core.go) - Core service interfaces
- **Service implementations**:
  - [internal/services/user_service.go](../../../internal/services/user_service.go)
  - [internal/services/account_service.go](../../../internal/services/account_service.go)
  - [internal/services/session_service.go](../../../internal/services/session_service.go)
  - [internal/services/verification_service.go](../../../internal/services/verification_service.go)
- **Token service pattern**: [internal/services/token_service.go](../../../internal/services/token_service.go) - Simple delegation pattern

### Plugin service patterns

Plugin services follow the same pattern but are scoped to the plugin:

- **Plugin interface**: [plugins/jwt/services/interfaces.go](../../../plugins/jwt/services/interfaces.go)
- **Plugin implementation**: [plugins/jwt/services/jwt_service.go](../../../plugins/jwt/services/jwt_service.go)
- **Secondary storage service**: [plugins/secondary-storage/service.go](../../../plugins/secondary-storage/service.go)

## Common mistakes to avoid

1. **Service-to-service direct calls without interfaces**: Always use interfaces, not concrete types
2. **Putting queries in services**: Let repositories handle all data access; services orchestrate
3. **Missing context parameter**: Every public method must accept `context.Context` as the first parameter
4. **Exporting the implementation struct**: Always keep implementation private (lowercase) and export only the interface
5. **Ignoring database hooks**: Services should respect `dbHooks` for pre/post operation extensions
6. **Not testing with mocks**: Always mock repository dependencies in tests
