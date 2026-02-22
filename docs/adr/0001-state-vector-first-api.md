# ADR-0001: State-Vector-First API Direction

## Status

Accepted

## Context

The quantum computing simulator currently has a dual architecture for gate application:

### The Problem

1. **Gate Interface Requires `Apply(q Qubit)`** (`quantum/interfaces.go:32-42`)

   The `Gate` interface mandates an `Apply(q Qubit) error` method. This works correctly only for single-qubit gates.

2. **Multi-Qubit Gates Cannot Implement `Apply(Qubit)` Correctly**

   - `CNOTGate.Apply(q)` always returns an error (`gates/gates.go:217-225`)
   - `SwapGate.Apply(q)` always returns an error (`gates/gates.go:275-283`)

3. **Non-Physical Helper Methods Exist**

   - `CNOTGate.ApplyControlled(control, target Qubit)` (`gates/gates.go:227-250`)
   - `SwapGate.ApplySwap(q1, q2 Qubit)` (`gates/gates.go:285-302`)

   These methods cannot handle entanglement correctly. The `ApplyControlled` implementation explicitly admits: "This simplified implementation won't handle entanglement correctly."

4. **The Correct Model Already Exists**

   `QuantumState.ApplyGate(gate Gate, targets ...int) error` (`quantum/interfaces.go:57-60`) is the physically accurate approach:
   - Operates on the full state vector
   - Correctly handles entanglement
   - Works uniformly for single and multi-qubit gates

### Current API Usage (Legacy)

The following code uses the deprecated `gate.Apply(q)` pattern:

| File | Lines | Usage |
|------|-------|-------|
| `quantum/quantum_test.go` | 245, 260, 275-276, 294, 305, 316, 339, 350, 363, 567 | `gate.Apply(q)` |
| `gates/gates.go` | 237 | `xGate.Apply(target)` in `ApplyControlled` |
| `internal/examples/tgate.go` | 21, 33, 45, 64, 77, 172, 178, 189, 195 | `gate.Apply(q)` |
| `internal/examples/hadamard.go` | 24 | `h.Apply(q)` |
| `internal/examples/bell.go` | 146, 196, 207 | `gate.Apply(qubit)` |
| `internal/examples/visualization.go` | 38, 42 | `gate.Apply(q)` |

## Decision

We adopt **state-vector-first** as the primary and recommended API direction.

### Primary Execution Model

`QuantumState.ApplyGate(gate Gate, targets ...int) error` is the **only** supported way to apply gates.

```go
// Correct: Apply gate to state vector with target indices
state.ApplyGate(hadamard, 0)           // Single-qubit gate
state.ApplyGate(cnot, 0, 1)            // Two-qubit gate (control, target)
state.ApplyGate(swap, 0, 1)            // Two-qubit gate
```

### Gate Interface Purpose

The `Gate` interface exists to provide gate metadata and matrix representation:

```go
// v2.0 Gate interface (proposed)
type Gate interface {
    Name() string
    Matrix() [][]complex128
}
```

The `Apply(q Qubit) error` method is **removed** from the interface in v2.0.

### Removed Methods

| Method | Reason |
|--------|--------|
| `Gate.Apply(q Qubit) error` | Cannot work correctly for multi-qubit gates |
| `CNOTGate.ApplyControlled(control, target Qubit)` | Non-physical, cannot handle entanglement |
| `SwapGate.ApplySwap(q1, q2 Qubit)` | Non-physical, ignores entanglement |

### Migration Path

Replace all legacy patterns:

```go
// Before (v1.x) - deprecated
gate.Apply(qubit)

// After (v2.0) - correct
state.ApplyGate(gate, qubitIndex)
```

For code using individual `qubit.Qubit` objects, rewrite to use `state.State` directly.

## Consequences

### Positive

1. **Correctness**: All gate applications correctly handle entanglement
2. **Simplicity**: Single, uniform API for all gate types
3. **Clarity**: No confusing "helper" methods that don't actually work
4. **Maintainability**: Reduced API surface, less code to maintain

### Negative

1. **Breaking Change**: All existing `gate.Apply(q)` calls must be migrated
2. **Examples Rewrite**: Examples using individual qubits need restructuring
3. **Test Updates**: Tests using the legacy pattern must be updated

### Neutral

1. The `Qubit` interface remains available for cases where single-qubit state inspection is useful
2. Individual gate implementations may retain `Apply` methods for backward compatibility during transition, but they are not part of the interface

## Rationale

### Why Individual Qubit Application Cannot Work

In quantum mechanics, multi-qubit systems exist in a shared Hilbert space. When qubits become entangled, you cannot describe them independently:

- A Bell state `(|00⟩ + |11⟩)/√2` has no individual qubit description
- Applying a gate to one qubit affects the entire state vector
- The `Apply(q Qubit)` model fundamentally cannot represent this

### Why State-Vector-First is Correct

The state vector `|ψ⟩ = Σᵢ αᵢ|bᵢ⟩` is the complete description of a quantum system:

- Gates are unitary operators acting on this vector
- Matrix multiplication `U|ψ⟩` is the correct transformation
- Entanglement is naturally preserved

## Decision Log

- 2026-02-21: Initial decision accepted
- Referenced by: `docs/critical-review-implementation-plan-2026-02-21.md` Phase 0.1
- Deprecation details: `docs/deprecation-policy-v2.md`
- Compatibility stance: `docs/compatibility-policy.md`
