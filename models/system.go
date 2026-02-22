package models

import "context"

// CoreSystem defines an interface for components that need their own initialization and lifecycle.
type CoreSystem interface {
	// Name returns the identifier for this core system
	Name() string
	// Init performs the core system's initialization logic (e.g. starting background loops)
	Init(ctx context.Context) error
	// Close performs cleanup for the core system
	Close() error
}
