---
name: plugin-architecture
description: Build pluggable authentication features using the plugin system with initialization, migrations, routes, and service registration.
---

# Plugin Architecture & Lifecycle

## When to use this skill

Use this skill when you need to:
- Create a new authentication plugin (e.g., OAuth2, Magic Link, JWT)
- Extend authentication with custom features via plugins
- Define plugin configuration and metadata
- Handle plugin initialization, migrations, and lifecycle
- Provide HTTP routes through a plugin
- Hook into core authentication events

## Pattern overview

Plugins are the extensibility mechanism of GoBetterAuth. They:
- Implement the `Plugin` interface for core lifecycle
- Optionally implement `PluginWithRoutes`, `PluginWithMigrations`, `PluginWithMiddleware`
- Are discovered via plugin factory during bootstrap
- Initialize with a `PluginContext` containing core services and event bus
- Register their own services for other plugins to use
- Can provide HTTP routes, database migrations, and middleware

### Key principles

1. **Interface-driven architecture**: Plugins adhere to optional interfaces
2. **Composition over inheritance**: Plugins receive dependencies via context
3. **Self-contained**: Each plugin has its own services, repositories, handlers, use cases
4. **Lazy initialization**: Heavy operations happen in `Init()`, not constructors
5. **Event-driven**: Plugins hook into core events for reactive patterns
6. **Service registry integration**: Plugins register and retrieve services at runtime

## Implementation checklist

### Step 1: Define plugin metadata

In `plugin.go`:

```
type MyPlugin struct {
    logger           models.Logger
    config           MyPluginConfig
    sessionService   services.SessionService
    myService        *services.MyServiceImpl
    // ... other dependencies
}

func (p *MyPlugin) Metadata() models.PluginMetadata {
    return models.PluginMetadata{
        ID:      "myplugin",
        Name:    "My Plugin",
        Version: "1.0.0",
    }
}

func (p *MyPlugin) Config() any {
    return p.config
}

func New(config MyPluginConfig) models.Plugin {
    return &MyPlugin{
        config: config,
    }
}
```

- Define a concrete struct for the plugin
- `Metadata()` returns plugin identification
- `Config()` returns the typed configuration
- Constructor accepts typed config (parsed by factory)

**Reference**: See [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go) or [plugins/jwt/plugin.go](../../../plugins/jwt/plugin.go)

### Step 2: Implement the Init lifecycle method

The `Init()` method wires dependencies and initializes the plugin:

```
func (p *MyPlugin) Init(ctx *models.PluginContext) error {
    // Store logger and event bus
    p.logger = ctx.Logger
    
    // Retrieve core services from registry
    sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(services.SessionService)
    if !ok {
        return errors.New("session service not found")
    }
    p.sessionService = sessionService
    
    // Create plugin-specific services
    p.myPluginService := services.NewMyPluginService(
        myPluginRepo,
        p.sessionService,
    )
    
    // Register plugin services for other plugins to use
    ctx.ServiceRegistry.Register(
        models.ServiceMyPlugin.String(),
        p.myPluginService,
    )
    
    // Hook into events if needed
    p.subscribeToEvents(ctx.EventBus)
    
    return nil
}

func (p *MyPlugin) Close() error {
    // Clean up resources (close connections, cancel subscriptions)
    return nil
}
```

- Retrieve core services from `ctx.ServiceRegistry`
- Always check type assertions with `ok` flag
- Create plugin-specific services
- Register plugin services immediately after creation
- Subscribe to events if the plugin is event-driven

**Reference**: See [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go) for a comprehensive Init pattern

### Step 3: Define plugin configuration

Create `types/types.go` with configuration:

```
type MyPluginConfig struct {
    Enabled bool   `json:"enabled" toml:"enabled"`
    SomeProperty string `json:"some_property" toml:"some_property"`
}
```

- Use struct tags for config parsing (json and toml is required)
- Provide sensible defaults
- Keep config simple and focused

**Reference**: See [plugins/email-password/types/types.go](../../../plugins/email-password/types/types.go) or [plugins/jwt/types/](../../../plugins/jwt/types/)

### Step 4: Implement optional interfaces

If the plugin provides routes:

```
func (p *MyPlugin) Routes() []models.Route {
    someHandler := handlers.NewSomeHandler(p.someUseCase)
    
    return []models.Route{
        {
            Method:  http.MethodPost,
            Path:    "/some-route",
            Handler: someHandler.Handle,
        },
    }
}
```

If the plugin has database migrations:

```
func (p *MyPlugin) Migrations(provider string) []migrations.Migration {
    return []migrations.Migration{
        {
            Name: "001_create_my_plugin_model_table",
            Up: func(db bun.IDB) error {
                _, err := db.NewCreateTable().Model((*models.MyPluginModel)(nil)).Exec(context.Background())
                return err
            },
        },
    }
}

func (p *MyPlugin) DependsOn() []string {
    return []string{} // Names of plugins this depends on
}
```

If the plugin provides middleware:

```
func (p *MyPlugin) Middleware() []func(http.Handler) http.Handler {
    return []func(http.Handler) http.Handler{
        func(next http.Handler) http.Handler {
            return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // Do something before
                next.ServeHTTP(w, r)
                // Do something after
            })
        },
    }
}
```

**Reference**: See [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go) for route provision

### Step 5: Create plugin services, repositories, handlers, use cases

Organize the plugin like a mini-application:

```
plugins/myplugin/
├── plugin.go              # Metadata, Init, lifecycle
├── api.go                 # Routes and use cases builder
├── constants/
│   └── constants.go       # Event names, hook IDs
├── types/
│   └── types.go           # Config and domain types
├── services/
│   ├── interfaces.go      # Service contracts
│   └── my_service.go      # Implementation
├── repositories/
│   ├── interfaces.go
│   └── bun_my_repo.go     # Bun implementation that interacts with the database
├── handlers/
│   ├── my_handler.go      # HTTP handlers for plugin routes
│   └── ...
└── usecases/
    ├── interfaces.go
    └── my_usecase.go      # Business logic that orchestrates repositories and services 
```

- Follow the same patterns as core services/repositories
- Keep everything scoped to the plugin
- Use dependency injection within the plugin

**Reference**: See [plugins/email-password/](../../../plugins/email-password/) structure

### Step 6: Hook into the plugin factory

The plugin is registered with the factory in [internal/bootstrap/plugin_factory.go](../../../internal/bootstrap/plugin_factory.go):

```
{
    ID: models.PluginMyPlugin.String(),
    RequiredByDefault: false,
    ConfigParser: func(rawConfig any) (any, error) {
        config := myplugin.MyPluginConfig{}
        if rawConfig != nil {
            if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
                return nil, fmt.Errorf("failed to parse myplugin config: %w", err)
            }
        }
        return config, nil
    },
    Constructor: func(typedConfig any) models.Plugin {
        return myplugin.New(typedConfig.(myplugin.MyPluginConfig))
    },
},
```

This is handled by the framework; you just ensure your plugin follows the patterns.

## Code file references

Study these implementations:

- **Plugin interface definition**: [models/plugin.go](../../../models/plugin.go)
- **Full plugin example**: [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go)
- **JWT plugin** (simpler example): [plugins/jwt/plugin.go](../../../plugins/jwt/plugin.go)
- **Plugin initialization patterns**:
  - [plugins/email-password/plugin.go](../../../plugins/email-password/plugin.go)
  - [plugins/magic-link/plugin.go](../../../plugins/magic-link/plugin.go)
- **Plugin factory**: [internal/bootstrap/plugin_factory.go](../../../internal/bootstrap/plugin_factory.go)
- **Plugin routes and use cases**: [plugins/email-password/api.go](../../../plugins/email-password/api.go)
- **Service registry**: [internal/plugins/service_registry.go](../../../internal/plugins/service_registry.go)

## Plugin lifecycle

1. **Discovery**: Plugin factory creates plugin instance with typed config
2. **Init**: `Init(ctx *PluginContext)` called; plugin retrieves services and registers own
3. **Route registration**: If `PluginWithRoutes`, routes are mounted
4. **Middleware registration**: If `PluginWithMiddleware`, middleware is added
5. **Migration running**: If `PluginWithMigrations`, migrations are executed in order
6. **Operation**: Plugin handles requests and events
7. **Shutdown**: `Close()` called to release resources

## Common mistakes to avoid

1. **Initializing in constructor**: Heavy initialization belongs in `Init()`, not `New()`
2. **Not retrieving services from registry**: Always use `ctx.ServiceRegistry.Get()`
3. **Forgetting type assertions**: Always check the `ok` flag when retrieving services
4. **Registering services with wrong names**: Use the constant from [models/services.go](../../../models/services.go)
5. **Not implementing Close()**: Always implement cleanup even if it's just returning nil
6. **Exporting implementation types**: Keep implementation structs private; export only interfaces
7. **Plugin-to-plugin direct dependencies**: Use the service registry, not direct imports
8. **Not handling initialization errors**: Return meaningful errors from Init()
9. **Creating services without dependency injection**: All dependencies should be injected

## Related skills

- See [services-and-interfaces](../services-and-interfaces) for service patterns
- See [repositories-and-data-access](../repositories-and-data-access) for data layer
- See [dependency-injection](../dependency-injection) for how plugins are wired
- See [service-registry](../service-registry) for service registration details
