package registry

import (
	"fmt"

	"github.com/pyrorhythm/moonshine/pkg/backend"
)

// Registry holds all registered backends keyed by name.
type Registry struct {
	backends map[string]backend.Backend
	order    []string // insertion order for deterministic iteration
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{backends: make(map[string]backend.Backend)}
}

// Register adds b to the registry. Panics on duplicate names.
func (r *Registry) Register(b backend.Backend) {
	name := b.Name()
	if _, exists := r.backends[name]; exists {
		panic(fmt.Sprintf("backend %q already registered", name))
	}
	r.backends[name] = b
	r.order = append(r.order, name)
}

// Get returns the backend with the given name and whether it was found.
func (r *Registry) Get(name string) (backend.Backend, bool) {
	b, ok := r.backends[name]
	return b, ok
}

// All returns all registered backends in registration order.
func (r *Registry) All() []backend.Backend {
	out := make([]backend.Backend, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.backends[name])
	}
	return out
}

// Names returns all registered backend names in registration order.
func (r *Registry) Names() []string {
	names := make([]string, len(r.order))
	copy(names, r.order)
	return names
}
