---
name: service-registry
description: Register and retrieve services at runtime using a thread-safe service registry pattern for loose coupling between plugins.
---

# Plugin Service Registry

## When to use this skill

- Register services created by plugins for other plugins to discover
- Retrieve services from the registry in plugin initialization
- Enable loose coupling between plugins via service lookup
- Access core services (User, Account, Session, etc.) in plugins

## Key principles

1. **Lazy discovery**: Services retrieved when needed, not all upfront
2. **Thread-safe**: RWMutex allows concurrent reads, exclusive writes
3. **Type-safe**: Use type assertions to convert `any` back to specific interfaces
4. **Named services**: Service IDs are constants from [models/services.go](../../../models/services.go)

## Pattern

Services are registered in plugin `Init()` methods using the registry interface:

```
// Get a core service
service, ok := ctx.ServiceRegistry.Get(models.ServiceXxx.String()).(ServiceInterface)
if !ok {
    return errors.New("service not available")
}

// Register your plugin's service
ctx.ServiceRegistry.Register(models.ServicePlugin.String(), myService)
```

Always check the `ok` flag on type assertions. Store retrieved services in plugin struct for internal use.

## Service initialization order

1. Core repositories created (all depend on database)
2. Core services created and registered
3. Plugins initialize in order (retrieve core services)
4. Plugin services registered as created
5. Routes registered after plugins

See [bootstrap.go](../../../bootstrap.go) for implementation.

## Code references

- **Registry interface**: [models/plugin.go](../../../models/plugin.go)
- **Registry implementation**: [internal/plugins/service_registry.go](../../../internal/plugins/service_registry.go)
- **Service IDs**: [models/services.go](../../../models/services.go)
- **Example usage**: [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go)

## Common mistakes

1. Not checking `ok` flag on type assertions
2. Using string literals instead of `models.ServiceXxx` constants
3. Registering before implementation complete
4. Creating circular dependencies between plugins
5. Assuming services are available in wrong initialization phase

## Related skills

- [plugin-architecture](../plugin-architecture) - Plugin lifecycle
- [dependency-injection](../dependency-injection) - Wiring core services
- [services-and-interfaces](../services-and-interfaces) - Service patterns
