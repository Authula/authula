# Plugin Integration Testing Conventions

Integration tests in `go-better-auth` exercise the full HTTP lifecycle, including routes, hooks, database persistence, and RBAC enforcement.

## File Organization

- **`routes_integration_test.go`**: Contains the actual test functions (`Test<Plugin>RouteIntegration_...`).
- **`test_helpers_integration_test.go`**: Definitions for test fixtures and shared helper methods.
- **Package Name**: Use `package <plugin>_test` to ensure testing through the public API.

## The Fixture Pattern

Every plugin should define a custom fixture struct that embeds `*internaltests.BaseTestFixture`.

```go
type myPluginFixture struct {
    *internaltests.BaseTestFixture
    Auth   *gobetterauth.Auth
    Plugin *myplugin.MyPlugin
    Router *gobetterauth.Router
}

func newMyPluginFixture(t *testing.T) *myPluginFixture {
    t.Helper()
    plugin := myplugin.New(myplugin.Config{Enabled: true})
    base := internaltests.NewBaseTestFixture(t)
    auth := gobetterauth.New(&gobetterauth.AuthConfig{
        Config:  base.Config,
        Plugins: []models.Plugin{plugin},
        DB:      base.DB,
    })
    return &myPluginFixture{
        BaseTestFixture: base,
        Auth:            auth,
        Plugin:          plugin,
        Router:          auth.Router(),
    }
}
```

## Key Concept: ID Aliasing

Use `f.ResolveID("my-test-user")` instead of hardcoded UUIDs. The `BaseTestFixture` maintains a map of string aliases to stable UUIDs, ensuring consistency between seeding and requests.

## Key Concept: Hook Injection for Auth

To simulate an authenticated user without performing a full login flow, register a temporary "Before Hook" in the router:

```go
func (f *myPluginFixture) AuthenticateAs(userID string) {
    resolvedUserID := f.ResolveID(userID)
    f.Router.RegisterHook(models.Hook{
        Stage: models.HookBefore,
        Order: 1,
        Handler: func(reqCtx *models.RequestContext) error {
            reqCtx.SetUserIDInContext(resolvedUserID)
            return nil
        },
    })
}
```

## Testing RBAC

1. Define test-specific `RouteMapping` arrays.
2. Use `f.Router.SetRouteMetadataFromConfig(metadata)` to apply them.
3. Test that unauthenticated/unauthorized requests return `401/403` and authorized requests return `200/201`.

## Best Practices

- **Use `JSONRequest`**: Abstract `httptest.NewRequest` and `httptest.NewRecorder` into a helper method.
- **Seed via API**: Use `f.Plugin.Api` to set up state (roles, permissions, data) before testing routes.
- **Clean Assertions**: Use `testify/assert` to verify status codes and JSON response bodies.
- **Isolation**: Each test should call `new<Plugin>Fixture(t)` to get a fresh database and router instance.
