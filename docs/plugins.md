# Plugin development guide

## 1) Plugin contract — exact required signatures

All plugins MUST implement the base `models.Plugin` interface. Optional interfaces are supported and used by boot/runner components.

Required (exact):

- Metadata():

```go
Metadata() models.PluginMetadata
```

- Config():

```go
Config() any
```

- Init():

```go
Init(ctx *models.PluginContext) error
```

- Close():

```go
Close() error
```

Optional (implement only if required):

- Migrations / dependency (database schema):

```go
Migrations(provider string) []migrations.Migration
DependsOn() []string
```

- Routes (HTTP routes):

```go
Routes() []models.Route
```

- Hooks (request lifecycle hooks):

```go
Hooks() []models.Hook
```

- Middleware provider:

```go
Middleware() []func(http.Handler) http.Handler
```

- Config hot-reload watcher (will be removed in a future version):

```go
OnConfigUpdate(config *models.Config) error
```

Context available in `Init` (exact struct fields):

```go
type PluginContext struct {
  DB              bun.IDB
  Logger          models.Logger
  EventBus        models.EventBus
  ServiceRegistry models.ServiceRegistry
  GetConfig       func() *models.Config
}
```

Source of truth for signatures: `models/plugin.go`.

---

## 2) File-map for `plugins/myplugin` (one-line purpose + key symbols)

Use this exact layout and responsibilities when creating a plugin.

- `plugin.go` — plugin entry point
  - Purpose: plugin struct + `New(...)`, `Metadata()`, `Init()`, `Config()`, `Close()` and optional interface implementations.
  - Key symbols: `type MyPlugin struct { ... }`, `func New(config types.MyPluginConfig) *MyPlugin`, `func (p *MyPlugin) Init(ctx *models.PluginContext) error`.

- `types/types.go` — typed plugin config & domain models
  - Purpose: `PluginConfig`, domain models, DTOs, `ApplyDefaults()` and validation helpers.
  - Key symbols: `MyPluginConfig`, `ApplyDefaults()` method, domain structs like `MyPluginThing`.

- `migrations.go` — provider-aware migrations
  - Purpose: declare provider-specific migrations and return `[]migrations.Migration`.
  - Key symbols: `myPluginMigrationsForProvider(provider string) []migrations.Migration`, migration `Version` strings and `Up/Down` closures.

- `routes.go` — HTTP route wiring
  - Purpose: return `[]models.Route` for plugin routes.
  - Key symbols: `func Routes(plugin *MyPlugin) []models.Route`.

- `hooks.go` — request lifecycle hooks
  - Purpose: register Hook handlers (HookStage, Matcher, Handler, Order, Async).
  - Key symbols: `func (p *MyPlugin) Hooks() []models.Hook`, hook handlers like `exampleHook`.

- `handlers/` — HTTP handlers
  - Purpose: HTTP handler structs that adapt usecases to `http.Handler`.
  - Example files: `some_handler.go`.

- `usecases/` — application logic boundary
  - Purpose: pure business logic, `NewSomeUseCase(...)` constructors.
  - Example files: `some_usecase.go`.

- `services/` — interfaces + implementations
  - Purpose: service interfaces in `interfaces.go`, concrete implementations, constructors `NewSomeService(...)`.
  - Example files: `myplugin_service.go`.

- `repositories/` — data access
  - Purpose: define repository interfaces + Bun-based implementations.
  - Key patterns: `New<Thing>Repository(db bun.IDB) <RepoInterface>`; see `bun_some_repository.go`.

- `constants/` — plugin constants (hook IDs, event names)

- `events/` — plugin-specific event definitions

- `plugin_test.go` — unit tests for config, metadata, helpers, and small integration points

---

## 3) How plugins are instantiated & registered (exact flow)

1. `internal/bootstrap/plugin_factory.go` contains `pluginFactories` array. Add an entry with:

```go
PluginFactory{
  ID: models.PluginMyPlugin.String(),
  ConfigParser: func(rawConfig any) (any, error) { /* parse to typed config */ },
  Constructor: func(typedConfig any) models.Plugin { return myplugin.New(typedConfig.(types.MyPluginConfig)) },
}
```

2. `BuildPluginsFromConfig(config)` instantiates plugins from the ordered `pluginFactories` list.
3. `internal/plugins/plugin_registry.go`'s `Register` + `InitAll()` call each plugin's `Init(ctx *models.PluginContext)` if enabled.

Files to edit when adding a new plugin:

- `internal/bootstrap/plugin_factory.go` (factory entry)
- `config.example.toml` (add plugin config block)

---

## 4) Services & DI (exact patterns)

- Register runtime services from `Init` using the ServiceRegistry:

```go
ctx.ServiceRegistry.Register(models.ServiceMyPlugin.String(), myServiceImpl)
```

- Retrieve services and assign them to fields using:

```go
func (p *MyPlugin) Init(ctx *models.PluginContext) error {
  sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(services.SessionService)
  if !ok {
    p.Logger.Error("service not found")
    return errors.New("service not available")
  }
  p.sessionService = sessionService
}
```

- Keep service interfaces in `services/interfaces.go` and implementations in `services/*.go`.

Reference: `plugins/myplugin/plugin.go` where MyPlugin registers its `myServiceImpl`.

---

## 5) Writing routes & handlers (pattern)

- Provide `Routes(plugin *MyPlugin) []models.Route` returning routes like:

```go
models.Route{
  Path: "/some/path",
  Method: http.MethodPost,

  Handler: someHandler.Handler(),
}
```

- Handlers should expose a `Handler() http.Handler` method.
- Attach route metadata when you need plugin-specific hook execution: `Metadata: map[string]any{"plugins": []string{"myplugin.respond_json"}}`.

Reference: `plugins/myplugin/routes.go` and `models/routes.go`.

---

## 6) Hooks — exact semantics and recommended usage

Hook definition (use these exact fields):

```go
models.Hook{
  Stage models.HookStage,           // HookOnRequest | HookBefore | HookAfter | HookOnResponse
  PluginID string,                  // optional filter
  Matcher models.HookMatcher,       // func(reqCtx *models.RequestContext) bool
  Handler models.HookHandler,       // func(reqCtx *models.RequestContext) error
  Order int,                        // lower executes first
  Async bool,                       // true = run in background
}
```

- Critical security hooks MUST be synchronous (Async=false).
- Use `Matcher` to only run for certain routes or conditions.

Example (pseudocode from MyPlugin): issue tokens at `HookAfter` then respond on `HookOnResponse`.

Reference: `plugins/myplugin/hooks.go`.

---

## 7) Migrations — exact rules & pseudo-code example

- Implement `PluginWithMigrations` when your plugin needs DB schema.
- Provide provider-specific migrations: `Migrations(provider string) []migrations.Migration`.
- Migration struct (exact):

```go
migrations.Migration{
  Version: "YYYYMMDDHHMMSS_myplugin_description",
  Up: func(ctx context.Context, tx bun.Tx) error { /* exec SQL */ },
  Down: func(ctx context.Context, tx bun.Tx) error { /* reverse SQL */ },
}
```

- Use `migrations.ExecStatements(ctx, tx, sql1, sql2, ...)` inside `Up/Down`.
- Follow provider variants pattern (sqlite/postgres/mysql) as in `plugins/myplugin/migrations.go`.
- If your plugin depends on another plugin's schema, return the dependency in `DependsOn()`.

Pseudocode example (initial migration):

```go
func myPluginMigrationsForProvider(provider string) []migrations.Migration {
  return migrations.ForProvider(provider, migrations.ProviderVariants{
    "sqlite": func() []migrations.Migration { return []migrations.Migration{mySQLiteInitial()} },
    "postgres": func() []migrations.Migration { return []migrations.Migration{myPostgresInitial()} },
    "mysql": func() []migrations.Migration { return []migrations.Migration{myMySQLInitial()} },
  })
}

func mySQLiteInitial() migrations.Migration {
  return migrations.Migration{
    Version: "20260131000000_myplugin_initial",
    Up: func(ctx context.Context, tx bun.Tx) error {
      return migrations.ExecStatements(ctx, tx, `CREATE TABLE ...;`, `CREATE INDEX ...;`)
    },
    Down: func(ctx context.Context, tx bun.Tx) error {
      return migrations.ExecStatements(ctx, tx, `DROP TABLE ...;`)
    },
  }
}
```

How migrations are executed:

- `internal/migrationmanager.Manager` collects plugin migration sets and calls `migrator.Migrate()`.
- Use CLI: `go run ./cmd/migrate plugins up --only=<plugin_id>`.

Reference: `plugins/myplugin/migrations.go`, `internal/migrationmanager/manager.go`, `cmd/migrate/main.go`.

---

## 8) Tests — recommended coverage and examples

Follow the MyPlugin testing patterns (fast unit tests + integration-style checks):

Mandatory tests:

- Config defaults and `ApplyDefaults()` behavior (`plugin/types/types.go`).
- `Metadata()` returns expected ID/version/description.
- `Config()` returns typed config.
- `Init()` behavior: registers services and fails cleanly if required services missing.
- `Routes()` & `Handlers()` return expected paths and HTTP behaviour.
- `Migrations(provider)` returns valid `[]migrations.Migration` and proper `Version` strings.

Test helpers to reuse:

- `internal/util/tests.go` contains `mockPlugin` and test helpers for `PluginContext`.

Reference tests: `plugins/myplugin/plugin_test.go`.

---

## 9) Config & hot-reload

- Add plugin config skeleton to `config.example.toml` under `[plugins.<plugin_id>]`.
- Include `enabled = true|false` and plugin-specific fields.
- Use `util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.pluginConfig)` inside `Init`.
- If supporting hot-reload, implement `PluginWithConfigWatcher` and register with `ConfigManagerService` (registry wiring is automatic in `PluginRegistry.registerConfigWatchers`).

---

## 10) PR checklist (pre-merge)

- [ ] `internal/bootstrap/plugin_factory.go` entry added.
- [ ] `config.example.toml` has sample config.
- [ ] `plugins/<your-plugin>/README.md` added (short usage + routes + config example).
- [ ] Unit tests added and passing (`go test ./...`).
- [ ] Migrations declared and `make migrate-plugins-up --only=<plugin_id>` succeeds locally.
- [ ] Services registered with `ServiceRegistry` when applicable.
- [ ] Lint & build: `make build`, `make lint`.

---

## 11) Example: skeleton plugin (pseudo-code using exact signatures)

```go
// plugins/myplugin/plugin.go — skeleton
package myplugin

type MyPlugin struct { /* plugin fields */ }

func New(cfg types.MyPluginConfig) *MyPlugin {
  cfg.ApplyDefaults();
  return &MyPlugin{/*...*/}
}

func (p *MyPlugin) Metadata() models.PluginMetadata {
  return models.PluginMetadata{
    ID: "myplugin",
    Version: "1.0.0"
  }
}

func (p *MyPlugin) Config() any { return p.pluginConfig }

func (p *MyPlugin) Init(ctx *models.PluginContext) error {
  // Load plugin config
  if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.pluginConfig); err != nil {
    return err
  }
  // Register services/repos
  ctx.ServiceRegistry.Register("my_service", NewMyService(...))
  return nil
}

func (p *MyPlugin) Close() error {
  return nil
}

// Optional: Routes, Hooks, Migrations
func (p *MyPlugin) Routes() []models.Route {
  return Routes(p)
}

func (p *MyPlugin) Hooks() []models.Hook {
  return p.buildHooks()
}

func (p *MyPlugin) Migrations(provider string) []migrations.Migration {
  return myPluginMigrationsForProvider(provider)
}
```

---

## 12) Verification commands (local)

- Build & test: `make build && make test`
- Apply plugin migrations only: `go run ./cmd/migrate plugins up --only=<plugin_id>`
- Show migration status: `go run ./cmd/migrate status --plugin=<plugin_id>`
- Start app (playground) with plugin enabled and call routes (e.g. `/myplugin/example` for MyPlugin).

---

## 13) References (canonical files you should inspect)

- `models/plugin.go` — plugin interfaces and `PluginContext`
- `internal/bootstrap/plugin_factory.go` — how plugins are constructed from config
- `internal/plugins/plugin_registry.go` — lifecycle, RunMigrations, InitAll
- `migrations/migrator.go` — migration contract and Migration struct
- Canonical example: `plugins/myplugin/*` (see `plugin.go`, `migrations.go`, `routes.go`, `hooks.go`, `types/types.go`, `plugin_test.go`)

---

## 14) Minimal README template for each plugin

- Short description
- Config example (`[plugins.<plugin_id>] enabled = true`)
- Routes and example curl commands
- Migration notes
- Tests to run

---

## Conclusion

- Follow the `MyPlugin` plugin structure and exact signatures above.
- When unsure, look at the plugins under the `plugins/` directory as examples e.g. the email-password plugin.
