package session

// SessionAPI defines the programmatic API for the session plugin.
type SessionAPI interface {
}

// sessionPluginAPI implements the SessionAPI interface.
type sessionPluginAPI struct {
	plugin *SessionPlugin
}

// newSessionAPI creates a new API instance for the session plugin.
func newSessionAPI(plugin *SessionPlugin) SessionAPI {
	return &sessionPluginAPI{
		plugin: plugin,
	}
}
