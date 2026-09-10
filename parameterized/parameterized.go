// Package parameterized provides circuit templates with named symbolic
// parameters. Bind materializes a template into an ordinary circuit, so
// variational loops can re-run the same structure at many parameter points
// without rebuilding gate lists by hand.
package parameterized

import (
	"errors"
	"fmt"
	"math"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)

// Params maps parameter names to gate angles.
type Params map[string]float64

// Factory builds a gate from a parameter value.
type Factory func(value float64) quantum.Gate

// Rx, Ry, Rz, and Phase are Factory constructors delegating to the gates
// package. Angles are not validated here: gates.NewRx and friends document
// that non-finite angles yield non-finite matrices; Bind rejects non-finite
// parameter values before any factory runs.
func Rx(value float64) quantum.Gate { return gates.NewRx(value) }

func Ry(value float64) quantum.Gate { return gates.NewRy(value) }

func Rz(value float64) quantum.Gate { return gates.NewRz(value) }

func Phase(value float64) quantum.Gate { return gates.NewPhase(value) }

// step is one template instruction: either a fixed gate (factory == nil,
// param unused) or a parameter-driven factory (factory != nil; param is
// the declared name, which may be any string, "" included). The factory,
// not the name, is what tells the two kinds apart: Bind and
// ParamStepCounts both test it, so no name can collide with a sentinel.
type step struct {
	param   string
	factory Factory
	gate    quantum.Gate
	targets []int
}

// Template is a circuit recipe with named parameter holes. The zero value
// is ready to use and is the template NewTemplate(0) returns: it declares
// nothing, rejects a declaration with an out-of-range target or with none,
// and, once Bind's own parameter checks pass, Bind fails as circuit.New
// does for a qubit count of zero. No method panics on it; NewTemplate is
// how a template for a positive qubit count is made, not a precondition
// of the methods.
//
// A Template is used in place or through a pointer, as NewTemplate returns
// it. Copying a Template by value after a declaration has been accepted is
// not supported: the copy shares the original's name-tracking map and the
// backing arrays of its step and name lists but not their slice headers,
// so declarations on the two would drift apart or overwrite each other.
// Like strings.Builder, a Template records the receiver of its first
// accepted declaration, and AddParamGate, AddGate, and Bind on a copy taken
// after that return an error, before any other check, instead of touching
// the shared state; NumQubits, ParamNames, and ParamStepCounts on such a
// copy describe the template as it was when copied. A copy taken before any
// declaration is accepted shares nothing and is an independent template.
// A Template is safe for concurrent reads after all Add calls complete.
type Template struct {
	// addr is the receiver of the first accepted declaration, set only by
	// AddParamGate and AddGate, so a by-value copy taken after that can be
	// told from the original, as strings.Builder does. Nil until then: a
	// copy of a template with no accepted declaration shares nothing.
	addr       *Template
	numQubits  int
	steps      []step
	paramOrder []string
	// seen is allocated by AddParamGate on the first declaration, the only
	// place it is written, so the zero value needs no constructor.
	seen map[string]bool
}

// errCopiedTemplate is what AddParamGate, AddGate, and Bind return on a
// Template copied by value after an accepted declaration; see Template.
var errCopiedTemplate = errors.New("Template copied by value after a declaration; a declared Template is used in place or through a pointer, never by copy")

// checkNotCopied returns errCopiedTemplate when t is a by-value copy of a
// Template that had already accepted a declaration when it was copied.
// Such a copy shares seen and the slices' backing arrays with the
// original, so a write through either, or Bind's read of seen on the
// copy, could desync the two; the three methods that touch seen or append
// call this first.
func (t *Template) checkNotCopied() error {
	if t.addr != nil && t.addr != t {
		return errCopiedTemplate
	}
	return nil
}

// NewTemplate returns a template for circuits on numQubits qubits.
func NewTemplate(numQubits int) *Template {
	return &Template{numQubits: numQubits}
}

// NumQubits returns the qubit count the template builds circuits for.
func (t *Template) NumQubits() int { return t.numQubits }

// ParamNames returns declared parameter names in first-use order, without
// duplicates.
func (t *Template) ParamNames() []string {
	out := make([]string, len(t.paramOrder))
	copy(out, t.paramOrder)
	return out
}

// ParamStepCounts returns, per declared parameter name, how many template
// steps consume it. A name driving exactly one gate counts 1; a name driving
// several gates counts one per step. Names never declared are absent, and
// every declared name is present: a step is parameter-driven when it
// carries a factory (AddParamGate rejects a nil one), the same test Bind
// applies, so a parameter named "" is counted like any other rather than
// mistaken for a fixed step.
func (t *Template) ParamStepCounts() map[string]int {
	counts := make(map[string]int, len(t.paramOrder))
	for _, s := range t.steps {
		if s.factory != nil {
			counts[s.param]++
		}
	}
	return counts
}

func (t *Template) checkTargets(targets []int) error {
	for _, target := range targets {
		if target < 0 || target >= t.numQubits {
			return &quantum.QubitsOutOfRangeError{Index: target, MaxIndex: t.numQubits - 1}
		}
	}
	return nil
}

// AddParamGate adds a gate built by factory from the named parameter's
// value at Bind time. The same name may drive several gates. The name is
// an opaque key: any string is accepted, the empty string included, and
// it is what Bind and ParamStepCounts key on. At least one target is
// required: a declaration with none is rejected here, with an error naming
// the parameter, rather than declared, counted, and left for
// circuit.AddGate to reject at Bind. On a Template copied by value after
// an accepted declaration the call is refused before any of these checks
// (see Template).
func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error {
	if err := t.checkNotCopied(); err != nil {
		return err
	}
	if factory == nil {
		return fmt.Errorf("parameter %q: factory must not be nil", name)
	}
	if len(targets) == 0 {
		return fmt.Errorf("parameter %q: at least one target is required", name)
	}
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	t.addr = t
	if t.seen == nil {
		t.seen = make(map[string]bool)
	}
	if !t.seen[name] {
		t.seen[name] = true
		t.paramOrder = append(t.paramOrder, name)
	}
	t.steps = append(t.steps, step{param: name, factory: factory, targets: targets})
	return nil
}

// AddGate adds a fixed gate needing no parameter. At least one target is
// required: a call with none is rejected here, with an error naming the
// gate, rather than appended and left for circuit.AddGate to reject at
// Bind. gate must be non-nil; a typed nil wrapped in a non-nil
// quantum.Gate value (for example, (*gates.MatrixGate)(nil)) passes the
// nil check undetected, the same caller-bug gap circuit.AddGate has, and
// panics when this method calls gate.Name() to name the gate in the
// no-target error. On a Template copied by value after an accepted
// declaration the call is refused before any of these checks (see
// Template).
func (t *Template) AddGate(gate quantum.Gate, targets ...int) error {
	if err := t.checkNotCopied(); err != nil {
		return err
	}
	if gate == nil {
		return fmt.Errorf("fixed gate must not be nil")
	}
	if len(targets) == 0 {
		return fmt.Errorf("fixed gate %q: at least one target is required", gate.Name())
	}
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	t.addr = t
	t.steps = append(t.steps, step{gate: gate, targets: targets})
	return nil
}

// Bind materializes the template into a circuit using values. Every declared
// parameter must be present and finite; unknown names are rejected so typos
// fail loudly instead of silently ignoring an angle. On a Template copied
// by value after an accepted declaration the call is refused before any of
// these checks (see Template); the refusal is a read, so Bind stays safe
// to call concurrently once all Add calls complete.
func (t *Template) Bind(values Params) (*circuit.Circuit, error) {
	if err := t.checkNotCopied(); err != nil {
		return nil, err
	}
	for _, name := range t.paramOrder {
		value, ok := values[name]
		if !ok {
			return nil, &MissingParameterError{Name: name}
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, &InvalidParameterValueError{Name: name, Value: value}
		}
	}
	for name := range values {
		if !t.seen[name] {
			return nil, &UnknownParameterError{Name: name}
		}
	}

	c, err := circuit.New(t.numQubits)
	if err != nil {
		return nil, err
	}
	for _, s := range t.steps {
		gate := s.gate
		if s.factory != nil {
			gate = s.factory(values[s.param])
		}
		if err := c.AddGate(gate, s.targets...); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// MissingParameterError indicates that a declared parameter is absent from
// the values a template is bound with.
type MissingParameterError struct {
	Name string
}

func (e *MissingParameterError) Error() string {
	return fmt.Sprintf("parameter %q missing from Bind values", e.Name)
}

// UnknownParameterError indicates Bind was given a name the template never
// declared (likely a typo).
type UnknownParameterError struct {
	Name string
}

func (e *UnknownParameterError) Error() string {
	return fmt.Sprintf("parameter %q was never declared in the template", e.Name)
}

// InvalidParameterValueError indicates a non-finite parameter value.
type InvalidParameterValueError struct {
	Name  string
	Value float64
}

func (e *InvalidParameterValueError) Error() string {
	return fmt.Sprintf("parameter %q has non-finite value %v", e.Name, e.Value)
}
