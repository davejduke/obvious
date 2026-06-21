package connector

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds all registered connectors and provides thread-safe access.
type Registry struct {
	mu         sync.RWMutex
	connectors map[string]Connector
}

// NewRegistry creates an empty, ready-to-use Registry.
func NewRegistry() *Registry {
	return &Registry{
		connectors: make(map[string]Connector),
	}
}

// Register adds c to the registry under its name.
// Returns an error if a connector with the same name is already registered.
func (r *Registry) Register(name string, c Connector) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.connectors[name]; exists {
		return fmt.Errorf("connector registry: %q already registered", name)
	}
	r.connectors[name] = c
	return nil
}

// MustRegister calls Register and panics on error.
// Use during init() or TestMain where registration failures are fatal.
func (r *Registry) MustRegister(name string, c Connector) {
	if err := r.Register(name, c); err != nil {
		panic(err)
	}
}

// Get returns the connector registered under name.
func (r *Registry) Get(name string) (Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.connectors[name]
	return c, ok
}

// List returns all registered connector names in alphabetical order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.connectors))
	for n := range r.connectors {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Unregister removes a connector by name. No-op if not found.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.connectors, name)
}

// Len returns the number of registered connectors.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.connectors)
}

// DefaultRegistry is a process-level registry for convenience.
// Large applications should prefer creating their own Registry.
var DefaultRegistry = NewRegistry()
