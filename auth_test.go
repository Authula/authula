package authula

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	authulaconfig "github.com/Authula/authula/config"
	"github.com/Authula/authula/internal/migrationmanager"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	coreservices "github.com/Authula/authula/services"
)

// authTestSecret is the public development secret used by the default config.
const authTestSecret = "YXV0aHVsYS1kZXYta2V5LTAxMjM0NTY3ODlhYmNkZWY="

// fakeAuthPluginConfig is a typed plugin config with the enabled flag that
// util.IsPluginEnabled looks for.
type fakeAuthPluginConfig struct {
	Enabled bool `json:"enabled"`
}

// fakeAuthPlugin implements models.Plugin plus every optional plugin interface
// so a single fake can exercise middleware, route, hook, and migration paths.
type fakeAuthPlugin struct {
	id         string
	config     any
	initErr    error
	closeErr   error
	middleware []func(http.Handler) http.Handler
	routes     []models.Route
	hooks      []models.Hook
	migrations []migrations.Migration
	dependsOn  []string

	initCalls  int
	closeCalls int
}

func newFakeAuthPlugin(id string, enabled bool) *fakeAuthPlugin {
	return &fakeAuthPlugin{id: id, config: fakeAuthPluginConfig{Enabled: enabled}}
}

func (p *fakeAuthPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{ID: p.id, Version: "test", Description: "fake auth plugin"}
}

func (p *fakeAuthPlugin) Config() any { return p.config }

func (p *fakeAuthPlugin) Init(*models.PluginContext) error {
	p.initCalls++
	return p.initErr
}

func (p *fakeAuthPlugin) Close() error {
	p.closeCalls++
	return p.closeErr
}

func (p *fakeAuthPlugin) Middleware() []func(http.Handler) http.Handler { return p.middleware }
func (p *fakeAuthPlugin) Routes() []models.Route                        { return p.routes }
func (p *fakeAuthPlugin) Hooks() []models.Hook                          { return p.hooks }
func (p *fakeAuthPlugin) DependsOn() []string                           { return p.dependsOn }

func (p *fakeAuthPlugin) Migrations(string) []migrations.Migration { return p.migrations }

// plainAuthPlugin implements only the base plugin interface.
type plainAuthPlugin struct {
	id string
}

func (p *plainAuthPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{ID: p.id, Version: "test"}
}

func (p *plainAuthPlugin) Config() any                      { return nil }
func (p *plainAuthPlugin) Init(*models.PluginContext) error { return nil }
func (p *plainAuthPlugin) Close() error                     { return nil }

// fakeAuthPluginRegistry is an in-memory models.PluginRegistry for tests that
// need full control over the registered plugins.
type fakeAuthPluginRegistry struct {
	config        *models.Config
	plugins       []models.Plugin
	closeAllCalls int
}

func (r *fakeAuthPluginRegistry) Register(plugin models.Plugin) error {
	r.plugins = append(r.plugins, plugin)
	return nil
}

func (r *fakeAuthPluginRegistry) InitAll() error                       { return nil }
func (r *fakeAuthPluginRegistry) RunMigrations(context.Context) error  { return nil }
func (r *fakeAuthPluginRegistry) DropMigrations(context.Context) error { return nil }
func (r *fakeAuthPluginRegistry) Plugins() []models.Plugin             { return r.plugins }
func (r *fakeAuthPluginRegistry) GetConfig() *models.Config            { return r.config }
func (r *fakeAuthPluginRegistry) CloseAll()                            { r.closeAllCalls++ }

func (r *fakeAuthPluginRegistry) GetPlugin(pluginID string) models.Plugin {
	for _, plugin := range r.plugins {
		if plugin.Metadata().ID == pluginID {
			return plugin
		}
	}
	return nil
}

// fakeAuthSystem is a models.CoreSystem with controllable errors.
type fakeAuthSystem struct {
	name       string
	initErr    error
	closeErr   error
	initCalls  int
	closeCalls int
}

func (s *fakeAuthSystem) Name() string { return s.name }

func (s *fakeAuthSystem) Init(context.Context) error {
	s.initCalls++
	return s.initErr
}

func (s *fakeAuthSystem) Close() error {
	s.closeCalls++
	return s.closeErr
}

// wrappedIDB implements bun.IDB without being a *bun.DB, which lets tests drive
// the migrator's rejected-input branch.
type wrappedIDB struct {
	bun.IDB
}

func newAuthTestConfig(t *testing.T) *models.Config {
	t.Helper()

	return authulaconfig.NewConfig(
		authulaconfig.WithDatabase(models.DatabaseConfig{Provider: "sqlite", URL: ":memory:"}),
	)
}

// newAuthMinimalConfig builds a config without the maps and event bus defaults
// that config.NewConfig fills in, so New has to initialize them itself.
func newAuthMinimalConfig() *models.Config {
	return &models.Config{
		AppName:  "Authula",
		BaseURL:  "http://localhost:8080",
		BasePath: "/auth",
		Secret:   authTestSecret,
		Database: models.DatabaseConfig{Provider: "sqlite"},
		EventBus: models.EventBusConfig{Provider: "gochannel"},
	}
}

func newAuthWithRouter(t *testing.T, config *models.Config, logger models.Logger) *Auth {
	t.Helper()

	if config == nil {
		config = &models.Config{}
	}

	return &Auth{
		config: config,
		logger: logger,
		router: NewRouter(config, logger, nil),
	}
}

func newComposedAuth(t *testing.T, plugins ...models.Plugin) *Auth {
	t.Helper()

	return New(&AuthConfig{
		Config:  newAuthTestConfig(t),
		DB:      internaltests.NewSQLiteIntegrationDB(t),
		Plugins: plugins,
	})
}

func performAuthRequest(handler http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func okAuthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// requirePanicMessage asserts that fn panics with the given error or string
// value. New panics with plain strings for programmer errors and with wrapped
// errors for runtime failures, so both are compared by message.
func requirePanicMessage(t *testing.T, want string, fn func()) {
	t.Helper()

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		fn()
	}()

	require.NotNil(t, recovered, "expected a panic")

	if err, ok := recovered.(error); ok {
		assert.EqualError(t, err, want)
		return
	}

	assert.Equal(t, want, recovered)
}

func TestAuthAccessors(t *testing.T) {
	t.Parallel()

	accessors := []struct {
		name    string
		nilSafe bool
		got     func(*Auth) any
		want    func(*Auth) any
	}{
		{name: "Logger", got: func(a *Auth) any { return a.Logger() }, want: func(a *Auth) any { return a.logger }},
		{name: "DB", nilSafe: true, got: func(a *Auth) any { return a.DB() }, want: func(a *Auth) any { return a.db }},
		{name: "EventBus", nilSafe: true, got: func(a *Auth) any { return a.EventBus() }, want: func(a *Auth) any { return a.eventBus }},
		{name: "Router", nilSafe: true, got: func(a *Auth) any { return a.Router() }, want: func(a *Auth) any { return a.router }},
		{name: "Migrator", nilSafe: true, got: func(a *Auth) any { return a.Migrator() }, want: func(a *Auth) any { return a.migrator }},
		{name: "MigrationManager", nilSafe: true, got: func(a *Auth) any { return a.MigrationManager() }, want: func(a *Auth) any { return a.migrationManager }},
		{name: "CoreServices", nilSafe: true, got: func(a *Auth) any { return a.CoreServices() }, want: func(a *Auth) any { return a.coreServices }},
	}

	t.Run("nil receiver returns nil", func(t *testing.T) {
		t.Parallel()

		var auth *Auth
		for _, tc := range accessors {
			if !tc.nilSafe {
				continue
			}
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Nil(t, tc.got(auth))
			})
		}
	})

	t.Run("zero value returns nil fields", func(t *testing.T) {
		t.Parallel()

		auth := &Auth{}
		for _, tc := range accessors {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Nil(t, tc.got(auth))
			})
		}
	})

	t.Run("populated returns stored values", func(t *testing.T) {
		t.Parallel()

		db := internaltests.NewSQLiteIntegrationDB(t)
		migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
		require.NoError(t, err)

		auth := &Auth{
			config:           &models.Config{},
			logger:           &internaltests.MockLogger{},
			db:               db,
			eventBus:         &internaltests.MockEventBus{},
			migrator:         migrator,
			migrationManager: migrationmanager.NewManager(migrator),
			coreServices:     &coreservices.CoreServices{},
			router:           NewRouter(&models.Config{}, &internaltests.MockLogger{}, nil),
		}

		for _, tc := range accessors {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.NotNil(t, tc.want(auth))
				assert.Same(t, tc.want(auth), tc.got(auth))
			})
		}
	})
}

func TestNewPanics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		arrange func(t *testing.T) *AuthConfig
		want    string
	}{
		{
			name:    "nil auth config",
			arrange: func(*testing.T) *AuthConfig { return nil },
			want:    "auth config and auth config config must not be nil",
		},
		{
			name:    "nil config",
			arrange: func(*testing.T) *AuthConfig { return &AuthConfig{} },
			want:    "auth config and auth config config must not be nil",
		},
		{
			name: "database init failure",
			arrange: func(t *testing.T) *AuthConfig {
				t.Helper()
				config := newAuthTestConfig(t)
				config.Database.Provider = "bogus"
				return &AuthConfig{Config: config}
			},
			want: "failed to initialize database: unsupported database provider: bogus",
		},
		{
			name: "migrator requires a bun db",
			arrange: func(t *testing.T) *AuthConfig {
				t.Helper()
				return &AuthConfig{
					Config: newAuthTestConfig(t),
					DB:     wrappedIDB{IDB: internaltests.NewSQLiteIntegrationDB(t)},
				}
			},
			want: "failed to create migrator: migrator requires *bun.DB instance",
		},
		{
			name: "core migration failure",
			arrange: func(t *testing.T) *AuthConfig {
				t.Helper()
				config := newAuthTestConfig(t)
				config.Database.Provider = "bogus"
				return &AuthConfig{Config: config, DB: internaltests.NewSQLiteIntegrationDB(t)}
			},
			want: "failed to run core migrations: unsupported database provider: bogus",
		},
		{
			name: "event bus init failure",
			arrange: func(t *testing.T) *AuthConfig {
				t.Helper()
				config := newAuthTestConfig(t)
				config.EventBus = models.EventBusConfig{Provider: "bogus"}
				return &AuthConfig{Config: config, DB: internaltests.NewSQLiteIntegrationDB(t)}
			},
			want: "failed to initialize event bus: unsupported event bus provider: bogus",
		},
		{
			name: "duplicate plugin ids",
			arrange: func(t *testing.T) *AuthConfig {
				t.Helper()
				return &AuthConfig{
					Config:  newAuthTestConfig(t),
					DB:      internaltests.NewSQLiteIntegrationDB(t),
					Plugins: []models.Plugin{newFakeAuthPlugin("dup", true), newFakeAuthPlugin("dup", true)},
				}
			},
			want: `failed to register plugin: plugin with ID "dup" is already registered`,
		},
		{
			name: "plugin migration failure",
			arrange: func(t *testing.T) *AuthConfig {
				t.Helper()
				plugin := newFakeAuthPlugin("fake", true)
				plugin.dependsOn = []string{"missing"}
				plugin.migrations = []migrations.Migration{
					{Version: "1", Up: func(context.Context, bun.Tx) error { return nil }},
				}
				return &AuthConfig{
					Config:  newAuthTestConfig(t),
					DB:      internaltests.NewSQLiteIntegrationDB(t),
					Plugins: []models.Plugin{plugin},
				}
			},
			want: "failed to run plugin migrations: plugin fake depends on missing which has no registered migrations",
		},
		{
			name: "plugin init failure",
			arrange: func(t *testing.T) *AuthConfig {
				t.Helper()
				plugin := newFakeAuthPlugin("fake", true)
				plugin.initErr = errors.New("init boom")
				return &AuthConfig{
					Config:  newAuthTestConfig(t),
					DB:      internaltests.NewSQLiteIntegrationDB(t),
					Plugins: []models.Plugin{plugin},
				}
			},
			want: "failed to initialize plugins: init boom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			authConfig := tc.arrange(t)
			requirePanicMessage(t, tc.want, func() { New(authConfig) })
		})
	}
}

func TestNewCompositionRoot(t *testing.T) {
	t.Parallel()

	config := newAuthTestConfig(t)
	config.RouteMappings = []models.RouteMapping{
		{Paths: []string{"GET:/fake/ping"}, Plugins: []string{"fake"}},
	}

	hookRuns := 0
	plugin := newFakeAuthPlugin("fake", true)
	plugin.middleware = []func(http.Handler) http.Handler{
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Fake-Middleware", "on")
				next.ServeHTTP(w, r)
			})
		},
	}
	plugin.routes = []models.Route{
		{
			Method: http.MethodGet,
			Path:   "/fake/ping",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("pong"))
			}),
		},
		{
			Method:  http.MethodGet,
			Path:    "/fake/other",
			Handler: http.HandlerFunc(okAuthHandler),
		},
	}
	plugin.hooks = []models.Hook{
		{
			Stage:    models.HookOnRequest,
			PluginID: "fake",
			Handler: func(*models.RequestContext) error {
				hookRuns++
				return nil
			},
		},
	}
	disabled := newFakeAuthPlugin("fake-disabled", false)

	auth := New(&AuthConfig{
		Config:  config,
		DB:      internaltests.NewSQLiteIntegrationDB(t),
		Plugins: []models.Plugin{plugin, disabled},
	})
	t.Cleanup(func() { _ = auth.EventBus().Close() })

	require.NotNil(t, auth)
	assert.NotNil(t, auth.Logger())
	assert.Same(t, auth.DB(), auth.db)
	assert.NotNil(t, auth.EventBus())
	assert.NotNil(t, auth.Router())
	assert.NotNil(t, auth.Migrator())
	assert.NotNil(t, auth.MigrationManager())
	assert.NotNil(t, auth.CoreServices())
	assert.NotNil(t, auth.ServiceRegistry)
	assert.NotNil(t, auth.PluginRegistry)
	assert.NotNil(t, auth.Api)

	assert.Equal(t, fakeAuthPluginConfig{Enabled: true}, config.PreParsedConfigs["fake"])
	assert.Equal(t, fakeAuthPluginConfig{Enabled: false}, config.PreParsedConfigs["fake-disabled"])
	assert.Equal(t, 1, plugin.initCalls)
	assert.Equal(t, 0, disabled.initCalls)
	assert.NotNil(t, auth.PluginRegistry.GetPlugin("fake"))
	assert.Nil(t, auth.PluginRegistry.GetPlugin("fake-disabled"))

	handler := auth.Handler()
	require.NotNil(t, handler)
	assert.Same(t, handler, auth.Handler(), "handler must be built once")

	assert.Equal(t, http.StatusUnauthorized, performAuthRequest(handler, http.MethodGet, "/auth/me").Code)

	rec := performAuthRequest(handler, http.MethodGet, "/auth/fake/ping")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
	assert.Equal(t, "on", rec.Header().Get("X-Fake-Middleware"))
	assert.Equal(t, 1, hookRuns)

	rec = performAuthRequest(handler, http.MethodGet, "/auth/fake/other")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, hookRuns, "plugin hook is scoped to configured route metadata")

	require.NoError(t, auth.RunCoreMigrations(context.Background()))

	require.NoError(t, auth.ClosePlugins())
	assert.Equal(t, 1, plugin.closeCalls)
	assert.Equal(t, 0, disabled.closeCalls)

	require.NoError(t, auth.CloseSystems())
}

func TestNewInitializesDatabaseWhenNotProvided(t *testing.T) {
	t.Parallel()

	config := newAuthTestConfig(t)
	config.Database.URL = filepath.Join(t.TempDir(), "auth.db")

	auth := New(&AuthConfig{Config: config})
	t.Cleanup(func() { _ = auth.EventBus().Close() })

	require.NotNil(t, auth)
	require.NotNil(t, auth.DB())

	db, ok := auth.DB().(*bun.DB)
	require.True(t, ok)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, auth.CloseSystems())
}

func TestNewInitializesConfigMaps(t *testing.T) {
	t.Parallel()

	config := newAuthMinimalConfig()
	plugin := &plainAuthPlugin{id: "plain"}

	auth := New(&AuthConfig{
		Config:  config,
		DB:      internaltests.NewSQLiteIntegrationDB(t),
		Plugins: []models.Plugin{plugin},
	})
	t.Cleanup(func() { _ = auth.EventBus().Close() })

	require.NotNil(t, auth)
	assert.NotNil(t, config.PreParsedConfigs)
	assert.NotNil(t, config.Plugins)
	assert.Nil(t, auth.PluginRegistry.GetPlugin("plain"), "plugin without enabled config stays disabled")
	require.NoError(t, auth.CloseSystems())
}

func TestAuthRegisterDelegation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		register   func(*Auth)
		method     string
		path       string
		wantStatus int
		wantHeader map[string]string
	}{
		{
			name: "single route",
			register: func(auth *Auth) {
				auth.RegisterRoute(models.Route{
					Method:  http.MethodGet,
					Path:    "/one",
					Handler: http.HandlerFunc(okAuthHandler),
				})
			},
			method:     http.MethodGet,
			path:       "/auth/one",
			wantStatus: http.StatusOK,
		},
		{
			name: "multiple routes",
			register: func(auth *Auth) {
				auth.RegisterRoutes([]models.Route{
					{Method: http.MethodGet, Path: "/first", Handler: http.HandlerFunc(okAuthHandler)},
					{Method: http.MethodPost, Path: "/second", Handler: http.HandlerFunc(okAuthHandler)},
				})
			},
			method:     http.MethodPost,
			path:       "/auth/second",
			wantStatus: http.StatusOK,
		},
		{
			name: "custom route ignores base path",
			register: func(auth *Auth) {
				auth.RegisterCustomRoute(models.Route{
					Method:  http.MethodGet,
					Path:    "/custom",
					Handler: http.HandlerFunc(okAuthHandler),
				})
			},
			method:     http.MethodGet,
			path:       "/custom",
			wantStatus: http.StatusOK,
		},
		{
			name: "custom route group",
			register: func(auth *Auth) {
				auth.RegisterCustomRouteGroup(models.RouteGroup{
					Path: "/group",
					Routes: []models.Route{
						{Method: http.MethodGet, Path: "/child", Handler: http.HandlerFunc(okAuthHandler)},
					},
					Metadata: map[string]any{"plugins": []string{"group.plugin"}},
				})
			},
			method:     http.MethodGet,
			path:       "/group/child",
			wantStatus: http.StatusOK,
		},
		{
			name: "multiple custom routes",
			register: func(auth *Auth) {
				auth.RegisterCustomRoutes([]models.Route{
					{Method: http.MethodGet, Path: "/custom-one", Handler: http.HandlerFunc(okAuthHandler)},
					{Method: http.MethodGet, Path: "/custom-two", Handler: http.HandlerFunc(okAuthHandler)},
				})
			},
			method:     http.MethodGet,
			path:       "/custom-two",
			wantStatus: http.StatusOK,
		},
		{
			name: "middleware",
			register: func(auth *Auth) {
				auth.RegisterMiddleware(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("X-Auth-Middleware", "on")
						next.ServeHTTP(w, r)
					})
				})
				auth.RegisterRoute(models.Route{
					Method:  http.MethodGet,
					Path:    "/middleware",
					Handler: http.HandlerFunc(okAuthHandler),
				})
			},
			method:     http.MethodGet,
			path:       "/auth/middleware",
			wantStatus: http.StatusOK,
			wantHeader: map[string]string{"X-Auth-Middleware": "on"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			auth := newAuthWithRouter(t, &models.Config{BasePath: "/auth"}, &testLogger{})
			tc.register(auth)

			rec := performAuthRequest(auth.Router().Handler(), tc.method, tc.path)
			assert.Equal(t, tc.wantStatus, rec.Code)

			for header, want := range tc.wantHeader {
				assert.Equal(t, want, rec.Header().Get(header))
			}
		})
	}
}

func TestAuthRegisterHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		register func(*Auth, *int)
		wantRuns int
	}{
		{
			name: "single hook",
			register: func(auth *Auth, runs *int) {
				auth.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Handler: func(*models.RequestContext) error {
						*runs++
						return nil
					},
				})
			},
			wantRuns: 1,
		},
		{
			name: "multiple hooks",
			register: func(auth *Auth, runs *int) {
				auth.RegisterHooks([]models.Hook{
					{
						Stage: models.HookBefore,
						Handler: func(*models.RequestContext) error {
							*runs++
							return nil
						},
					},
					{
						Stage: models.HookOnRequest,
						Handler: func(*models.RequestContext) error {
							*runs++
							return nil
						},
					},
				})
			},
			wantRuns: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			auth := newAuthWithRouter(t, &models.Config{BasePath: "/auth"}, &testLogger{})
			auth.RegisterRoute(models.Route{
				Method:  http.MethodGet,
				Path:    "/hooked",
				Handler: http.HandlerFunc(okAuthHandler),
			})

			runs := 0
			tc.register(auth, &runs)

			rec := performAuthRequest(auth.Router().Handler(), http.MethodGet, "/auth/hooked")
			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tc.wantRuns, runs)
		})
	}
}

func TestAuthRegisterMiddlewareFromPlugins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		plugin     models.Plugin
		enabled    bool
		wantHeader string
		wantDebug  string
	}{
		{
			name: "enabled plugin middleware is registered",
			plugin: func() models.Plugin {
				plugin := newFakeAuthPlugin("middleware", true)
				plugin.middleware = []func(http.Handler) http.Handler{
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("X-Plugin-Middleware", "on")
							next.ServeHTTP(w, r)
						})
					},
				}
				return plugin
			}(),
			enabled:    true,
			wantHeader: "on",
		},
		{
			name:      "plugin without middleware functions",
			plugin:    newFakeAuthPlugin("empty", true),
			enabled:   true,
			wantDebug: "no middleware functions returned",
		},
		{
			name: "disabled plugin is skipped",
			plugin: func() models.Plugin {
				plugin := newFakeAuthPlugin("disabled", false)
				plugin.middleware = []func(http.Handler) http.Handler{
					func(next http.Handler) http.Handler { return next },
				}
				return plugin
			}(),
			enabled:   false,
			wantDebug: "skipping disabled plugin",
		},
		{
			name:    "plugin without middleware interface",
			plugin:  &plainAuthPlugin{id: "plain"},
			enabled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := &models.Config{
				BasePath: "/auth",
				Plugins: models.PluginsConfig{
					tc.plugin.Metadata().ID: map[string]any{"enabled": tc.enabled},
				},
			}
			logger := &internaltests.LevelRecordingLogger{}
			auth := newAuthWithRouter(t, config, logger)
			auth.PluginRegistry = &fakeAuthPluginRegistry{config: config, plugins: []models.Plugin{tc.plugin}}

			auth.registerMiddleware()
			auth.RegisterRoute(models.Route{
				Method:  http.MethodGet,
				Path:    "/probe",
				Handler: http.HandlerFunc(okAuthHandler),
			})

			rec := performAuthRequest(auth.Router().Handler(), http.MethodGet, "/auth/probe")
			require.Equal(t, http.StatusOK, rec.Code)

			if tc.wantHeader != "" {
				assert.Equal(t, tc.wantHeader, rec.Header().Get("X-Plugin-Middleware"))
			} else {
				assert.Empty(t, rec.Header().Get("X-Plugin-Middleware"))
			}

			if tc.wantDebug != "" {
				assert.Contains(t, logger.Messages(), tc.wantDebug)
			}
		})
	}
}

func TestAuthHandlerPluginRegistrations(t *testing.T) {
	t.Parallel()

	hookRuns := 0

	enabled := newFakeAuthPlugin("enabled", true)
	enabled.routes = []models.Route{
		{
			Method:  http.MethodGet,
			Path:    "/enabled/ping",
			Handler: http.HandlerFunc(okAuthHandler),
		},
		{
			Method:  http.MethodGet,
			Path:    "/enabled/other",
			Handler: http.HandlerFunc(okAuthHandler),
		},
	}
	enabled.hooks = []models.Hook{
		{
			Stage:    models.HookOnRequest,
			PluginID: "enabled",
			Handler: func(*models.RequestContext) error {
				hookRuns++
				return nil
			},
		},
	}

	disabled := newFakeAuthPlugin("disabled", false)
	disabled.routes = []models.Route{
		{
			Method:  http.MethodGet,
			Path:    "/disabled/ping",
			Handler: http.HandlerFunc(okAuthHandler),
		},
	}
	disabled.hooks = []models.Hook{
		{
			Stage: models.HookOnRequest,
			Handler: func(*models.RequestContext) error {
				hookRuns += 100
				return nil
			},
		},
	}

	config := &models.Config{
		BasePath: "/auth",
		RouteMappings: []models.RouteMapping{
			{Paths: []string{"GET:/enabled/ping"}, Plugins: []string{"enabled"}},
		},
		Plugins: models.PluginsConfig{
			"enabled":  map[string]any{"enabled": true},
			"disabled": map[string]any{"enabled": false},
			"empty":    map[string]any{"enabled": true},
			"plain":    map[string]any{"enabled": true},
		},
	}

	logger := &internaltests.RecordingLogger{}
	registry := &fakeAuthPluginRegistry{
		config: config,
		plugins: []models.Plugin{
			enabled,
			disabled,
			newFakeAuthPlugin("empty", true),
			&plainAuthPlugin{id: "plain"},
		},
	}

	auth := newAuthWithRouter(t, config, logger)
	auth.PluginRegistry = registry
	auth.coreServices = &coreservices.CoreServices{
		UserService:    &internaltests.MockUserService{},
		SessionService: &internaltests.MockSessionService{},
	}

	handler := auth.Handler()
	require.NotNil(t, handler)
	assert.Same(t, handler, auth.Handler())

	assert.Equal(t, http.StatusUnauthorized, performAuthRequest(handler, http.MethodGet, "/auth/me").Code)
	assert.Equal(t, http.StatusOK, performAuthRequest(handler, http.MethodGet, "/auth/enabled/ping").Code)
	assert.Equal(t, http.StatusOK, performAuthRequest(handler, http.MethodGet, "/auth/enabled/other").Code)
	assert.Equal(t, http.StatusNotFound, performAuthRequest(handler, http.MethodGet, "/auth/disabled/ping").Code)

	assert.Equal(t, 1, hookRuns)
	assert.Empty(t, logger.ErrorCalls)
}

func TestAuthHandlerRouteMetadataConversionError(t *testing.T) {
	t.Parallel()

	config := &models.Config{
		BasePath: "/auth",
		RouteMappings: []models.RouteMapping{
			{Paths: []string{"not-a-route"}, Plugins: []string{"enabled"}},
		},
	}

	logger := &internaltests.RecordingLogger{}
	auth := newAuthWithRouter(t, config, logger)
	auth.PluginRegistry = &fakeAuthPluginRegistry{config: config}
	auth.coreServices = &coreservices.CoreServices{
		UserService:    &internaltests.MockUserService{},
		SessionService: &internaltests.MockSessionService{},
	}

	handler := auth.Handler()

	assert.Equal(t, http.StatusUnauthorized, performAuthRequest(handler, http.MethodGet, "/auth/me").Code)
	assert.Contains(t, logger.ErrorCalls, "failed to convert route metadata")
}

func TestAuthActorHelpers(t *testing.T) {
	t.Parallel()

	actor := &models.Actor{ID: "actor-1", Type: models.ActorUser}

	tests := []struct {
		name   string
		ctx    context.Context
		want   *models.Actor
		wantOK bool
	}{
		{
			name:   "actor stored in the go context",
			ctx:    context.WithValue(context.Background(), models.ContextAuthActor, actor),
			want:   actor,
			wantOK: true,
		},
		{
			name: "actor stored in the request context",
			ctx: models.NewContextWithRequestContext(
				context.Background(),
				&models.RequestContext{Actor: actor},
			),
			want:   actor,
			wantOK: true,
		},
		{
			name: "actor set through the request context helper",
			ctx: func() context.Context {
				reqCtx := &models.RequestContext{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
				reqCtx.SetActorInContext(actor)
				return models.NewContextWithRequestContext(reqCtx.Request.Context(), reqCtx)
			}(),
			want:   actor,
			wantOK: true,
		},
		{
			name:   "missing actor",
			ctx:    context.Background(),
			want:   nil,
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			auth := &Auth{}
			got, ok := auth.GetActorFromContext(tc.ctx)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}

	requestTests := []struct {
		name   string
		build  func() *http.Request
		wantOK bool
	}{
		{
			name: "request with an actor",
			build: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				reqCtx := &models.RequestContext{Request: req}
				reqCtx.SetActorInContext(actor)
				return req.WithContext(models.NewContextWithRequestContext(reqCtx.Request.Context(), reqCtx))
			},
			wantOK: true,
		},
		{
			name:   "request without an actor",
			build:  func() *http.Request { return httptest.NewRequest(http.MethodGet, "/", nil) },
			wantOK: false,
		},
	}

	for _, tc := range requestTests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			auth := &Auth{}
			got, ok := auth.GetActorFromRequest(tc.build())
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, actor, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestAuthCoreMigrations(t *testing.T) {
	t.Parallel()

	uninitializedAuth := func(*testing.T) *Auth { return &Auth{} }
	sqliteAuth := func(t *testing.T) *Auth {
		t.Helper()

		auth := newComposedAuth(t)
		t.Cleanup(func() { _ = auth.EventBus().Close() })
		return auth
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T) *Auth
		act     func(*Auth) error
		wantErr string
	}{
		{
			name:    "run without an initialized migrator returns an error",
			setup:   uninitializedAuth,
			act:     func(auth *Auth) error { return auth.RunCoreMigrations(context.Background()) },
			wantErr: "migrator not initialized",
		},
		{
			name:    "drop without an initialized migrator returns an error",
			setup:   uninitializedAuth,
			act:     func(auth *Auth) error { return auth.DropCoreMigrations(context.Background()) },
			wantErr: "migrator not initialized",
		},
		{
			name:  "run core migrations on sqlite",
			setup: sqliteAuth,
			act:   func(auth *Auth) error { return auth.RunCoreMigrations(context.Background()) },
		},
		{
			name:  "drop core migrations on sqlite",
			setup: sqliteAuth,
			act:   func(auth *Auth) error { return auth.DropCoreMigrations(context.Background()) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			auth := tc.setup(t)

			err := tc.act(auth)
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthClosePlugins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		withRegistry      bool
		wantCloseAllCalls int
	}{
		{
			name: "nil registry",
		},
		{
			name:              "closes every registered plugin",
			withRegistry:      true,
			wantCloseAllCalls: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			registry := &fakeAuthPluginRegistry{}
			auth := &Auth{}
			if tc.withRegistry {
				auth.PluginRegistry = registry
			}

			require.NoError(t, auth.ClosePlugins())
			assert.Equal(t, tc.wantCloseAllCalls, registry.closeAllCalls)
		})
	}
}

func TestAuthCloseSystems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		systems       []*fakeAuthSystem
		wantErrorLogs []string
	}{
		{
			name: "no systems",
		},
		{
			name: "closes systems and logs failures",
			systems: []*fakeAuthSystem{
				{name: "ok"},
				{name: "failing", closeErr: errors.New("close failed")},
			},
			wantErrorLogs: []string{"failed to close core system"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := &internaltests.RecordingLogger{}
			auth := &Auth{logger: logger}
			for _, system := range tc.systems {
				auth.systems = append(auth.systems, system)
			}

			require.NoError(t, auth.CloseSystems())
			for _, system := range tc.systems {
				assert.Equal(t, 1, system.closeCalls, "system %s", system.name)
			}
			assert.Equal(t, tc.wantErrorLogs, logger.ErrorCalls)
		})
	}
}
