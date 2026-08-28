# ADR-0003: Sparse Backend Capability Model

## Status

Accepted. The gate-width limit (1–2 qubit gates only) was later lifted by
[ADR-0008](0008-sparse-generic-k-qubit-gates.md); the capability-check API
(`BackendCapabilities`, `SupportsGateQubits`, `MaxGateQubits`) decided here
is retained unchanged.

## Context

The sparse state backend (`internal/sparsestate/state.go`) currently has limited gate support compared to the dense backend.

### Current Capabilities

| Gate Type | Dense Backend | Sparse Backend |
|-----------|---------------|----------------|
| Single-qubit (2x2 matrix) | Supported | Supported |
| Two-qubit (4x4 matrix) | Supported | Supported (CNOT optimized, others generic) |
| Three+ qubit (8x8+ matrix) | Supported | **Not Supported** |

### Current Implementation (`internal/sparsestate/state.go:101-114`)

```go
if requiredQubits == 1 {
    return s.applySingleQubitGate(gate, targets[0])
}

if requiredQubits == 2 {
    return s.applyTwoQubitGate(gate, targets)
}

return &quantum.InvalidGateApplicationError{
    Gate:        gate.Name(),
    RequiredLen: requiredQubits,
    ActualLen:   len(targets),
}
```

For 3+ qubit gates, the sparse backend returns an error claiming "invalid gate application" which is misleading - the gate is valid, the backend just doesn't support it.

### Two Possible Approaches

| Approach | Pros | Cons |
|----------|------|------|
| **Full Generic Support**: Implement k-qubit sparse gate application | Feature parity with dense | Complex implementation, may have poor performance |
| **Explicit Capability Limits**: Document limits, fail clearly | Simple, honest about capabilities | Users must choose dense for 3+ qubit gates |

## Decision

We adopt **Explicit Capability Limits with Capability-Check API**.

### Capability Declaration

The sparse backend explicitly supports:
- Single-qubit gates (1-qubit operations)
- Two-qubit gates (2-qubit operations)

The sparse backend explicitly does **not** support:
- Three-qubit gates (e.g., Toffoli, Fredkin)
- Four+ qubit gates

### New Error Type

```go
// UnsupportedOperationError indicates that an operation is valid but not
// supported by this backend. Callers should use a different backend.
type UnsupportedOperationError struct {
    Operation   string  // e.g., "3-qubit gate application"
    Backend     string  // e.g., "sparse"
    Alternative string  // e.g., "dense state backend"
}

func (e *UnsupportedOperationError) Error() string {
    return fmt.Sprintf("%s does not support %s: use %s instead",
        e.Backend, e.Operation, e.Alternative)
}
```

### Capability-Check API

Add a capability interface that backends can implement:

```go
// BackendCapabilities describes what operations a quantum state backend supports.
type BackendCapabilities interface {
    // SupportsGateQubits returns whether this backend can apply gates
    // operating on the specified number of qubits.
    SupportsGateQubits(qubitCount int) bool

    // MaxGateQubits returns the maximum number of qubits a gate can operate on.
    // Returns 0 if there is no limit.
    MaxGateQubits() int
}
```

The sparse backend implements this:

```go
func (s *State) SupportsGateQubits(qubitCount int) bool {
    return qubitCount >= 1 && qubitCount <= 2
}

func (s *State) MaxGateQubits() int {
    return 2
}
```

### Error Handling in ApplyGate

Update `sparsestate.State.ApplyGate` to return `UnsupportedOperationError` instead of `InvalidGateApplicationError`:

```go
if requiredQubits > 2 {
    return &quantum.UnsupportedOperationError{
        Operation:   fmt.Sprintf("%d-qubit gate application", requiredQubits),
        Backend:     "sparse",
        Alternative: "dense state backend (state.State)",
    }
}
```

### Circuit-Backend Compatibility

When executing a circuit:
1. Circuit compilation can check backend capabilities before execution.
2. If the circuit contains unsupported gates, fail early with clear message.
3. Suggest switching to dense backend when appropriate.

## Consequences

### Positive

1. **Clarity**: Users immediately understand why their operation failed.
2. **Early Detection**: Capability checks happen before partial execution.
3. **Actionable**: Error message tells users what to do (use dense backend).
4. **Honest**: We don't claim feature parity we don't have.

### Negative

1. **Feature Gap**: Sparse backend cannot run all circuits.
2. **Migration**: Users with 3+ qubit gates must use dense backend.

### Neutral

The sparse backend remains the recommended choice for:
- Large qubit counts with limited entanglement
- Memory-constrained environments
- Circuits using only 1-2 qubit gates

The dense backend is recommended for:
- Small qubit counts (< 25 qubits)
- Circuits with 3+ qubit gates (Toffoli, Fredkin, etc.)
- Maximum feature compatibility

## Implementation Plan

1. Add `UnsupportedOperationError` to `quantum/errortypes.go`.
2. Add `BackendCapabilities` interface to `quantum/interfaces.go`.
3. Implement `BackendCapabilities` on `sparsestate.State`.
4. Implement `BackendCapabilities` on `state.State` (returning no limit).
5. Update `sparsestate.State.ApplyGate` to return `UnsupportedOperationError`.
6. Add capability checking to circuit execution planning phase.

## Decision Log

- 2026-02-21: Initial decision accepted
- Referenced by: `docs/critical-review-implementation-plan-2026-02-21.md` Phase 0.3
