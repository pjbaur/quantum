package gates

import (
	"errors"
	"fmt"

	"github.com/pjbaur/quantum/quantum"
)

// Registry stores gate instances by name.
type Registry struct {
	gates map[string]quantum.Gate
}

// NewRegistry creates an empty gate registry.
func NewRegistry() *Registry {
	return &Registry{
		gates: make(map[string]quantum.Gate),
	}
}

// Register adds a gate to the registry using its Name.
func (r *Registry) Register(gate quantum.Gate) error {
	if r == nil {
		return errors.New("registry is nil")
	}
	if gate == nil {
		return errors.New("gate must not be nil")
	}

	name := gate.Name()
	if name == "" {
		return errors.New("gate name must not be empty")
	}
	if _, exists := r.gates[name]; exists {
		return fmt.Errorf("gate %q already registered", name)
	}

	r.gates[name] = gate
	return nil
}

// Lookup returns the registered gate by name.
func (r *Registry) Lookup(name string) (quantum.Gate, bool) {
	if r == nil {
		return nil, false
	}
	gate, ok := r.gates[name]
	return gate, ok
}
