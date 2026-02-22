---
name: dependency-injection
description: Wire dependencies using constructor-based dependency injection throughout services, repositories, and handlers.
---

# Dependency Injection & Construction

## When to use this skill

Use this skill when you need to:
- Wire services, repositories, and handlers with their dependencies
- Initialize core services during application bootstrap
- Pass dependencies to plugins during initialization
- Create dependency graphs that are testable and mockable
- Avoid service locators and global state

## Pattern overview

GoBetterAuth uses constructor-based dependency injection exclusively. This means:
- Dependencies are explicit and passed via constructor functions
- No global service locators or "get instance" patterns
- All dependencies are resolved at bootstrap time in [bootstrap.go](../../../bootstrap.go)
- Services receive only what they need to operate
- Easy to mock dependencies for testing
- Type-safe (no runtime reflection or string-based lookups)

### Key principles

1. **Constructor-based**: All dependencies passed via function parameters
2. **Explicit over implicit**: Dependencies are visible in constructor signatures
3. **Interface-based**: Depend on interfaces, not concrete implementations
4. **Single responsibility**: Each constructor handles one entity's wiring
5. **Bootstrap concentrated**: Wiring happens in `bootstrap.go` and plugin initialization

## Implementation checklist

### Step 1: Define constructor with all dependencies

For a service:

```
func NewYourService(
    repo repositories.YourRepository,
    otherService services.OtherService,
    config *models.Config,
    dbHooks *models.CoreDatabaseHooksConfig,
) services.YourService {
    return &yourService{
        repo:    repo,
        other:   otherService,
        config:  config,
        dbHooks: dbHooks,
    }
}
```

Dependency ordering convention:
1. Repositories first
2. Other services
3. Configuration and hooks
4. Specialized dependencies (signers, loggers)

**Reference**: See [internal/services/account_service.go](../../../internal/services/account_service.go) constructor pattern

### Step 2: Create repository instances with database connection

In bootstrap:

```
userRepo := internalrepositories.NewBunUserRepository(db)
accountRepo := internalrepositories.NewBunAccountRepository(db)
```

- All repositories share the same `bun.IDB` connection
- Repositories are created before services that depend on them
- Never create repository instances lazily; instantiate all upfront

**Reference**: See [bootstrap.go](../../../bootstrap.go) for repository initialization

### Step 3: Wire services with their dependencies

```
userService := internalservices.NewUserService(userRepo, config.CoreDatabaseHooks)
accountService := internalservices.NewAccountService(
    config, 
    accountRepo, 
    tokenRepo, 
    config.CoreDatabaseHooks,
)
```

- Create services after all their dependencies exist
- Pass dependencies in the exact order the constructor expects
- Register services with the ServiceRegistry after creation

**Reference**: See [bootstrap.go](../../../bootstrap.go) for service wiring

### Step 4: Register services with ServiceRegistry

After creating a service, register it:

```
serviceRegistry.Register(models.ServiceUser.String(), userService)
serviceRegistry.Register(models.ServiceAccount.String(), accountService)
```

- Use the service ID constant (defined in [models/services.go](../../../models/services.go))
- String representation ensures consistent naming
- Registration happens immediately after service creation

**Reference**: See [bootstrap.go](../../../bootstrap.go) for registration pattern

### Step 5: Inject dependencies into use cases

Use cases depend on services:

```
func BuildUseCases(
    logger models.Logger, 
    userService services.UserService, 
    sessionService services.SessionService,
) *usecases.UseCases {
    return &usecases.UseCases{
        GetMeUseCase: &usecases.GetMeUseCase{
            Logger:         logger,
            UserService:    userService,
            SessionService: sessionService,
        },
    }
}
```

- Pass services as constructor parameters
- Assign them to the use case struct
- Use cases should never instantiate services

**Reference**: See [internal/api.go](../../../internal/api.go) for use case wiring

### Step 6: Plugin service initialization

In plugin's `Init` method:

```
func (p *MyPlugin) Init(ctx *models.PluginContext) error {
    // Retrieve service from registry with type assertion
    sessionService, ok := ctx.ServiceRegistry.Get(
        models.ServiceSession.String(),
    ).(services.SessionService)
    if !ok {
        return errors.New("session service not found")
    }
    p.sessionService = sessionService
    
    // Create plugin's own service and register it
    myService := mynewservice.NewMyService(sessionService)
    ctx.ServiceRegistry.Register(models.ServiceMyPlugin.String(), myService)
    
    return nil
}
```

- Use `ctx.ServiceRegistry.Get()` to retrieve core or other plugin services
- Type assert to convert `any` to specific interface
- Always check the ok flag to ensure service exists
- Register plugin services after initialization

**Reference**: See [plugins/jwt/plugin.go](../../../plugins/jwt/plugin.go) for plugin service wiring pattern

## Code file references

Study these files for DI patterns:

- **Bootstrap and core service wiring**: [bootstrap.go](../../../bootstrap.go) - Main DI entry point
- **Core service initialization**: [bootstrap.go](../../../bootstrap.go)
- **Service ID constants**: [models/services.go](../../../models/services.go)
- **Use case wiring**: [internal/api.go](../../../internal/api.go)
- **Service registry interface**: [models/plugin.go](../../../models/plugin.go)
- **Service registry implementation**: [internal/plugins/service_registry.go](../../../internal/plugins/service_registry.go)

### Plugin examples

- **Plugin service retrieval**: [plugins/jwt/plugin.go](../../../plugins/jwt/plugin.go)
- **Plugin factory pattern**: [internal/bootstrap/plugin_factory.go](../../../internal/bootstrap/plugin_factory.go) - Shows how plugins are instantiated with typed configs

## Common mistakes to avoid

1. **Circular dependencies**: Avoid service A depending on service B that depends on service A. Consider refactoring to use a third service
2. **Lazy initialization**: Don't create services when needed; wire everything at bootstrap time
3. **Depending on concrete types**: Always depend on interfaces, not concrete structs
4. **Registering with wrong key**: Use the constant from [models/services.go](../../../models/services.go), not string literals
5. **Not checking type assertions**: Always verify `ok` when retrieving services from registry
6. **Creating service instances multiple times**: Wire each service once and reuse the instance
7. **Service retrieval without type assertion**: Always type assert when retrieving services from the registry

## Testing with DI

For unit tests, pass mock implementations:

```
mockUserRepo := &MockUserRepository{...}
mockDbHooks := &models.CoreDatabaseHooksConfig{}

userService := services.NewUserService(mockUserRepo, mockDbHooks)
// Now test userService with mock repository
```

See [testing-with-mocks](../testing-with-mocks) skill for mock patterns.
