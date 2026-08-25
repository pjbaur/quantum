package gates

import (
	"errors"
	"fmt"
	"sort"

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

// Names returns the sorted names of all registered gates.
func (r *Registry) Names() []string {
	if r == nil || len(r.gates) == 0 {
		return nil
	}
	names := make([]string, 0, len(r.gates))
	for name := range r.gates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Builtin returns a new Registry preloaded with every built-in gate.
// Each call returns a fresh registry, so callers may extend their copy
// without affecting others.
//
// Only gates with a fixed matrix live here. The parameterized constructors
// (NewRx, NewRy, NewRz, NewPhase) and NewControlled describe a family of
// gates rather than one gate, so there is no single instance to register
// under their name; callers build the member they want and register it in
// their own copy of the registry if they need lookup by name.
func Builtin() *Registry {
	registry := NewRegistry()
	builtins := []*MatrixGate{
		NewHadamard(), NewPauliX(), NewPauliY(), NewPauliZ(),
		NewS(), NewT(), NewCNOT(), NewSwap(), NewToffoli(),
	}
	for _, gate := range builtins {
		if err := registry.Register(gate); err != nil {
			panic(fmt.Sprintf("builtin registry: %v", err))
		}
	}
	return registry
}
