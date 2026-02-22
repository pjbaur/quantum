# v2.0 API Changes and Deprecation Policy

This document lists all API removals planned for v2.0, with migration guidance.

## Overview

Version 2.0 removes the legacy single-qubit gate application API in favor of the state-vector-first approach. See `docs/adr/0001-state-vector-first-api.md` for the rationale.

## Breaking Changes

### 1. Gate Interface Simplification

**Before (v1.x):**

```go
type Gate interface {
    Apply(q Qubit) error
    Name() string
    Matrix() [][]complex128
}
```

**After (v2.0):**

```go
type Gate interface {
    Name() string
    Matrix() [][]complex128
}
```

### 2. Removed Gate Methods

| Method | Location | Replacement |
|--------|----------|-------------|
| `HadamardGate.Apply(q)` | `gates/gates.go:31-40` | `state.ApplyGate(h, target)` |
| `PauliXGate.Apply(q)` | `gates/gates.go:63-70` | `state.ApplyGate(x, target)` |
| `PauliYGate.Apply(q)` | `gates/gates.go:93-100` | `state.ApplyGate(y, target)` |
| `PauliZGate.Apply(q)` | `gates/gates.go:123-130` | `state.ApplyGate(z, target)` |
| `SGate.Apply(q)` | `gates/gates.go:153-160` | `state.ApplyGate(s, target)` |
| `TGate.Apply(q)` | `gates/gates.go:183-191` | `state.ApplyGate(t, target)` |
| `CNOTGate.Apply(q)` | `gates/gates.go:217-225` | Always errored; use `state.ApplyGate(cnot, control, target)` |
| `CNOTGate.ApplyControlled()` | `gates/gates.go:227-250` | `state.ApplyGate(cnot, control, target)` |
| `SwapGate.Apply(q)` | `gates/gates.go:275-283` | Always errored; use `state.ApplyGate(swap, q1, q2)` |
| `SwapGate.ApplySwap()` | `gates/gates.go:285-302` | `state.ApplyGate(swap, q1, q2)` |

### 3. Qubit Interface

The `quantum.Qubit` interface remains but is discouraged for new code. It may be fully removed in a future version.

Consider full removal in v3.0 if state-vector-first proves sufficient for all use cases.

## Migration Examples

### Single-Qubit Gates

**Before:**

```go
qubit := qubit.New()
h := gates.NewHadamard()
err := h.Apply(qubit)  // Deprecated
```

**After:**

```go
state := state.New(1)  // 1-qubit state
h := gates.NewHadamard()
err := state.ApplyGate(h, 0)  // Apply to qubit index 0
```

### Two-Qubit Gates (CNOT)

**Before:**

```go
control := qubit.New()
target := qubit.New()
cnot := gates.NewCNOT()
err := cnot.ApplyControlled(control, target)  // Deprecated, doesn't handle entanglement
```

**After:**

```go
state := state.New(2)  // 2-qubit state
cnot := gates.NewCNOT()
err := state.ApplyGate(cnot, 0, 1)  // control=0, target=1
```

### Bell State Creation

**Before:**

```go
// Using individual qubits - cannot represent entanglement correctly
q0 := qubit.New()
q1 := qubit.New()
h := gates.NewHadamard()
h.Apply(q0)
cnot := gates.NewCNOT()
cnot.ApplyControlled(q0, q1)  // Wrong: doesn't create true Bell state
```

**After:**

```go
// Using state vector - correctly creates Bell state
state := state.New(2)
h := gates.NewHadamard()
state.ApplyGate(h, 0)        // |+⟩|0⟩ = (|0⟩+|1⟩)|0⟩/√2
cnot := gates.NewCNOT()
state.ApplyGate(cnot, 0, 1)  // (|00⟩+|11⟩)/√2 - true Bell state
```

## Files Requiring Migration

| File | Lines | Changes Required |
|------|-------|------------------|
| `quantum/quantum_test.go` | 245, 260, 275-276, 294, 305, 316, 339, 350, 363, 567 | Update tests to use state.ApplyGate |
| `internal/examples/tgate.go` | 21, 33, 45, 64, 77, 172, 178, 189, 195 | Rewrite using state vector |
| `internal/examples/hadamard.go` | 24 | Rewrite using state vector |
| `internal/examples/bell.go` | 146, 196, 207 | Rewrite using state vector |
| `internal/examples/visualization.go` | 38, 42 | Rewrite using state vector |

## Timeline

- **v1.x**: Legacy API deprecated, warnings added
- **v2.0**: Legacy API removed, state-vector-first only

## See Also

- `docs/adr/0001-state-vector-first-api.md` - Architecture decision rationale
- `docs/compatibility-policy.md` - Compatibility stance for this project
