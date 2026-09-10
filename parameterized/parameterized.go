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
// not supported, whether by assignment, by passing or returning it, or by
// storing it in a slice, map, or struct field that is later copied. Like
// strings.Builder, a Template records the receiver of its first accepted
// declaration, and AddParamGate, AddGate, and Bind on a copy taken after
// that return an error, before any other check. A copy taken before any
// declaration is accepted shares nothing and is an independent template.
//
// The step list and the declared names live behind one pointer, allocated
// by the first accepted declaration, so every copy of a declared Template
// refers to the same declaration state rather than to its own slice
// headers over shared arrays. Only the variable that founded a state ever
// writes it, and its two lists grow together, so whichever state a value
// holds, ParamNames and ParamStepCounts describe the same declarations and
// Bind demands exactly the listed names: no sequence of by-value copies,
// copy-backs, and declarations can leave the two disagreeing or let Bind
// accept a binding that omits a declared name. The receiver check cannot
// tell a copy assigned back over the original from the original, and does
// not need to: the state it restores is one the same variable founded.
// That is the variable's current state, so the copy-back drops nothing,
// unless the variable was reset to an undeclared template in between (for
// example a = *NewTemplate(n)), which founds a second state on its next
// accepted declaration; a copy taken before such a reset restores the
// earlier state and drops what was declared into the later one, which is
// what assigning an older value means rather than a desync. NumQubits, ParamNames, and
// ParamStepCounts on a refused copy report that shared state, so they
// describe the template as it is now, declarations the original has made
// since the copy included. A Template is safe for concurrent reads after
// all Add calls complete.
type Template struct {
	// addr is the receiver of the first accepted declaration, set only by
	// AddParamGate and AddGate, so a by-value copy taken after that can be
	// told from the original, as strings.Builder does. Nil until then: a
	// copy of a template with no accepted declaration shares nothing.
	addr      *Template
	numQubits int
	// state holds every declaration. It is allocated together with addr,
	// by the first accepted declaration, so the zero value and a copy
	// taken before that carry nil and later get a state of their own.
	// Every copy taken after that carries this same pointer, which is what
	// keeps a copy assigned back over the original from restoring a view
	// of the lists that is inconsistent, or older than the state the
	// variable is currently on (backlog item 21). Resetting the variable
	// to an undeclared template founds a second state, and a copy taken
	// before the reset restores the first one; see Template.
	state *templateState
}

// templateState is a Template's declaration state: the steps in order and
// the declared names in first-use order, without duplicates. Every step
// with a factory names a parameter in paramOrder, and every name in
// paramOrder has at least one such step; AddParamGate maintains both in
// one call, and nothing else writes either list.
type templateState struct {
	steps      []step
	paramOrder []string
}

// names returns the declared names; nil on a nil state, which is a
// template with no accepted declaration.
func (s *templateState) names() []string {
	if s == nil {
		return nil
	}
	return s.paramOrder
}

// stepList returns the steps in order; nil on a nil state.
func (s *templateState) stepList() []step {
	if s == nil {
		return nil
	}
	return s.steps
}

// errCopiedTemplate is what AddParamGate, AddGate, and Bind return on a
// Template copied by value after an accepted declaration; see Template.
var errCopiedTemplate = errors.New("Template copied by value after a declaration; a declared Template is used in place or through a pointer, never by copy")

// checkNotCopied returns errCopiedTemplate when t is a by-value copy of a
// Template that had already accepted a declaration when it was copied.
// Such a copy shares the original's declaration state, which only the
// original may write; the two writers and Bind call this first. A copy
// assigned back over the original passes (its addr is the receiver again)
// and holds a state pointer that same variable founded, so what it
// restores is always internally consistent: it is the variable's current
// state unless the variable was reset to an undeclared template between
// the copy and the copy-back, in which case it restores the earlier of
// the variable's own states. See Template.
func (t *Template) checkNotCopied() error {
	if t.addr != nil && t.addr != t {
		return errCopiedTemplate
	}
	return nil
}

// pin records t as the writer of its declaration state and allocates that
// state on the first accepted declaration. The two writers call it after
// their argument checks pass and before their first mutation, so a
// rejected declaration leaves both fields untouched and the pin and the
// state come into being in the same call.
func (t *Template) pin() {
	t.addr = t
	if t.state == nil {
		t.state = &templateState{}
	}
}

// declared reports whether name is in the declared names. The list is the
// single record of the declared names, read through the state every copy
// shares, so the answer is the same for the original and for any copy
// (backlog item 21).
func (t *Template) declared(name string) bool {
	for _, n := range t.state.names() {
		if n == name {
			return true
		}
	}
	return false
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
	names := t.state.names()
	out := make([]string, len(names))
	copy(out, names)
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
	counts := make(map[string]int, len(t.state.names()))
	for _, s := range t.state.stepList() {
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
	t.pin()
	if !t.declared(name) {
		t.state.paramOrder = append(t.state.paramOrder, name)
	}
	t.state.steps = append(t.state.steps, step{param: name, factory: factory, targets: targets})
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
	t.pin()
	t.state.steps = append(t.state.steps, step{gate: gate, targets: targets})
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
	names := t.state.names()
	for _, name := range names {
		value, ok := values[name]
		if !ok {
			return nil, &MissingParameterError{Name: name}
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, &InvalidParameterValueError{Name: name, Value: value}
		}
	}
	// Every declared name is present and the declared names are distinct,
	// so values holds an undeclared key exactly when it has more keys than
	// the template has names; the scan that names one runs only then.
	if len(values) != len(names) {
		for name := range values {
			if !t.declared(name) {
				return nil, &UnknownParameterError{Name: name}
			}
		}
	}

	c, err := circuit.New(t.numQubits)
	if err != nil {
		return nil, err
	}
	for _, s := range t.state.stepList() {
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
