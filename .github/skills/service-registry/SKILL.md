---
name: service-registry
description: Register and retrieve services at runtime using a thread-safe service registry pattern for loose coupling between plugins.
---

# Plugin Service Registry

## When to use this skill

Use this skill when you need to:
- Register services created by plugins for other plugins to discover
- Retrieve services from the registry in plugin initialization
- Enable loose coupling between plugins via service lookup
- Access core services (User, Account, Session, etc.) in plugins
- Ensure thread-safe service access during concurrent requests

## Pattern overview

The service registry is a runtime lookup mechanism that:
- Maintains a map of service name to service interface
- Is thread-safe (uses RWMutex for concurrent access)
- Is populated during bootstrap and plugin initialization
- Allows plugins to discover and use each other's services
- Uses string keys and `any` values with type assertions

### Key principles

1. **Lazy discovery**: Services are retrieved when needed, not all upfront
2. **Thread-safe**: Multiple goroutines can safely read/write to registry
3. **Type-safe**: Type assertions convert `any` back to specific interfaces
4. **Central coordination**: Registry prevents hardcoded dependencies between plugins
5. **Named services**: Service IDs are constants defined in [models/services.go](../../../models/services.go)

## Implementation checklist

### Step 1: Understand the registry interface

In [models/plugin.go](../../../models/plugin.go):

```
type ServiceRegistry interface {
    Register(name string, service any)
    Get(name string) any
}
```

- `Register()` adds a service to the registry
- `Get()` retrieves a service (returns nil if not found)
- Services are stored as `any`; callers type assert

The registry is contained in `PluginContext`:

```
type PluginContext struct {
    Logger          Logger
    EventBus        EventBus
    ServiceRegistry ServiceRegistry
    GetConfig       func() *Config
}
```

**Reference**: See [models/plugin.go](../../../models/plugin.go) for interface definition

### Step 2: Register services in plugin's Init method

```
func (p *MyPlugin) Init(ctx *models.PluginContext) error {
    // Register services
    someService := NewSomeService(/*...dependencies...*/)
    ctx.ServiceRegistry.Register(models.SomeService.String(), someService)
}
```

- Register each service immediately after creation
- Use `models.ServiceXxx.String()` as the key (defined in [models/services.go](../../../models/services.go))

### Step 3: Define service ID constants

In [models/services.go](../../../models/services.go):

```
const (
    // Existing Plugins...
    ServiceSomeService     ServiceID = "some_service"
)
```

- Each service has a unique constant
- Call `.String()` to get the string key for registration
- Add new constants when adding new services

**Reference**: See [models/services.go](../../../models/services.go) for current services

### Step 4: Retrieve services in plugin initialization

In plugin's `Init()` method:

```
func (p *MyPlugin) Init(ctx *models.PluginContext) error {
    p.logger = ctx.Logger
    
    // Retrieve with type assertion
    sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(services.SessionService)
    if !ok {
        p.logger.Error("Session service not found in registry")
        return errors.New("session service not available")
    }
    p.sessionService = sessionService
    
    // Retrieve multiple services as needed
    userService, ok := ctx.ServiceRegistry.Get(models.ServiceUser.String()).(services.UserService)
    if !ok {
        return errors.New("user service not available")
    }
    p.userService = userService
    
    return nil
}
```

- Use `ctx.ServiceRegistry.Get(key)` to retrieve
- Type assert immediately: `.(ServiceInterface)`
- Always check `ok` flag; service might not be registered yet
- Store retrieved services in plugin struct for use during requests

**Reference**: See [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go) for retrieval pattern

### Step 5: Register plugin services

After initializing plugin services, register them:

```
func (p *MyPlugin) Init(ctx *models.PluginContext) error {
    // ... retrieve core services ...
    
    // Create plugin service
    myService := services.NewMyService(p.sessionService, p.userService)
    p.myService = myService
    
    // Register plugin service for other plugins
    ctx.ServiceRegistry.Register(
        models.ServiceMyPlugin.String(),
        myService,
    )
    
    return nil
}
```

- Store the service in a plugin field for handler/use case injection
- Also register it for other plugins to discover
- Registration happens after service creation

**Reference**: See [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go) for registration within plugin

### Step 6: Use registered services in handlers/use cases

Create handlers and use cases with registered services:

```
func (p *MyPlugin) Routes() []models.Route {
    // Services are already stored in p during Init()
    // Use them directly
    handler := handlers.NewMyHandler(p.myService, p.logger)
    
    return []models.Route{
        {
            Method:  http.MethodPost,
            Path:    "/my-endpoint",
            Handler: handler.Handle,
        },
    }
}
```

- Services are available as plugin struct fields after `Init()`
- Inject them into handlers and use cases normally
- No need to query registry again

**Reference**: See plugin route definitions like [plugins/email-password/api.go](../../../plugins/email-password/api.go)

## Code file references

Study these implementations:

- **Registry interface**: [models/plugin.go](../../../models/plugin.go)
- **Registry implementation**: [internal/plugins/service_registry.go](../../../internal/plugins/service_registry.go)
- **Service ID constants**: [models/services.go](../../../models/services.go)
- **Core service registration**: [bootstrap.go](../../../bootstrap.go)
- **Plugin service retrieval**: [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go)
- **Plugin context**: [models/plugin.go](../../../models/plugin.go)

## Thread safety

- The registry is thread-safe using `sync.RWMutex`
- Multiple goroutines can read services concurrently
- `RWMutex` allows multiple concurrent readers
- Write access (Register) is exclusive
- Get is safe for concurrent access

**Reference**: See [internal/plugins/service_registry.go](../../../internal/plugins/service_registry.go)

## Service initialization order

Services must be initialized in dependency order:

1. **Core repositories** are created first (all depend on database)
2. **Core services** are created and registered (User, Account, Session, Verification etc.)
3. **Plugins initialize in order**, each retrieving core services
4. **Plugin services** are registered as they're created
5. **Routes** are registered after plugins initialize

This ensures that when a plugin's `Init()` is called, all core services are available.

**Reference**: See [bootstrap.go](../../../bootstrap.go) for initialization sequence

## Common mistakes to avoid

1. **Not checking ok flag**: Always verify type assertion succeeded: `service, ok := registry.Get(...).(Type)`
2. **Using string literals for service names**: Always use constants from `models.ServiceXxx`
3. **Registering with wrong type**: Ensure registered service implements the expected interface
4. **Forgetting to register plugin services**: If other plugins need your service, register it
5. **Service not found in registry**: Ensure registering plugin runs before retrieving plugin
6. **Race conditions in Get/Register**: Avoid reading registry during concurrent initialization (but normal operation is safe)
7. **Type assertions with unchecked interfaces**: Always capture and check `ok` when asserting types
8. **Circular dependencies**: Plugin A depends on B, B depends on A—avoid this via service design

## Related skills

- See [plugin-architecture](../plugin-architecture) for full plugin lifecycle
- See [dependency-injection](../dependency-injection) for how core services are wired
- See [services-and-interfaces](../services-and-interfaces) for service patterns
