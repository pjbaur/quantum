# Data-Driven Gates Design

**Date:** 2026-08-25
**Status:** Approved
**Resolves:** "Dead and speculative code across gates/ and algorithm/" in
`specs/architecture/architectural-issues.md`

## Context

The quality assessment found ~400 LOC of unwired surface:

- `algorithm/algorithm.go`: `matrixGate`, `newMatrixGate`, three methods, and
  `descendingTargets` — all flagged staticcheck U1000 (unused, including by
  tests). `matrixGate.Apply(Qubit)` implements a retired v1 interface.
- `gates.Registry` and `gates.DecomposeSwap`: zero consumers outside their own
  tests.
- Empty `measurement/` directory persisting against a README TODO.

The issue's direction recommends promoting the `matrixGate` design by making
gates data-driven, rather than deleting it. That is the approach taken here.

Survey results that shape the design:

- No code inside or outside the repo depends on the concrete gate struct types
  (`*HadamardGate`, `*PauliXGate`, …). Every consumer calls a `gates.NewX()`
  constructor and uses the result as a `quantum.Gate`.
- `docs/compatibility-policy.md` records that there are no external API
  consumers, so changing constructor return types is an acceptable break.
- The sparse backend's CNOT fast path dispatches on matrix content
  (`isCanonicalCNOT`), not on Go types, so it is unaffected.

## Design

### 1. `gates.MatrixGate` — one gate type, promoted from `algorithm`

```go
// MatrixGate is a quantum gate defined entirely by its unitary matrix.
type MatrixGate struct {
    name   string
    matrix [][]complex128
    qubits int
}

// NewMatrixGate validates the matrix and returns a gate.
// The matrix must be non-empty, square, and its dimension a power of two >= 2.
func NewMatrixGate(name string, matrix [][]complex128) (*MatrixGate, error)

func (g *MatrixGate) Name() string
func (g *MatrixGate) Matrix() [][]complex128 // returns a deep copy
func (g *MatrixGate) NumQubits() int
```

Decisions:

- **`Matrix()` returns a deep copy.** Today every gate builds a fresh literal
  per call, so callers can mutate the returned slice without corrupting shared
  state. Data-driven gates share one canonical table per gate, so the copy is
  required to preserve that property.
- **Validation matches the old `newMatrixGate`** (non-empty, square,
  power-of-two dimension) plus a minimum dimension of 2. Unitarity is not
  checked; the state backends already enforce normalization on application.
- **No `Apply(Qubit)` method.** That was the retired v1 interface; the v2
  contract is `QuantumState.ApplyGate(gate, targets...)`.
- `NewMatrixGate` also rejects an empty name, since `Registry.Register`
  requires a non-empty `Name()`.

### 2. Built-in gates become data

The eight concrete types and their 16 methods are replaced by canonical matrix
tables and thin constructors:

```go
func NewHadamard() *MatrixGate // Name() == "Hadamard"
func NewPauliX() *MatrixGate   // "PauliX"
func NewPauliY() *MatrixGate   // "PauliY"
func NewPauliZ() *MatrixGate   // "PauliZ"
func NewS() *MatrixGate        // "S"
func NewT() *MatrixGate        // "T"
func NewCNOT() *MatrixGate     // "CNOT"
func NewSwap() *MatrixGate     // "SWAP"
```

- Constructor names and gate name strings are unchanged; only the return type
  changes (`*HadamardGate` → `*MatrixGate`, etc.). All in-repo callers already
  treat the results as `quantum.Gate`.
- A `mustGate(name, matrix)` helper panics if a built-in table is invalid —
  a programmer error, covered by tests that construct every built-in.
- The concrete types (`HadamardGate`, …) are deleted, not aliased. The
  compatibility policy permits the break and aliases would preserve the same
  dead surface the issue complains about.

### 3. Registry and DecomposeSwap gain real consumers

- `gates.Builtin() *Registry` returns a registry preloaded with the eight
  built-ins. Each call returns a fresh registry so callers can extend their
  copy without affecting others.
- `Registry.Names() []string` (sorted) is added for enumeration.
- A new `gates` CLI demo (`internal/examples/gates.go`, registered in
  `cmd/quantum`) consumes both:
  - iterates `Builtin().Names()`, looking up each gate and printing its
    matrix — a genuine name→gate lookup consumer;
  - demonstrates SWAP ≡ CNOT·CNOT·CNOT by applying `NewSwap()` and the
    `DecomposeSwap` sequence to identical two-qubit states and comparing
    amplitudes.

### 4. Deletions

- `algorithm/algorithm.go` — entire file (`matrixGate` moves to `gates` in
  spirit; `descendingTargets` is dead with no successor).
- `measurement/` — empty directory removed; README line about the TODO
  reworded to drop the dead pointer.

### 5. Testing

TDD throughout:

- `NewMatrixGate` validation: empty name, empty matrix, non-square,
  non-power-of-two, 1×1; success for 2×2 and 4×4.
- `Matrix()` deep-copy: mutating a returned matrix must not affect a
  subsequent `Matrix()` call or another instance from the same constructor.
- `NumQubits()`: 1 for 2×2, 2 for 4×4.
- Every built-in constructor: name and matrix unchanged from today (golden
  values), constructed without panic.
- `Builtin()`: contains exactly the eight names; registries are independent;
  `Names()` sorted.
- CLI drift tests: `gates` demo added to both expected-demo lists plus a
  smoke test, mirroring the existing pattern.
- Existing suites (`gates`, `quantum`, `state`, `sparsestate`, `circuit`,
  `algorithm`, `cmd/quantum`) must stay green — they are the proof that the
  data-driven swap is behavior-preserving.

## Error handling

- `NewMatrixGate` returns descriptive errors (reused semantics from the old
  `newMatrixGate`).
- Built-in constructors cannot fail; invalid tables panic at first use and are
  caught by tests.
- The demo follows the existing example pattern: print the error and return.

## Out of scope

- Unitarity validation of user matrices.
- Migrating `algorithm/` or `circuit/` to construct gates via `Registry`.
- The other open issues (CI gates, LICENSE, injectable randomness).
