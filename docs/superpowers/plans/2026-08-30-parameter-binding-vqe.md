# Parameter Binding + VQE Driver Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a template-based symbolic parameter binding layer (`parameterized` package) and an H2 VQE driver (`algorithm` package), resolving the deferred half of backlog item 4.

**Architecture:** A `Template` holds ordered steps (fixed gates or parameter-driven factories); `Bind(Params)` materializes an ordinary `*circuit.Circuit`. `quantum.Gate` stays metadata-only; zero backend changes. The VQE driver binds/executes per iteration, computes exact energy via `quantum.Expectation`, and optimizes with parameter-shift gradients.

**Tech Stack:** Go, stdlib only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`

## Global Constraints

- Go module `github.com/pjbaur/quantum`, stdlib only — no new external deps.
- Typed-error taxonomy: exported struct types with fields + `Error() string`, style of `quantum/errortypes.go`.
- CI coverage floor is 50.0 — every task ships real tests; no thin wrappers without coverage.
- `gofmt` clean, `go vet ./...` clean, `go test -race ./...` clean at every commit.
- Commit messages: conventional style (`feat:`, `test:`, `docs:` …), Co-Authored-By trailer per repo convention.
- Import direction: `parameterized` → `circuit`, `gates`, `quantum`. `algorithm` → `parameterized`, `state`, `circuit`, `gates`, `quantum`. Nothing imports `parameterized` except `algorithm` and tests.

## Spec Deviations (flagged to user, approved with plan)

1. `NewTemplate(numQubits int)` takes the qubit count (spec showed no-arg).
   `Bind` must call `circuit.New(numQubits)`; an explicit count mirrors
   `circuit.New` and lets `AddParamGate`/`AddGate` reject out-of-range targets
   at declaration time instead of on every bind.
2. `Template.Steps()` from the spec is dropped (no consumer); replaced by
   `Template.NumQubits() int`, which VQE needs for `state.New`.

---

### Task 1: `parameterized` package — Template, Bind, factories, errors

**Files:**
- Create: `parameterized/parameterized.go`
- Create: `parameterized/parameterized_test.go`

**Interfaces:**
- Consumes: `circuit.New(numQubits int) (*Circuit, error)`, `(*Circuit).AddGate(gate quantum.Gate, targets ...int) error`; `gates.NewRx/NewRy/NewRz/NewPhase(theta float64) *MatrixGate`; `quantum.Gate`, `quantum.QubitsOutOfRangeError{Index, MaxIndex int}`.
- Produces (Task 3/5 depend on these):
  - `type Params map[string]float64`
  - `type Factory func(value float64) quantum.Gate`
  - `func NewTemplate(numQubits int) *Template`
  - `func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error`
  - `func (t *Template) AddGate(gate quantum.Gate, targets ...int) error`
  - `func (t *Template) ParamNames() []string`
  - `func (t *Template) NumQubits() int`
  - `func (t *Template) Bind(values Params) (*circuit.Circuit, error)`
  - `func Rx(value float64) quantum.Gate` (same shape: `Ry`, `Rz`, `Phase`)
  - `MissingParameterError{Name string}`, `UnknownParameterError{Name string}`, `InvalidParameterValueError{Name string; Value float64}`

- [ ] **Step 1: Write the failing tests**

Create `parameterized/parameterized_test.go`:

```go
package parameterized_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)

// newTestTemplate returns the canonical mixed template used across tests:
// Ry(theta) on 0, CNOT 0->1, Rx(phi) on 1.
func newTestTemplate() *parameterized.Template {
	t := parameterized.NewTemplate(2)
	if err := t.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		panic(err)
	}
	if err := t.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	if err := t.AddParamGate("phi", parameterized.Rx, 1); err != nil {
		panic(err)
	}
	return t
}

func TestBindMatchesManuallyBuiltCircuit(t *testing.T) {
	tmpl := newTestTemplate()

	bound, err := tmpl.Bind(parameterized.Params{"theta": 0.4, "phi": 1.1})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	manual, err := circuit.New(2)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := manual.AddGate(gates.NewRy(0.4), 0); err != nil {
		t.Fatalf("AddGate Ry: %v", err)
	}
	if err := manual.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("AddGate CNOT: %v", err)
	}
	if err := manual.AddGate(gates.NewRx(1.1), 1); err != nil {
		t.Fatalf("AddGate Rx: %v", err)
	}

	// Same output state on dense and sparse backends.
	denseA, _ := state.New(2)
	denseB, _ := state.New(2)
	if err := bound.Execute(denseA); err != nil {
		t.Fatalf("bound.Execute: %v", err)
	}
	if err := manual.Execute(denseB); err != nil {
		t.Fatalf("manual.Execute: %v", err)
	}
	for i := 0; i < 4; i++ {
		a, _ := denseA.Amplitude(i)
		b, _ := denseB.Amplitude(i)
		if a != b {
			t.Fatalf("amplitude %d: bound %v != manual %v", i, a, b)
		}
	}

	sparseA, _ := sparsestate.New(2)
	sparseB, _ := sparsestate.New(2)
	if err := bound.Execute(sparseA); err != nil {
		t.Fatalf("bound.Execute sparse: %v", err)
	}
	if err := manual.Execute(sparseB); err != nil {
		t.Fatalf("manual.Execute sparse: %v", err)
	}
	for i := 0; i < 4; i++ {
		a, _ := sparseA.Amplitude(i)
		b, _ := sparseB.Amplitude(i)
		if a != b {
			t.Fatalf("sparse amplitude %d: bound %v != manual %v", i, a, b)
		}
	}
}

func TestParamNamesFirstUseOrder(t *testing.T) {
	tmpl := parameterized.NewTemplate(3)
	if err := tmpl.AddParamGate("beta", parameterized.Rz, 2); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("alpha", parameterized.Rx, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("beta", parameterized.Ry, 1); err != nil {
		t.Fatal(err)
	}
	got := tmpl.ParamNames()
	want := []string{"beta", "alpha"}
	if len(got) != len(want) {
		t.Fatalf("ParamNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParamNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBindErrors(t *testing.T) {
	tmpl := newTestTemplate()

	tests := []struct {
		name    string
		values  parameterized.Params
		wantErr interface{} // pointer to expected error type
	}{
		{"missing phi", parameterized.Params{"theta": 0.1}, &parameterized.MissingParameterError{}},
		{"unknown key", parameterized.Params{"theta": 0.1, "phi": 0.2, "typo": 0.3}, &parameterized.UnknownParameterError{}},
		{"NaN theta", parameterized.Params{"theta": math.NaN(), "phi": 0.2}, &parameterized.InvalidParameterValueError{}},
		{"Inf phi", parameterized.Params{"theta": 0.1, "phi": math.Inf(1)}, &parameterized.InvalidParameterValueError{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tmpl.Bind(tt.values)
			if err == nil {
				t.Fatal("Bind succeeded, want error")
			}
			switch tt.wantErr.(type) {
			case *parameterized.MissingParameterError:
				var e *parameterized.MissingParameterError
				if !errors.As(err, &e) {
					t.Fatalf("err = %T (%v), want MissingParameterError", err, err)
				}
			case *parameterized.UnknownParameterError:
				var e *parameterized.UnknownParameterError
				if !errors.As(err, &e) {
					t.Fatalf("err = %T (%v), want UnknownParameterError", err, err)
				}
			case *parameterized.InvalidParameterValueError:
				var e *parameterized.InvalidParameterValueError
				if !errors.As(err, &e) {
					t.Fatalf("err = %T (%v), want InvalidParameterValueError", err, err)
				}
			}
		})
	}
}

func TestBindErrorMessagesCarryNames(t *testing.T) {
	_, err := newTestTemplate().Bind(parameterized.Params{"theta": 0.1})
	if err == nil || !strings.Contains(err.Error(), "phi") {
		t.Fatalf("err = %v, want message naming %q", err, "phi")
	}
}

func TestAddRejectsOutOfRangeTargets(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("theta", parameterized.Ry, 2); err == nil {
		t.Fatal("target 2 on 2-qubit template accepted, want range error")
	}
	if err := tmpl.AddGate(gates.NewCNOT(), 0, 2); err == nil {
		t.Fatal("fixed-gate target 2 accepted, want range error")
	}
}

func TestSameParamDrivesTwoGates(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("theta", parameterized.Rz, 1); err != nil {
		t.Fatal(err)
	}
	c, err := tmpl.Bind(parameterized.Params{"theta": 0.7})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	dense, _ := state.New(2)
	if err := c.Execute(dense); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	manual, _ := circuit.New(2)
	_ = manual.AddGate(gates.NewRy(0.7), 0)
	_ = manual.AddGate(gates.NewRz(0.7), 1)
	ref, _ := state.New(2)
	_ = manual.Execute(ref)
	for i := 0; i < 4; i++ {
		a, _ := dense.Amplitude(i)
		b, _ := ref.Amplitude(i)
		if a != b {
			t.Fatalf("amplitude %d: %v != %v", i, a, b)
		}
	}
}

func TestNumQubits(t *testing.T) {
	if got := parameterized.NewTemplate(5).NumQubits(); got != 5 {
		t.Fatalf("NumQubits() = %d, want 5", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./parameterized/`
Expected: FAIL — `no required module provides package` / undefined symbols (package does not exist yet).

- [ ] **Step 3: Write the implementation**

Create `parameterized/parameterized.go`:

```go
// Package parameterized provides circuit templates with named symbolic
// parameters. Bind materializes a template into an ordinary circuit, so
// variational loops can re-run the same structure at many parameter points
// without rebuilding gate lists by hand.
package parameterized

import (
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

// step is one template instruction: either a fixed gate (param == "") or a
// parameter-driven factory.
type step struct {
	param   string
	factory Factory
	gate    quantum.Gate
	targets []int
}

// Template is a circuit recipe with named parameter holes. A Template is
// safe for concurrent reads after all Add calls complete.
type Template struct {
	numQubits int
	steps     []step
	paramOrder []string
	seen      map[string]bool
}

// NewTemplate returns a template for circuits on numQubits qubits.
func NewTemplate(numQubits int) *Template {
	return &Template{
		numQubits: numQubits,
		seen:      make(map[string]bool),
	}
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

func (t *Template) checkTargets(targets []int) error {
	for _, target := range targets {
		if target < 0 || target >= t.numQubits {
			return &quantum.QubitsOutOfRangeError{Index: target, MaxIndex: t.numQubits - 1}
		}
	}
	return nil
}

// AddParamGate adds a gate built by factory from the named parameter's
// value at Bind time. The same name may drive several gates.
func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error {
	if factory == nil {
		return fmt.Errorf("parameter %q: factory must not be nil", name)
	}
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	if !t.seen[name] {
		t.seen[name] = true
		t.paramOrder = append(t.paramOrder, name)
	}
	t.steps = append(t.steps, step{param: name, factory: factory, targets: targets})
	return nil
}

// AddGate adds a fixed gate needing no parameter.
func (t *Template) AddGate(gate quantum.Gate, targets ...int) error {
	if gate == nil {
		return fmt.Errorf("fixed gate must not be nil")
	}
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	t.steps = append(t.steps, step{gate: gate, targets: targets})
	return nil
}

// Bind materializes the template into a circuit using values. Every declared
// parameter must be present and finite; unknown names are rejected so typos
// fail loudly instead of silently ignoring an angle.
func (t *Template) Bind(values Params) (*circuit.Circuit, error) {
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

// MissingParameterError indicates Bind was called without a declared name.
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./parameterized/ -v`
Expected: PASS, all tests.

- [ ] **Step 5: Lint and race-check**

Run: `gofmt -l parameterized/` then `go vet ./parameterized/` then `go test -race ./parameterized/`
Expected: empty output / OK / PASS.

- [ ] **Step 6: Commit**

```bash
git add parameterized/
git commit -m "feat(parameterized): template circuit binding with named parameters"
```

---

### Task 2: `algorithm.Hamiltonian` — Pauli-sum energy

**Files:**
- Create: `algorithm/hamiltonian.go`
- Create: `algorithm/hamiltonian_test.go`

**Interfaces:**
- Consumes: `quantum.Expectation(s QuantumState, axes []PauliAxis) (float64, error)`, `quantum.PauliI/X/Y/Z`, `state.New(numQubits int) (*State, error)`, `gates.NewRy/NewCNOT`.
- Produces (Tasks 3/5 depend):
  - `type Hamiltonian struct{ … }` (opaque)
  - `func NewHamiltonian() *Hamiltonian`
  - `func (h *Hamiltonian) AddTerm(coeff float64, axes ...quantum.PauliAxis) *Hamiltonian` — empty axes = identity term
  - `func (h *Hamiltonian) Energy(s quantum.QuantumState) (float64, error)`

- [ ] **Step 1: Write the failing tests**

Create `algorithm/hamiltonian_test.go`:

```go
package algorithm_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// bellState prepares (|00>+|11>)/sqrt2 on a fresh 2-qubit state.
func bellState(t *testing.T) *state.State {
	t.Helper()
	c, err := circuit.New(2)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("AddGate H: %v", err)
	}
	if err := c.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("AddGate CNOT: %v", err)
	}
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return s
}

func TestHamiltonianEnergyIdentityOnly(t *testing.T) {
	h := algorithm.NewHamiltonian().AddTerm(-1.25)
	s, _ := state.New(2)
	got, err := h.Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	if got != -1.25 {
		t.Fatalf("Energy = %v, want -1.25", got)
	}
}

func TestHamiltonianEnergyKnownValues(t *testing.T) {
	// Bell state: <ZZ>=1, <XX>=1, <YY>=-1.
	s := bellState(t)

	h := algorithm.NewHamiltonian().
		AddTerm(2.0, quantum.PauliZ, quantum.PauliZ).
		AddTerm(1.5, quantum.PauliX, quantum.PauliX).
		AddTerm(-0.5, quantum.PauliY, quantum.PauliY)
	got, err := h.Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	want := 2.0*1 + 1.5*1 + (-0.5)*(-1) // 4.0
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("Energy = %v, want %v", got, want)
	}
}

func TestHamiltonianEnergySingleQubit(t *testing.T) {
	// |+> on qubit 0 of a 2-qubit register: <XI> = 1.
	c, err := circuit.New(2)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("AddGate H: %v", err)
	}
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	h := algorithm.NewHamiltonian().AddTerm(3.0, quantum.PauliX, quantum.PauliI)
	got, err := h.Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	if math.Abs(got-3.0) > 1e-12 {
		t.Fatalf("Energy = %v, want 3.0", got)
	}
}

func TestHamiltonianEnergyPropagatesExpectationErrors(t *testing.T) {
	h := algorithm.NewHamiltonian().AddTerm(1.0, quantum.PauliZ)
	if _, err := h.Energy(nil); err == nil {
		t.Fatal("Energy(nil) succeeded, want error")
	}
	// Axis length mismatch: 3 axes on a 2-qubit state.
	s, _ := state.New(2)
	h2 := algorithm.NewHamiltonian().AddTerm(1.0, quantum.PauliZ, quantum.PauliZ, quantum.PauliZ)
	if _, err := h2.Energy(s); err == nil {
		t.Fatal("Energy with 3 axes on 2 qubits succeeded, want error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./algorithm/ -run Hamiltonian`
Expected: FAIL — `undefined: algorithm.NewHamiltonian`.

- [ ] **Step 3: Write the implementation**

Create `algorithm/hamiltonian.go`:

```go
package algorithm

import (
	"github.com/pjbaur/quantum/quantum"
)

// HamiltonianTerm is one weighted Pauli string.
type HamiltonianTerm struct {
	Coeff float64
	Axes  []quantum.PauliAxis
}

// Hamiltonian is a sum of weighted Pauli strings:
//
//	H = sum_i c_i * P_i
//
// Terms are stored in insertion order so Energy's floating-point summation
// is deterministic for a given construction sequence.
type Hamiltonian struct {
	terms []HamiltonianTerm
}

// NewHamiltonian returns an empty Hamiltonian (the zero operator).
func NewHamiltonian() *Hamiltonian {
	return &Hamiltonian{}
}

// AddTerm appends coeff * (axes as a Pauli string) and returns h for
// chaining. Empty axes denote the identity, contributing coeff directly.
// Coefficients are not validated: NaN and Inf flow into Energy results,
// where they surface as non-finite energies the caller can detect.
func (h *Hamiltonian) AddTerm(coeff float64, axes ...quantum.PauliAxis) *Hamiltonian {
	h.terms = append(h.terms, HamiltonianTerm{Coeff: coeff, Axes: axes})
	return h
}

// Energy returns the exact expectation value <psi|H|psi> computed term by
// term via quantum.Expectation. Identity terms (no axes) add their
// coefficient directly.
func (h *Hamiltonian) Energy(s quantum.QuantumState) (float64, error) {
	total := 0.0
	for _, term := range h.terms {
		if len(term.Axes) == 0 {
			total += term.Coeff
			continue
		}
		value, err := quantum.Expectation(s, term.Axes)
		if err != nil {
			return 0, err
		}
		total += term.Coeff * value
	}
	return total, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./algorithm/ -run Hamiltonian -v`
Expected: PASS.

- [ ] **Step 5: Lint and race-check**

Run: `gofmt -l algorithm/` && `go vet ./algorithm/` && `go test -race ./algorithm/ -run Hamiltonian`
Expected: clean / OK / PASS.

- [ ] **Step 6: Commit**

```bash
git add algorithm/hamiltonian.go algorithm/hamiltonian_test.go
git commit -m "feat(algorithm): Pauli-sum Hamiltonian with exact Energy"
```

---

### Task 3: H2 Hamiltonian, H2 ansatz, and test-only Jacobi verifier

**Files:**
- Create: `algorithm/h2.go`
- Create: `algorithm/h2_test.go`
- Create: `algorithm/jacobi_test.go` (test-only eigensolver helper)

**Interfaces:**
- Consumes: Task 2's `Hamiltonian` API; Task 1's `parameterized.NewTemplate/AddParamGate/AddGate/Ry`; `gates.NewCNOT()`; `quantum.PauliI/X/Y/Z`.
- Produces (Task 5 depends):
  - `func H2Hamiltonian() *Hamiltonian`
  - `func H2Ansatz() *parameterized.Template` — Ry("theta") on qubit 0, CNOT(0→1)
  - Test-only: `func h2GroundEnergy(t testing.TB) float64` (or equivalent helper) in `jacobi_test.go`

- [ ] **Step 1: Write the Jacobi test helper**

Create `algorithm/jacobi_test.go` (package `algorithm_test`):

```go
package algorithm_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/quantum"
)

// pauliMatrix returns the 2x2 matrix for a Pauli axis.
func pauliMatrix(axis quantum.PauliAxis) [][]float64 {
	switch axis {
	case quantum.PauliI:
		return [][]float64{{1, 0}, {0, 1}}
	case quantum.PauliX:
		return [][]float64{{0, 1}, {1, 0}}
	case quantum.PauliY:
		return [][]float64{{0, -1}, {1, 0}} // real representation of Y (up to global phase per factor); Y x Y products stay real
	case quantum.PauliZ:
		return [][]float64{{1, 0}, {0, -1}}
	}
	return nil
}

// pauliStringMatrix builds the 2^n x 2^n real matrix for axes, where axes[i]
// acts on qubit i. Qubit 0 is the least significant bit (index bit 0),
// matching quantum.Expectation's convention.
func pauliStringMatrix(axes []quantum.PauliAxis) [][]float64 {
	size := 1
	for range axes {
		size *= 2
	}
	m := make([][]float64, size)
	for i := range m {
		m[i] = make([]float64, size)
	}
	for row := 0; row < size; row++ {
		for col := 0; col < size; col++ {
			prod := 1.0
			for q, axis := range axes {
				pm := pauliMatrix(axis)
				// Matrix element of a single-qubit Pauli between basis states.
				rowBit := (row >> q) & 1
				colBit := (col >> q) & 1
				prod *= pm[rowBit][colBit]
			}
			m[row][col] = prod
		}
	}
	return m
}

// jacobiMinEigenvalue computes the smallest eigenvalue of a small real
// symmetric matrix via cyclic Jacobi rotations. Test-only verifier: the
// H2 ground energy target must come from diagonalizing the same Pauli sum
// the simulator evaluates, never from a hand-copied constant.
func jacobiMinEigenvalue(t testing.TB, a [][]float64) float64 {
	t.Helper()
	n := len(a)
	// Work on a copy.
	m := make([][]float64, n)
	for i := range m {
		m[i] = append([]float64(nil), a[i]...)
	}
	for sweep := 0; sweep < 100; sweep++ {
		off := 0.0
		for p := 0; p < n; p++ {
			for q := p + 1; q < n; q++ {
				off += m[p][q] * m[p][q]
			}
		}
		if off < 1e-24 {
			break
		}
		for p := 0; p < n; p++ {
			for q := p + 1; q < n; q++ {
				if math.Abs(m[p][q]) < 1e-15 {
					continue
				}
				theta := (m[q][q] - m[p][p]) / (2 * m[p][q])
				tSign := 1.0
				if theta < 0 {
					tSign = -1.0
				}
				tval := tSign / (theta*tSign + math.Sqrt(theta*theta+1))
				c := 1 / math.Sqrt(tval*tval+1)
				s := tval * c
				for k := 0; k < n; k++ {
					mkp, mkq := m[k][p], m[k][q]
					m[k][p] = c*mkp - s*mkq
					m[k][q] = s*mkp + c*mkq
				}
				for k := 0; k < n; k++ {
					mpk, mqk := m[p][k], m[q][k]
					m[p][k] = c*mpk - s*mqk
					m[q][k] = s*mpk + c*mqk
				}
			}
		}
	}
	min := m[0][0]
	for i := 1; i < n; i++ {
		if m[i][i] < min {
			min = m[i][i]
		}
	}
	return min
}

// h2Matrix builds the 4x4 real-symmetric H2 matrix from the same Pauli
// terms the simulator evaluates (algorithm.H2Terms).
func h2Matrix(t testing.TB) [][]float64 {
	t.Helper()
	n := 4
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
	}
	for _, term := range algorithm.H2Terms() {
		if len(term.Axes) == 0 {
			for i := 0; i < n; i++ {
				m[i][i] += term.Coeff
			}
			continue
		}
		pm := pauliStringMatrix(term.Axes)
		for r := 0; r < n; r++ {
			for c := 0; c < n; c++ {
				m[r][c] += term.Coeff * pm[r][c]
			}
		}
	}
	return m
}

// h2GroundEnergy diagonalizes the H2 Hamiltonian's Pauli sum independently
// of the simulator path.
func h2GroundEnergy(t testing.TB) float64 {
	t.Helper()
	return jacobiMinEigenvalue(t, h2Matrix(t))
}
```

- [ ] **Step 2: Write the failing H2 tests**

Create `algorithm/h2_test.go`:

```go
package algorithm_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/parameterized"
)

func TestH2AnsatzShape(t *testing.T) {
	tmpl := algorithm.H2Ansatz()
	if tmpl.NumQubits() != 2 {
		t.Fatalf("NumQubits = %d, want 2", tmpl.NumQubits())
	}
	names := tmpl.ParamNames()
	if len(names) != 1 || names[0] != "theta" {
		t.Fatalf("ParamNames = %v, want [theta]", names)
	}
}

func TestH2GroundEnergyInLiteratureRange(t *testing.T) {
	ground := h2GroundEnergy(t)
	// Total H2 ground energy at R = 0.735 A is about -1.857 Ha. The Jacobi
	// verifier is authoritative; this range check catches a wrong coefficient
	// convention (wrong basis / missing nuclear term) by a wide margin.
	if math.Abs(ground-(-1.857)) > 0.3 {
		t.Fatalf("H2 ground energy = %v, want within 0.3 of -1.857 Ha; check the coefficient convention (O'Malley et al., PRA 93, 052337 (2016))", ground)
	}
}

func TestH2HamiltonianEnergyAtZeroParams(t *testing.T) {
	// theta = 0: ansatz is identity+CNOT, state stays |00>; energy is the
	// diagonal element H[0][0] (the g4 term has zero diagonal). Compare
	// against the Jacobi-built matrix's [0][0] entry.
	tmpl := algorithm.H2Ansatz()
	c, err := tmpl.Bind(parameterized.Params{"theta": 0})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := algorithm.H2Hamiltonian().Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	want := h2Matrix(t)[0][0]
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("Energy(|00>) = %v, want matrix[0][0] = %v", got, want)
	}
}
```

Imports for `h2_test.go`: `math`, `testing`, and the project packages `algorithm`, `parameterized`, `state`. `h2Matrix`/`h2GroundEnergy` come from `jacobi_test.go` in the same `algorithm_test` package.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./algorithm/ -run H2`
Expected: FAIL — `undefined: algorithm.H2Hamiltonian`.

- [ ] **Step 4: Write the implementation**

Create `algorithm/h2.go`:

```go
package algorithm

import (
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)

// H2 coefficients for the two-qubit reduced Hamiltonian at the equilibrium
// bond length R = 0.735 Angstrom, in Hartree. Source: O'Malley et al.,
// "Scalable Quantum Simulation of Molecular Energies", PRA 93, 052337
// (2016), XZX (parity) basis reduction. g0 includes the nuclear repulsion
// and electronic constants; the g4 term couples the two parity sectors.
//
// If TestH2GroundEnergyInLiteratureRange fails, the convention (which Pauli
// string carries g4, or the qubit ordering of Z terms) differs from the
// source's basis. Diagnose with the Jacobi verifier, not by tuning numbers.
const (
	h2G0 = -1.052373245772859
	h2G1 = 0.39793742484318045
	h2G2 = -0.39793742484318045
	h2G3 = -0.01128010425623538
	h2G4 = 0.18093119978423156
)

// HamiltonianTerm is one weighted Pauli string of a Hamiltonian.
type HamiltonianTerm struct {
	Coeff float64
	Axes  []quantum.PauliAxis
}

// H2Terms returns the H2 Pauli terms with axes ordered qubit 0 first,
// matching quantum.Expectation's convention.
func H2Terms() []HamiltonianTerm {
	return []HamiltonianTerm{
		{Coeff: h2G0},
		{Coeff: h2G1, Axes: []quantum.PauliAxis{quantum.PauliZ, quantum.PauliI}},
		{Coeff: h2G2, Axes: []quantum.PauliAxis{quantum.PauliI, quantum.PauliZ}},
		{Coeff: h2G3, Axes: []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ}},
		{Coeff: h2G4, Axes: []quantum.PauliAxis{quantum.PauliY, quantum.PauliY}},
	}
}

// H2Hamiltonian returns the two-qubit reduced H2 Hamiltonian at R = 0.735 A.
func H2Hamiltonian() *Hamiltonian {
	h := NewHamiltonian()
	for _, term := range H2Terms() {
		h.AddTerm(term.Coeff, term.Axes...)
	}
	return h
}

// H2Ansatz returns the canonical UCC-inspired H2 ansatz template:
// Ry(theta) on qubit 0, then CNOT with control 0 and target 1.
func H2Ansatz() *parameterized.Template {
	t := parameterized.NewTemplate(2)
	if err := t.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		panic(err) // targets are static; an error here is a programming bug
	}
	if err := t.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	return t
}
```

If `TestH2GroundEnergyInLiteratureRange` fails with these constants: the XZX-basis convention places g4 on `Y0Y1` per the source's Eq. (4)-style reduction. Try the coupling term as `X0X1` (some reductions express the off-diagonal coupling in the X parity) — the Jacobi verifier decides empirically which convention yields ≈ −1.857 Ha. Do not weaken the range test; fix the term structure.

(Task 2 already defines the exported `HamiltonianTerm{Coeff, Axes}` — `h2.go` reuses it directly; no rename step.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./algorithm/ -run 'H2|Hamiltonian' -v`
Expected: PASS, including the literature-range check.

- [ ] **Step 6: Lint and race-check**

Run: `gofmt -l algorithm/` && `go vet ./algorithm/` && `go test -race ./algorithm/ -run 'H2|Hamiltonian'`
Expected: clean / OK / PASS.

- [ ] **Step 7: Commit**

```bash
git add algorithm/h2.go algorithm/h2_test.go algorithm/jacobi_test.go algorithm/hamiltonian.go algorithm/hamiltonian_test.go
git commit -m "feat(algorithm): H2 Hamiltonian, ansatz, and Jacobi ground-energy verifier"
```

---

### Task 4: Parameter-shift gradient (internal) with finite-difference test

**Files:**
- Create: `algorithm/vqe.go` (gradient + shared evaluate helper only in this task; the VQE loop lands in Task 5)
- Create: `algorithm/vqe_test.go` (gradient tests only in this task)

**Interfaces:**
- Consumes: Tasks 1–3 APIs; `state.New`; `(*circuit.Circuit).Execute`.
- Produces (Task 5 depends):
  - `func evaluate(h *Hamiltonian, t *parameterized.Template, params parameterized.Params) (float64, error)` — unexported
  - `func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)` — unexported; returns gradient per name and evaluation count

- [ ] **Step 1: Write the failing tests**

Create `algorithm/vqe_test.go` (package `algorithm`, white-box):

```go
package algorithm

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
)

// gradientTargetTemplate: Ry(a) q0, CNOT 0->1, Rx(b) q1, evaluated against
// the H2 Hamiltonian.
func gradientTargetTemplate() *parameterized.Template {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("a", parameterized.Ry, 0); err != nil {
		panic(err)
	}
	if err := tmpl.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	if err := tmpl.AddParamGate("b", parameterized.Rx, 1); err != nil {
		panic(err)
	}
	return tmpl
}

func TestParameterShiftMatchesFiniteDifference(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate()

	for _, base := range []float64{-1.2, -0.3, 0.0, 0.45, 1.7} {
		params := parameterized.Params{"a": base, "b": base/2 + 0.1}
		grad, evals, err := parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())
		if err != nil {
			t.Fatalf("parameterShiftGradient(base=%v): %v", base, err)
		}
		if evals != 4 { // two parameters, two evaluations each
			t.Fatalf("evals = %d, want 4", evals)
		}
		for _, name := range []string{"a", "b"} {
			const hstep = 1e-6
			plus := parameterized.Params{"a": params["a"], "b": params["b"]}
			minus := parameterized.Params{"a": params["a"], "b": params["b"]}
			plus[name] += hstep
			minus[name] -= hstep
			ePlus, err := evaluate(h, tmpl, plus)
			if err != nil {
				t.Fatalf("evaluate plus: %v", err)
			}
			eMinus, err := evaluate(h, tmpl, minus)
			if err != nil {
				t.Fatalf("evaluate minus: %v", err)
			}
			fd := (ePlus - eMinus) / (2 * hstep)
			if math.Abs(grad[name]-fd) > 1e-6 {
				t.Fatalf("base=%v param=%s: parameter-shift %v != finite-diff %v", base, name, grad[name], fd)
			}
		}
	}
}

func TestEvaluateIsDeterministicAndFinite(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := H2Ansatz()
	e1, err := evaluate(h, tmpl, parameterized.Params{"theta": 0.2})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	e2, err := evaluate(h, tmpl, parameterized.Params{"theta": 0.2})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if e1 != e2 {
		t.Fatalf("evaluate not deterministic: %v vs %v", e1, e2)
	}
	if math.IsNaN(e1) || math.IsInf(e1, 0) {
		t.Fatalf("evaluate = %v, want finite", e1)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./algorithm/ -run 'ParameterShift|Evaluate'`
Expected: FAIL — `undefined: parameterShiftGradient`.

- [ ] **Step 3: Write the implementation**

Create `algorithm/vqe.go`:

```go
package algorithm

import (
	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)

// evaluate binds params, executes on a fresh dense state, and returns the
// exact energy. One variational cost unit.
func evaluate(h *Hamiltonian, t *parameterized.Template, params parameterized.Params) (float64, error) {
	c, err := t.Bind(params)
	if err != nil {
		return 0, err
	}
	s, err := state.New(t.NumQubits())
	if err != nil {
		return 0, err
	}
	if err := c.Execute(s); err != nil {
		return 0, err
	}
	return h.Energy(s)
}

// parameterShiftGradient computes dE/dtheta per parameter via the exact
// parameter-shift rule: dE/dtheta = (E(theta+pi/2) - E(theta-pi/2)) / 2.
// Valid because every parameterized factory is a generators-of-Pauli
// rotation (Rx/Ry/Rz/Phase in parameterized all are). Returns the gradient
// keyed by name plus the number of energy evaluations consumed.
func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error) {
	grad := make(map[string]float64, len(names))
	evals := 0
	for _, name := range names {
		plus := parameterized.Params{}
		minus := parameterized.Params{}
		for k, v := range params {
			plus[k] = v
			minus[k] = v
		}
		plus[name] += math.Pi / 2
		minus[name] -= math.Pi / 2
		ePlus, err := evaluate(h, t, plus)
		if err != nil {
			return nil, evals, err
		}
		eMinus, err := evaluate(h, t, minus)
		if err != nil {
			return nil, evals, err
		}
		evals += 2
		grad[name] = (ePlus - eMinus) / 2
	}
	return grad, evals, nil
}
```

Add `math` to imports. (It is used by `math.Pi`.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./algorithm/ -run 'ParameterShift|Evaluate' -v`
Expected: PASS — parameter-shift matches finite difference to 1e-6 at all five base points.

- [ ] **Step 5: Lint and race-check**

Run: `gofmt -l algorithm/` && `go vet ./algorithm/` && `go test -race ./algorithm/ -run 'ParameterShift|Evaluate'`
Expected: clean / OK / PASS.

- [ ] **Step 6: Commit**

```bash
git add algorithm/vqe.go algorithm/vqe_test.go
git commit -m "feat(algorithm): exact parameter-shift gradient for variational loops"
```

---

### Task 5: VQE driver

**Files:**
- Modify: `algorithm/vqe.go` (append driver)
- Modify: `algorithm/vqe_test.go` (append driver tests)

**Interfaces:**
- Consumes: Task 4's `evaluate`, `parameterShiftGradient`; Tasks 1–3 APIs.
- Produces:
  - `type VQEOptions struct { InitialParams parameterized.Params; StepSize float64; MaxIterations int; Tolerance float64 }`
  - `type VQEResult struct { Energy float64; Params parameterized.Params; Iterations int; Evaluations int; Converged bool }`
  - `type InvalidVQEInputError struct { Reason string }`
  - `func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error)`

- [ ] **Step 1: Write the failing tests**

Append to `algorithm/vqe_test.go`:

```go
func TestVQEConvergesToH2GroundState(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := H2Ansatz()

	res, err := VQE(h, tmpl, VQEOptions{
		InitialParams: parameterized.Params{"theta": 0.1},
	})
	if err != nil {
		t.Fatalf("VQE: %v", err)
	}
	ground := h2GroundEnergy(t)
	if math.Abs(res.Energy-ground) > 1e-6 {
		t.Fatalf("VQE energy %v, want ground %v (Converged=%v, iters=%d)", res.Energy, ground, res.Converged, res.Iterations)
	}
	if !res.Converged {
		t.Fatal("VQE did not converge within default MaxIterations")
	}
	// Evaluation accounting: initial energy + per-iteration (1 accept eval +
	// 2*params gradient evals), plus one eval per reverted step.
	if res.Evaluations <= 0 || res.Evaluations > 3*res.Iterations+2*len(res.Params)+1 {
		t.Fatalf("Evaluations = %d implausible for %d iterations", res.Evaluations, res.Iterations)
	}
}

func TestVQEDefaultsAndExplicitOptions(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := H2Ansatz()

	// Zero-value options must work: defaults for everything.
	res, err := VQE(h, tmpl, VQEOptions{})
	if err != nil {
		t.Fatalf("VQE zero opts: %v", err)
	}
	if !res.Converged {
		t.Fatal("zero-value options failed to converge")
	}
	if _, ok := res.Params["theta"]; !ok {
		t.Fatalf("res.Params = %v, want theta present", res.Params)
	}
}

func TestVQEInvalidInputs(t *testing.T) {
	tmpl := H2Ansatz()
	if _, err := VQE(nil, tmpl, VQEOptions{}); err == nil {
		t.Fatal("VQE(nil Hamiltonian) succeeded, want error")
	}
	h := H2Hamiltonian()
	if _, err := VQE(h, nil, VQEOptions{}); err == nil {
		t.Fatal("VQE(nil template) succeeded, want error")
	}
	var e *InvalidVQEInputError
	_, err := VQE(h, tmpl, VQEOptions{InitialParams: parameterized.Params{"nope": 1.0}})
	if !errors.As(err, &e) {
		t.Fatalf("err = %v, want InvalidVQEInputError (unknown initial param)", err)
	}
}

func TestVQERespectsMaxIterations(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := H2Ansatz()
	res, err := VQE(h, tmpl, VQEOptions{MaxIterations: 2, Tolerance: 1e-15})
	if err != nil {
		t.Fatalf("VQE: %v", err)
	}
	if res.Converged {
		t.Fatal("Converged=true with MaxIterations=2 and unreachable tolerance")
	}
	if res.Iterations > 2 {
		t.Fatalf("Iterations = %d, want <= 2", res.Iterations)
	}
}
```

Package layout: the gradient tests (Task 4) live white-box in `vqe_test.go` (`package algorithm`); these driver tests go in a separate file `algorithm/vqe_driver_test.go` with `package algorithm_test`, since `h2GroundEnergy` lives in `jacobi_test.go` under `algorithm_test` and the two Go test packages cannot see each other's identifiers. Imports for `vqe_driver_test.go`: `errors`, `math`, `testing`, plus `algorithm` and `parameterized`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./algorithm/ -run VQE`
Expected: FAIL — `undefined: VQE`.

- [ ] **Step 3: Write the implementation**

Append to `algorithm/vqe.go`:

```go
// VQEOptions configures the VQE loop. Zero values select defaults.
type VQEOptions struct {
	// InitialParams names starting angles. Missing declared parameters
	// default to 0. Unknown names are rejected.
	InitialParams parameterized.Params
	// StepSize is the initial gradient-descent step (default 0.3).
	StepSize float64
	// MaxIterations caps the loop (default 200).
	MaxIterations int
	// Tolerance is the convergence threshold on |delta E| between
	// consecutive iterations (default 1e-10).
	Tolerance float64
}

// VQEResult reports the outcome of a VQE run.
type VQEResult struct {
	Energy      float64
	Params      parameterized.Params
	Iterations  int
	Evaluations int
	Converged   bool
}

// InvalidVQEInputError indicates a malformed VQE invocation.
type InvalidVQEInputError struct {
	Reason string
}

func (e *InvalidVQEInputError) Error() string {
	return "invalid VQE input: " + e.Reason
}

// VQE minimizes <psi(params)|H|psi(params)> with parameter-shift gradients
// and plain descent. Each iteration binds, executes on a fresh dense state,
// evaluates exactly, shifts every parameter, and steps. A step that raises
// the energy reverts the parameters and halves the step size (floor 1e-6).
// The loop stops when |delta E| < Tolerance (Converged) or MaxIterations.
func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error) {
	if h == nil {
		return nil, &InvalidVQEInputError{Reason: "Hamiltonian must not be nil"}
	}
	if t == nil {
		return nil, &InvalidVQEInputError{Reason: "template must not be nil"}
	}

	step := opts.StepSize
	if step == 0 {
		step = 0.3
	}
	maxIter := opts.MaxIterations
	if maxIter == 0 {
		maxIter = 200
	}
	tol := opts.Tolerance
	if tol == 0 {
		tol = 1e-10
	}

	names := t.ParamNames()
	params := parameterized.Params{}
	for _, name := range names {
		params[name] = 0
	}
	for name, value := range opts.InitialParams {
		if _, ok := params[name]; !ok {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q is not declared in the template", name)}
		}
		params[name] = value
	}

	evals := 0
	energy, err := evaluate(h, t, params)
	if err != nil {
		return nil, err
	}
	evals++

	converged := false
	accepted := 0
	for iter := 0; iter < maxIter; iter++ {
		grad, gradEvals, err := parameterShiftGradient(h, t, params, names)
		if err != nil {
			return nil, err
		}
		evals += gradEvals

		steps := parameterized.Params{}
		for _, name := range names {
			steps[name] = params[name] - step*grad[name]
		}
		newEnergy, err := evaluate(h, t, steps)
		if err != nil {
			return nil, err
		}
		evals++

		if newEnergy > energy {
			// Revert; shrink the step and retry next iteration.
			step /= 2
			if step < 1e-6 {
				step = 1e-6
			}
			continue
		}

		accepted++
		delta := math.Abs(newEnergy - energy)
		params, energy = steps, newEnergy
		if delta < tol {
			converged = true
			break
		}
	}

	return &VQEResult{
		Energy:      energy,
		Params:      params,
		Iterations:  accepted,
		Evaluations: evals,
		Converged:   converged,
	}, nil
}
```

Imports for `vqe.go` after this task: `fmt`, `math`, plus the project packages already present from Task 4.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./algorithm/ -run 'VQE|ParameterShift|Evaluate|H2|Hamiltonian' -v`
Expected: PASS. VQE converges to the Jacobi ground energy within 1e-6.

- [ ] **Step 5: Full package verification**

Run: `go test ./algorithm/ -v && go vet ./algorithm/ && gofmt -l algorithm/`
Expected: all PASS / OK / empty.

- [ ] **Step 6: Commit**

```bash
git add algorithm/vqe.go algorithm/vqe_test.go algorithm/vqe_driver_test.go
git commit -m "feat(algorithm): VQE driver with parameter-shift descent"
```

---

### Task 6: Benchmark, ADR, backlog/example-ideas updates, full verification

**Files:**
- Create: `algorithm/vqe_bench_test.go`
- Create: `docs/adr/0010-parameter-binding-template.md`
- Modify: `docs/enhancement-backlog-2026-08-27.md` (item 4 addendum)
- Modify: `docs/example-ideas.md` (stage 7 capability note)

**Interfaces:**
- Consumes: all prior tasks; `quantum.Resetter` (`Reset() error`), `state.New`.

- [ ] **Step 1: Write the benchmark**

Create `algorithm/vqe_bench_test.go` (package `algorithm_test`):

```go
package algorithm_test

import (
	"testing"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)

// BenchmarkVQEIteration measures one bind + execute + energy evaluation,
// the inner cost unit of the driver.
func BenchmarkVQEIteration(b *testing.B) {
	h := algorithm.H2Hamiltonian()
	tmpl := algorithm.H2Ansatz()
	params := parameterized.Params{"theta": 0.3}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params["theta"] += 0.01 // vary so binds differ
		c, err := tmpl.Bind(params)
		if err != nil {
			b.Fatal(err)
		}
		s, err := state.New(2)
		if err != nil {
			b.Fatal(err)
		}
		if err := c.Execute(s); err != nil {
			b.Fatal(err)
		}
		if _, err := h.Energy(s); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManualRebuild measures the pre-binding variational pattern from
// backlog item 4: rebuild gates by hand, Reset, re-Execute on one state.
func BenchmarkManualRebuild(b *testing.B) {
	h := algorithm.H2Hamiltonian()
	s, err := state.New(2)
	if err != nil {
		b.Fatal(err)
	}
	theta := 0.3
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		theta += 0.01
		c, err := circuit.New(2)
		if err != nil {
			b.Fatal(err)
		}
		if err := c.AddGate(gates.NewRy(theta), 0); err != nil {
			b.Fatal(err)
		}
		if err := c.AddGate(gates.NewCNOT(), 0, 1); err != nil {
			b.Fatal(err)
		}
		if err := s.Reset(); err != nil {
			b.Fatal(err)
		}
		if err := c.Execute(s); err != nil {
			b.Fatal(err)
		}
		if _, err := h.Energy(s); err != nil {
			b.Fatal(err)
		}
	}
}
```

Verify `state.State` implements `Reset() error` (the `quantum.Resetter` capability from backlog item 4) before relying on it; if the method name differs, check `quantum/interfaces.go` and use the real one.

- [ ] **Step 2: Run benchmarks and record numbers**

Run: `go test ./algorithm/ -bench 'VQEIteration|ManualRebuild' -benchtime 2s -run '^$'`
Expected: both run; note ns/op in the ADR. Bind overhead should be small relative to execute+energy at 2 qubits; the point is evidence, not a win.

- [ ] **Step 3: Write ADR-0010**

Create `docs/adr/0010-parameter-binding-template.md`:

```markdown
# ADR-0010: Parameter Binding via Template Materialization

Date: 2026-08-30
Status: Accepted

## Context

Backlog item 4 deferred symbolic parameter binding until a real
variational driver existed. The VQE driver (H2, stage 7 of the
example-ideas ladder) is that driver. Two mechanisms were considered:

1. Symbolic gate types: gates carrying unevaluated angles, resolved at
   execution time.
2. Template materialization: a recipe object that produces concrete
   circuits from parameter values.

## Decision

Template materialization (`parameterized.Template` + `Bind`).

Symbolic gates would force every backend and every execution path to
resolve parameters before `Matrix()` can return numbers — touching the
metadata-only `quantum.Gate` contract that ADR-0001 just cleaned up, and
every backend for one consumer. Templates live above the core: `Bind`
emits ordinary circuits, backends never know parameters existed.

Tradeoff: each `Bind` re-allocates the gate list and re-bakes rotation
matrices. Benchmark `BenchmarkVQEIteration` vs `BenchmarkManualRebuild`
(records: fill in from Step 2) shows this is negligible next to the
O(2^n) state execution that dominates every variational iteration.

## Consequences

- `quantum.Gate`, `Registry`, and all backends unchanged.
- Parameter validation (missing/unknown/non-finite) fails at `Bind`,
  before any state is touched.
- Derived angles (`theta/2`) are out of scope: declare another parameter.
- QAOA (stage 8) can reuse the template unchanged.
```

Fill in the recorded benchmark numbers where indicated.

- [ ] **Step 4: Update backlog and example-ideas**

In `docs/enhancement-backlog-2026-08-27.md`, item 4: append after the existing "Done" note:

```markdown
  > **Addendum (2026-08-30)**: the deferred symbolic half is now done too —
  > `parameterized.Template` + `Bind` materializes circuits from named
  > parameter maps, with the H2 VQE driver as the consuming evidence.
  > See `docs/adr/0010-parameter-binding-template.md` and
  > `docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`.
```

In `docs/example-ideas.md` stage 7 "Capabilities required": annotate the
"Parameterized circuits (RX/RY/RZ with symbolic parameters)" bullet with
`(satisfied: parameterized.Template, 2026-08-30)` and same for
"Ability to run the same circuit many times with different parameters"
`(satisfied: Template.Bind + VQE driver, 2026-08-30)`.

- [ ] **Step 5: Full verification**

Run: `gofmt -l . && go vet ./... && go test ./... && go test -race ./...`
Expected: empty / OK / all PASS / all PASS.

- [ ] **Step 6: Commit**

```bash
git add algorithm/vqe_bench_test.go docs/adr/0010-parameter-binding-template.md docs/enhancement-backlog-2026-08-27.md docs/example-ideas.md
git commit -m "docs+bench: ADR-0010 template binding; close deferred half of backlog item 4"
```
