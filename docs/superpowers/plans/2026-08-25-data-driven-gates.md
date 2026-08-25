# Data-Driven Gates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the eight concrete gate structs with one data-driven `MatrixGate`, give `Registry`/`DecomposeSwap` a real consumer (a `gates` CLI demo), and delete the dead code flagged in the architectural issue.

**Architecture:** `gates.MatrixGate` (promoted from the dead `algorithm.matrixGate`) becomes the single gate implementation; built-ins are canonical matrix tables behind the existing constructor names. `gates.Builtin()` returns a preloaded `Registry`; a new CLI demo consumes `Builtin()`, `Names()`, `Lookup()`, and `DecomposeSwap`. `algorithm/algorithm.go` and the empty `measurement/` directory are deleted.

**Tech Stack:** Go (module `github.com/pjbaur/quantum`), standard `testing` package. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-08-25-data-driven-gates-design.md`

## Global Constraints

- Constructor names (`NewHadamard`, `NewPauliX`, `NewPauliY`, `NewPauliZ`, `NewS`, `NewT`, `NewCNOT`, `NewSwap`) and gate name strings (`"Hadamard"`, `"PauliX"`, `"PauliY"`, `"PauliZ"`, `"S"`, `"T"`, `"CNOT"`, `"SWAP"`) MUST NOT change.
- Matrix element expressions for built-ins must be copied verbatim from the current `gates/gates.go` (e.g. `1.0 / complex(math.Sqrt(2), 0)`, `complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))`) so values stay bit-identical — the sparse backend's `isCanonicalCNOT` fast path and existing tests depend on today's exact values.
- `MatrixGate.Matrix()` must return a deep copy (current gates build a fresh literal per call; callers may mutate the result).
- After every task: `go build ./... && go test ./...` green before committing.
- Commit messages: Conventional Commits, ending with the trailer `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
- All commands run from the repository root.

---

### Task 1: MatrixGate core

**Files:**
- Create: `gates/matrixgate.go`
- Create: `gates/matrixgate_test.go`

**Interfaces:**
- Consumes: nothing new (package `gates` already exists).
- Produces: `type MatrixGate struct` with `NewMatrixGate(name string, matrix [][]complex128) (*MatrixGate, error)`, methods `Name() string`, `Matrix() [][]complex128`, `NumQubits() int`, and unexported helper `copyMatrix(matrix [][]complex128) [][]complex128`. Tasks 2–4 rely on these exact signatures.

- [ ] **Step 1: Write the failing tests**

Create `gates/matrixgate_test.go`:

```go
package gates

import (
	"math/cmplx"
	"testing"
)

func TestNewMatrixGateValidation(t *testing.T) {
	valid2x2 := [][]complex128{{0, 1}, {1, 0}}
	cases := []struct {
		name     string
		gateName string
		matrix   [][]complex128
		wantErr  bool
	}{
		{"valid 2x2", "X", valid2x2, false},
		{"valid 4x4", "CZ", [][]complex128{
			{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1},
		}, false},
		{"empty name", "", valid2x2, true},
		{"nil matrix", "G", nil, true},
		{"1x1 matrix", "G", [][]complex128{{1}}, true},
		{"non-square", "G", [][]complex128{{1, 0}, {0}}, true},
		{"non power of two", "G", [][]complex128{
			{1, 0, 0}, {0, 1, 0}, {0, 0, 1},
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gate, err := NewMatrixGate(tc.gateName, tc.matrix)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NewMatrixGate(%q) succeeded, want error", tc.gateName)
				}
				if gate != nil {
					t.Errorf("NewMatrixGate(%q) returned non-nil gate with error", tc.gateName)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewMatrixGate(%q) failed: %v", tc.gateName, err)
			}
			if gate.Name() != tc.gateName {
				t.Errorf("Name() = %q, want %q", gate.Name(), tc.gateName)
			}
		})
	}
}

func TestMatrixGateNumQubits(t *testing.T) {
	twoByTwo, err := NewMatrixGate("X", [][]complex128{{0, 1}, {1, 0}})
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}
	if got := twoByTwo.NumQubits(); got != 1 {
		t.Errorf("2x2 NumQubits() = %d, want 1", got)
	}

	fourByFour, err := NewMatrixGate("CZ", [][]complex128{
		{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1},
	})
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}
	if got := fourByFour.NumQubits(); got != 2 {
		t.Errorf("4x4 NumQubits() = %d, want 2", got)
	}
}

func TestMatrixGateMatrixIsDeepCopy(t *testing.T) {
	input := [][]complex128{{0, 1}, {1, 0}}
	gate, err := NewMatrixGate("X", input)
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}

	// Mutating the caller's input after construction must not affect the gate.
	input[0][0] = 99
	if got := gate.Matrix()[0][0]; got != 0 {
		t.Errorf("gate matrix affected by input mutation: got %v, want 0", got)
	}

	// Mutating a returned matrix must not affect later calls.
	first := gate.Matrix()
	first[0][1] = 42
	if got := gate.Matrix()[0][1]; cmplx.Abs(got-1) > 0 {
		t.Errorf("gate matrix affected by returned-slice mutation: got %v, want 1", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./gates/ -run 'TestNewMatrixGateValidation|TestMatrixGateNumQubits|TestMatrixGateMatrixIsDeepCopy'`
Expected: FAIL — build error `undefined: NewMatrixGate` (feature missing).

- [ ] **Step 3: Write minimal implementation**

Create `gates/matrixgate.go`:

```go
package gates

import (
	"errors"
	"fmt"
	"math/bits"
)

// MatrixGate is a quantum gate defined entirely by its unitary matrix.
type MatrixGate struct {
	name   string
	matrix [][]complex128
	qubits int
}

// NewMatrixGate validates the matrix and returns a gate.
// The matrix must be non-empty, square, and its dimension a power of two >= 2.
func NewMatrixGate(name string, matrix [][]complex128) (*MatrixGate, error) {
	if name == "" {
		return nil, errors.New("gate name must not be empty")
	}
	size := len(matrix)
	if size < 2 {
		return nil, errors.New("matrix must be at least 2x2")
	}
	if size&(size-1) != 0 {
		return nil, fmt.Errorf("matrix size %d is not a power of two", size)
	}
	for _, row := range matrix {
		if len(row) != size {
			return nil, errors.New("matrix must be square")
		}
	}

	return &MatrixGate{
		name:   name,
		matrix: copyMatrix(matrix),
		qubits: bits.Len(uint(size)) - 1,
	}, nil
}

// Name returns the name of the gate.
func (g *MatrixGate) Name() string {
	return g.name
}

// Matrix returns a copy of the matrix representation of the gate.
func (g *MatrixGate) Matrix() [][]complex128 {
	return copyMatrix(g.matrix)
}

// NumQubits returns the number of qubits the gate operates on.
func (g *MatrixGate) NumQubits() int {
	return g.qubits
}

func copyMatrix(matrix [][]complex128) [][]complex128 {
	out := make([][]complex128, len(matrix))
	for i, row := range matrix {
		out[i] = append([]complex128(nil), row...)
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./gates/`
Expected: PASS (all gates tests, new and pre-existing).

- [ ] **Step 5: Full build and test, then commit**

Run: `go build ./... && go test ./...`
Expected: all packages pass.

```bash
git add gates/matrixgate.go gates/matrixgate_test.go
git commit -m "feat(gates): add MatrixGate data-driven gate type

Promoted from the unused algorithm.matrixGate per the data-driven
gates design (docs/superpowers/specs/2026-08-25-data-driven-gates-design.md).
Matrix() deep-copies so callers cannot corrupt the shared table.

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: Built-in gates become data

**Files:**
- Modify: `gates/gates.go` (full rewrite — replace all eight structs)
- Create: `gates/gates_test.go`

**Interfaces:**
- Consumes: `NewMatrixGate`, `*MatrixGate` from Task 1.
- Produces: `NewHadamard() *MatrixGate`, `NewPauliX() *MatrixGate`, `NewPauliY() *MatrixGate`, `NewPauliZ() *MatrixGate`, `NewS() *MatrixGate`, `NewT() *MatrixGate`, `NewCNOT() *MatrixGate`, `NewSwap() *MatrixGate`, and unexported `mustGate(name string, matrix [][]complex128) *MatrixGate`. Task 3 relies on the constructors.

- [ ] **Step 1: Write the failing golden tests**

Create `gates/gates_test.go`. The expected matrices are copied verbatim from the current concrete types so the test locks today's exact values in place:

```go
package gates

import (
	"math"
	"testing"
)

// TestBuiltinGateGoldenValues locks the name and exact matrix of every
// built-in gate to the values the concrete types produced before the
// data-driven rewrite.
func TestBuiltinGateGoldenValues(t *testing.T) {
	cases := []struct {
		gate   *MatrixGate
		name   string
		matrix [][]complex128
	}{
		{NewHadamard(), "Hadamard", [][]complex128{
			{1.0 / complex(math.Sqrt(2), 0), 1.0 / complex(math.Sqrt(2), 0)},
			{1.0 / complex(math.Sqrt(2), 0), -1.0 / complex(math.Sqrt(2), 0)},
		}},
		{NewPauliX(), "PauliX", [][]complex128{
			{0, 1},
			{1, 0},
		}},
		{NewPauliY(), "PauliY", [][]complex128{
			{0, complex(0, -1)},
			{complex(0, 1), 0},
		}},
		{NewPauliZ(), "PauliZ", [][]complex128{
			{1, 0},
			{0, -1},
		}},
		{NewS(), "S", [][]complex128{
			{1, 0},
			{0, complex(0, 1)},
		}},
		{NewT(), "T", [][]complex128{
			{1, 0},
			{0, complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))},
		}},
		{NewCNOT(), "CNOT", [][]complex128{
			{1, 0, 0, 0},
			{0, 1, 0, 0},
			{0, 0, 0, 1},
			{0, 0, 1, 0},
		}},
		{NewSwap(), "SWAP", [][]complex128{
			{1, 0, 0, 0},
			{0, 0, 1, 0},
			{0, 1, 0, 0},
			{0, 0, 0, 1},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.gate.Name() != tc.name {
				t.Errorf("Name() = %q, want %q", tc.gate.Name(), tc.name)
			}
			got := tc.gate.Matrix()
			if len(got) != len(tc.matrix) {
				t.Fatalf("matrix size %d, want %d", len(got), len(tc.matrix))
			}
			for i := range tc.matrix {
				for j := range tc.matrix[i] {
					// Exact comparison on purpose: values must stay
					// bit-identical to the pre-rewrite matrices.
					if got[i][j] != tc.matrix[i][j] {
						t.Errorf("matrix[%d][%d] = %v, want %v", i, j, got[i][j], tc.matrix[i][j])
					}
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./gates/ -run TestBuiltinGateGoldenValues`
Expected: FAIL — build error: `NewHadamard()` (type `*HadamardGate`) cannot be used as `*MatrixGate`. The type mismatch IS the missing feature.

- [ ] **Step 3: Rewrite gates.go data-driven**

Replace the entire contents of `gates/gates.go` with:

```go
// Package gates contains the definition of quantum gates used in the simulator.
package gates

import (
	"fmt"
	"math"
)

// mustGate builds a built-in gate and panics if its matrix table is invalid.
// A panic here is a programmer error in this package, caught by tests.
func mustGate(name string, matrix [][]complex128) *MatrixGate {
	gate, err := NewMatrixGate(name, matrix)
	if err != nil {
		panic(fmt.Sprintf("invalid built-in gate %q: %v", name, err))
	}
	return gate
}

// NewHadamard creates a new Hadamard gate.
func NewHadamard() *MatrixGate {
	return mustGate("Hadamard", [][]complex128{
		{1.0 / complex(math.Sqrt(2), 0), 1.0 / complex(math.Sqrt(2), 0)},
		{1.0 / complex(math.Sqrt(2), 0), -1.0 / complex(math.Sqrt(2), 0)},
	})
}

// NewPauliX creates a new Pauli-X (NOT) gate.
func NewPauliX() *MatrixGate {
	return mustGate("PauliX", [][]complex128{
		{0, 1},
		{1, 0},
	})
}

// NewPauliY creates a new Pauli-Y gate.
func NewPauliY() *MatrixGate {
	return mustGate("PauliY", [][]complex128{
		{0, complex(0, -1)},
		{complex(0, 1), 0},
	})
}

// NewPauliZ creates a new Pauli-Z gate.
func NewPauliZ() *MatrixGate {
	return mustGate("PauliZ", [][]complex128{
		{1, 0},
		{0, -1},
	})
}

// NewS creates a new S (phase) gate.
func NewS() *MatrixGate {
	return mustGate("S", [][]complex128{
		{1, 0},
		{0, complex(0, 1)},
	})
}

// NewT creates a new T (π/8) gate.
func NewT() *MatrixGate {
	return mustGate("T", [][]complex128{
		{1, 0},
		{0, complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))},
	})
}

// NewCNOT creates a new Controlled-NOT gate (4x4, 2-qubit).
func NewCNOT() *MatrixGate {
	return mustGate("CNOT", [][]complex128{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 0, 1},
		{0, 0, 1, 0},
	})
}

// NewSwap creates a new SWAP gate which exchanges two qubits.
func NewSwap() *MatrixGate {
	return mustGate("SWAP", [][]complex128{
		{1, 0, 0, 0},
		{0, 0, 1, 0},
		{0, 1, 0, 0},
		{0, 0, 0, 1},
	})
}
```

- [ ] **Step 4: Run the full test suite**

Run: `go build ./... && go test ./...`
Expected: PASS for every package. The existing `state`, `sparsestate`, `circuit`, `algorithm`, `quantum`, and `cmd/quantum` suites are the behavior-preservation proof. If anything fails, fix `gates.go` — do not touch the failing suites.

- [ ] **Step 5: Commit**

```bash
git add gates/gates.go gates/gates_test.go
git commit -m "refactor(gates): make built-in gates data-driven

Replace eight concrete gate structs with canonical matrix tables
behind the existing constructor names. Golden tests lock names and
exact matrix values; matrix expressions are copied verbatim so
values stay bit-identical (the sparse CNOT fast path dispatches on
matrix content).

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Registry Builtin() and Names()

**Files:**
- Modify: `gates/registry.go`
- Modify: `gates/registry_test.go`

**Interfaces:**
- Consumes: the eight built-in constructors from Task 2.
- Produces: `Builtin() *Registry` and method `(r *Registry) Names() []string` (sorted). Task 4 relies on both plus the existing `Lookup(name string) (quantum.Gate, bool)`.

- [ ] **Step 1: Write the failing tests**

Append to `gates/registry_test.go` (keep the existing tests):

```go
func TestBuiltinRegistryContents(t *testing.T) {
	registry := Builtin()
	want := []string{"CNOT", "Hadamard", "PauliX", "PauliY", "PauliZ", "S", "SWAP", "T"}

	names := registry.Names()
	if len(names) != len(want) {
		t.Fatalf("Names() = %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("Names() = %v, want %v (sorted)", names, want)
		}
	}

	for _, name := range want {
		gate, ok := registry.Lookup(name)
		if !ok {
			t.Errorf("Lookup(%q) not found", name)
			continue
		}
		if gate.Name() != name {
			t.Errorf("Lookup(%q).Name() = %q", name, gate.Name())
		}
	}
}

func TestBuiltinRegistriesAreIndependent(t *testing.T) {
	first := Builtin()
	second := Builtin()

	extra, err := NewMatrixGate("CZ", [][]complex128{
		{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1},
	})
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}
	if err := first.Register(extra); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if _, ok := second.Lookup("CZ"); ok {
		t.Error("registering on one Builtin() registry leaked into another")
	}
}

func TestNamesOnNilRegistry(t *testing.T) {
	var registry *Registry
	if names := registry.Names(); names != nil {
		t.Errorf("nil registry Names() = %v, want nil", names)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./gates/ -run 'TestBuiltin|TestNamesOnNilRegistry'`
Expected: FAIL — build error `undefined: Builtin` (feature missing).

- [ ] **Step 3: Implement Builtin and Names**

In `gates/registry.go`, extend the import block to include `sort`, then append:

```go
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
func Builtin() *Registry {
	registry := NewRegistry()
	builtins := []*MatrixGate{
		NewHadamard(), NewPauliX(), NewPauliY(), NewPauliZ(),
		NewS(), NewT(), NewCNOT(), NewSwap(),
	}
	for _, gate := range builtins {
		if err := registry.Register(gate); err != nil {
			panic(fmt.Sprintf("builtin registry: %v", err))
		}
	}
	return registry
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go test ./gates/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add gates/registry.go gates/registry_test.go
git commit -m "feat(gates): add Builtin registry and Names enumeration

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: gates CLI demo consuming Registry and DecomposeSwap

**Files:**
- Create: `internal/examples/gates.go`
- Modify: `cmd/quantum/main.go`
- Modify: `cmd/quantum/main_test.go`
- Modify: `README.md` (demo list)

**Interfaces:**
- Consumes: `gates.Builtin()`, `Registry.Names()`, `Registry.Lookup()`, `gates.DecomposeSwap(first, second int) []gates.DecomposedGate`, `gates.NewHadamard()`, `gates.NewT()`, `gates.NewSwap()`, `state.New(numQubits int) (*state.State, error)`, `(*state.State).ApplyGate(gate quantum.Gate, targets ...int) error`, `(*state.State).Amplitude(basisState int) complex128`.
- Produces: `examples.RunAllGatesDemos()`, `examples.GateCatalogDemo()`, `examples.SwapDecompositionDemo()`; CLI demo name `gates`.

- [ ] **Step 1: Extend the CLI drift tests (failing first)**

In `cmd/quantum/main_test.go`:

1. In `TestCLIHelpOutputDemos`, after the `{"noise demo", "noise"},` entry add:

```go
		{"gates demo", "gates"},
```

2. In `TestCLIHelpOutputFlagDescription`, replace the `expectedDemos` line with:

```go
	expectedDemos := []string{"hadamard", "tgate", "bell", "algorithm", "visual", "noise", "gates", "all"}
```

3. Before `// TestMain runs setup/teardown for CLI tests.` add:

```go
// TestCLIGatesDemoRuns verifies the gates demo executes successfully.
func TestCLIGatesDemoRuns(t *testing.T) {
	// Skip in short mode since this runs the actual demo
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cmd := exec.Command("go", "run", "./cmd/quantum", "gates")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		t.Fatalf("gates demo failed to run.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	// Verify some expected output content
	output := stdout.String()
	for _, want := range []string{"Hadamard", "CNOT", "SWAP", "decomposition", "match"} {
		if !strings.Contains(output, want) {
			t.Errorf("gates demo output missing %q, got:\n%s", want, output)
		}
	}
}
```

- [ ] **Step 2: Run drift tests to verify they fail**

Run: `go test ./cmd/quantum/ -run 'TestCLIHelpOutputDemos|TestCLIHelpOutputFlagDescription|TestCLIGatesDemoRuns'`
Expected: FAIL — help output missing "gates"; gates demo run exits non-zero with `unknown demo`.

- [ ] **Step 3: Write the demo**

Create `internal/examples/gates.go`:

```go
/*
This `internal/examples/gates.go` file demonstrates the data-driven gate
catalog and gate decomposition:

 1. `GateCatalogDemo()` - Iterates the built-in gate registry and prints each
    gate's matrix, showing that every gate is just a named unitary matrix.
 2. `SwapDecompositionDemo()` - Verifies SWAP ≡ CNOT·CNOT·CNOT by applying
    both to identical two-qubit states and comparing amplitudes.
 3. `RunAllGatesDemos()` - A convenience function that runs all demonstrations.
*/

package examples

import (
	"fmt"
	"math/cmplx"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// printGateMatrix prints a gate matrix in rows of "a+bi" entries.
func printGateMatrix(matrix [][]complex128) {
	for _, row := range matrix {
		fmt.Print("  [")
		for j, v := range row {
			if j > 0 {
				fmt.Print("  ")
			}
			fmt.Printf("%6.3f%+.3fi", real(v), imag(v))
		}
		fmt.Println("]")
	}
}

// GateCatalogDemo lists every built-in gate with its matrix.
func GateCatalogDemo() {
	fmt.Println("\n=== Built-in Gate Catalog ===")
	fmt.Println("Every gate is a named unitary matrix in a registry.")

	registry := gates.Builtin()
	for _, name := range registry.Names() {
		gate, ok := registry.Lookup(name)
		if !ok {
			fmt.Printf("Error: gate %q missing from registry\n", name)
			return
		}
		fmt.Printf("\n%s:\n", name)
		printGateMatrix(gate.Matrix())
	}
}

// SwapDecompositionDemo shows that SWAP equals three alternating CNOTs.
func SwapDecompositionDemo() {
	fmt.Println("\n=== SWAP Decomposition Demonstration ===")
	fmt.Println("SWAP(a,b) = CNOT(a,b) · CNOT(b,a) · CNOT(a,b)")
	fmt.Println("Prepare (H⊗T)|00⟩ two ways and compare amplitudes.")

	direct, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	decomposed, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}

	// Identical non-trivial preparation on both states.
	for _, s := range []*state.State{direct, decomposed} {
		if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
			fmt.Printf("Error applying Hadamard gate: %v\n", err)
			return
		}
		if err := s.ApplyGate(gates.NewT(), 1); err != nil {
			fmt.Printf("Error applying T gate: %v\n", err)
			return
		}
	}

	if err := direct.ApplyGate(gates.NewSwap(), 0, 1); err != nil {
		fmt.Printf("Error applying SWAP gate: %v\n", err)
		return
	}
	for _, step := range gates.DecomposeSwap(0, 1) {
		if err := decomposed.ApplyGate(step.Gate, step.Targets...); err != nil {
			fmt.Printf("Error applying %s gate: %v\n", step.Gate.Name(), err)
			return
		}
	}

	fmt.Println("\nbasis  |  SWAP gate           |  CNOT decomposition")
	maxDiff := 0.0
	for basis := 0; basis < 4; basis++ {
		a := direct.Amplitude(basis)
		b := decomposed.Amplitude(basis)
		if diff := cmplx.Abs(a - b); diff > maxDiff {
			maxDiff = diff
		}
		fmt.Printf("|%02b⟩   | %8.4f%+.4fi  | %8.4f%+.4fi\n",
			basis, real(a), imag(a), real(b), imag(b))
	}

	if maxDiff < 1e-10 {
		fmt.Printf("\nAmplitudes match (max difference %.2e): the decomposition is exact.\n", maxDiff)
	} else {
		fmt.Printf("\nAmplitudes DO NOT match (max difference %.2e).\n", maxDiff)
	}
}

// RunAllGatesDemos runs all gate catalog demonstrations in sequence.
func RunAllGatesDemos() {
	fmt.Println("\n========================================")
	fmt.Println("       GATE CATALOG DEMONSTRATIONS")
	fmt.Println("========================================")

	GateCatalogDemo()
	SwapDecompositionDemo()

	fmt.Println("\n=== All gate demonstrations completed ===")
}
```

- [ ] **Step 4: Wire the demo into the CLI**

In `cmd/quantum/main.go`:

1. In `usage()`, after the `noise` line add:

```go
	fmt.Fprintln(out, "  gates     - Gate catalog and decomposition demonstrations")
```

2. In `runDemos`, after the `case "noise":` block add:

```go
	case "gates":
		examples.RunAllGatesDemos()
```

3. In the `case "all":` sequence, after `examples.RunAllNoiseDemos()` add:

```go
		fmt.Println("\nPress Enter to continue to gate catalog demonstrations...")
		fmt.Scanln()

		examples.RunAllGatesDemos()
```

4. Update the `-demo` flag description to:

```go
	demoFlag := flag.String("demo", "", "Demo to run (hadamard, tgate, bell, algorithm, visual, noise, gates, all)")
```

In `README.md`, after the `noise` demo line add:

```bash
go run ./cmd/quantum gates      # Run gate catalog and decomposition demonstrations
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/quantum/ && go run ./cmd/quantum gates`
Expected: tests PASS; demo prints the eight-gate catalog and "Amplitudes match".

- [ ] **Step 6: Full build and test, then commit**

Run: `go build ./... && go test ./...`
Expected: all packages pass.

```bash
git add internal/examples/gates.go cmd/quantum/main.go cmd/quantum/main_test.go README.md
git commit -m "feat(cli): add gates demo consuming Builtin registry and DecomposeSwap

Gives gates.Registry and gates.DecomposeSwap their first real
consumers: the demo iterates the built-in catalog by name and
verifies SWAP = CNOT·CNOT·CNOT on live states.

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 5: Delete dead code and close the issue

**Files:**
- Delete: `algorithm/algorithm.go`
- Delete: `measurement/` (empty directory)
- Modify: `README.md` (line about the `measurement` TODO; project-structure list)
- Modify: `specs/architecture/architectural-issues.md`

**Interfaces:**
- Consumes: nothing.
- Produces: nothing — deletions and documentation only.

- [ ] **Step 1: Verify the dead file really is unreferenced**

Run: `grep -rn "matrixGate\|newMatrixGate\|descendingTargets" --include="*.go" . | grep -v algorithm/algorithm.go`
Expected: no output. If anything appears, STOP — the plan's premise is wrong; re-assess.

- [ ] **Step 2: Delete**

```bash
git rm algorithm/algorithm.go
rmdir measurement
```

(`measurement/` is empty and untracked — git never saw it; `rmdir` is enough.)

`algorithm/algorithm.go` carried the package doc comment. Preserve it by
editing `algorithm/deutsch_jozsa.go`: replace its first line

```go
package algorithm
```

with:

```go
// Package algorithm provides small, composable quantum algorithms.
package algorithm
```

- [ ] **Step 3: Update README**

In `README.md`:

1. Replace the line

```markdown
- Measurement helpers currently live on `state.State` and `qubit.Qubit`; a dedicated `measurement` package is TODO.
```

with:

```markdown
- Measurement helpers live on `state.State` and `qubit.Qubit`.
```

2. In the project-structure list, replace the `gates/gates.go` line with:

```markdown
- [`gates/`](gates/): Data-driven gate definitions (H, X, Y, Z, S, T, CNOT, SWAP), `MatrixGate` for custom gates, the built-in registry, and SWAP decomposition.
```

3. In the examples sub-list, after the `noise.go` line add:

```markdown
  - [`gates.go`](internal/examples/gates.go): Gate catalog registry and SWAP decomposition.
```

- [ ] **Step 4: Mark the issue resolved**

In `specs/architecture/architectural-issues.md`, in the "Dead and speculative code across gates/ and algorithm/" section, after the `**Category:** RECOMMENDATION (P1)` line add:

```markdown
**Status:** Resolved (2026-08-25) — data-driven route per the issue's recommendation: `matrixGate` promoted to exported `gates.MatrixGate` (validated constructor, deep-copying `Matrix()`, `NumQubits()`); the eight concrete gate structs replaced by canonical matrix tables behind unchanged constructor names (golden tests lock exact values); `gates.Builtin()`/`Registry.Names()` added and consumed, with `DecomposeSwap`, by the new `gates` CLI demo; `algorithm/algorithm.go` (incl. `descendingTargets` and the v1 `Apply`) deleted; empty `measurement/` directory removed and README TODO dropped. Design: `docs/superpowers/specs/2026-08-25-data-driven-gates-design.md`.
```

- [ ] **Step 5: Full verification**

Run: `go build ./... && go vet ./... && gofmt -l . && go test -count=1 ./...`
Expected: build exit 0, vet clean, gofmt lists nothing, every package passes.

If `staticcheck` is installed (`which staticcheck`), also run `staticcheck ./...` and confirm the six U1000 findings for `algorithm/algorithm.go` are gone. Skip if not installed.

- [ ] **Step 6: Commit**

```bash
git add algorithm/algorithm.go algorithm/deutsch_jozsa.go README.md specs/architecture/architectural-issues.md
git commit -m "chore: delete dead algorithm scaffolding and measurement stub

algorithm/algorithm.go (matrixGate now lives in gates as MatrixGate;
descendingTargets had no consumer) and the empty measurement/
directory with its README TODO. Closes the dead-and-speculative-code
finding in specs/architecture/architectural-issues.md.

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```
