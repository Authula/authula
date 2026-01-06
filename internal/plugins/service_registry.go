package plugins

import (
	"sync"
)

type DefaultServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]any
}

func NewServiceRegistry() *DefaultServiceRegistry {
	return &DefaultServiceRegistry{
		services: make(map[string]any),
	}
}

func (r *DefaultServiceRegistry) Register(name string, service any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[name] = service
}

func (r *DefaultServiceRegistry) Get(name string) any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services[name]
}
