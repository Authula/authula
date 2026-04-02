---
name: plugin-architecture
description: Build pluggable authentication features using the plugin system with initialization, migrations, routes, and service registration.
---

# Plugin Architecture & Lifecycle

## When to use this skill

- Create new authentication plugins (OAuth2, Magic Link, JWT, etc.)
- Extend authentication with custom features
- Define plugin configuration and metadata
- Handle plugin initialization, migrations, and lifecycle
- Provide HTTP routes through a plugin
- Hook into core authentication events

## Key principles

1. **Interface-driven**: Plugins implement optional interfaces
2. **Composition**: Dependencies via context, not inheritance
3. **Self-contained**: Each plugin has its own services, repositories, handlers
4. **Lazy initialization**: Heavy operations in Init(), not constructors
5. **Event-driven**: Plugins hook into core events
6. **Service registry**: Plugins register services for other plugins

## Pattern

Plugin lifecycle:
- Define metadata (ID, name, version)
- Implement Init to retrieve services, create repositories/services
- Implement optional Routes, Migrations, Middleware
- Register custom services with registry
- Implement Close for cleanup

## Example

See [examples/todo_plugin.go](examples/todo_plugin.go) for:
- TodosPlugin metadata and Init pattern
- Service retrieval and registration
- Routes definition and handler wiring
- Lifecycle management (Close)

## Common mistakes

1. Initializing in constructor instead of Init()
2. Not retrieving services from registry (use Get with type assertion)
3. Forgetting type assertions on service retrieval
4. Registering services with wrong names
5. Not implementing Close()
6. Direct plugin-to-plugin dependencies instead of service registry
7. Not handling initialization errors

## References

- [models/plugin.go](../../../models/plugin.go) - Plugin interface definitions
- [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go) - Email-password plugin
- [plugins/jwt/plugin.go](../../../plugins/jwt/plugin.go) - JWT plugin
- [internal/bootstrap/plugin_factory.go](../../../internal/bootstrap/plugin_factory.go) - Plugin factory
