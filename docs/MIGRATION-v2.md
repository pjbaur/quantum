# Migration Guide: v1.x to v2.0

This guide documents all breaking changes introduced in v2.0 and provides migration paths for each.

## Overview

Version 2.0 introduces a **state-vector-first API** that corrects architectural issues with the legacy single-qubit gate application model. The changes ensure:

- Correct handling of entanglement in multi-qubit systems
- Explicit backend capability contracts
- Consistent error handling across all packages
- Enforced safety in parallel execution

## Summary of Breaking Changes

| Category | Change | Impact |
|----------|--------|--------|
| Gate Interface | `Apply(q Qubit)` removed from interface | High |
| Gate Methods | `ApplyControlled`, `ApplySwap` removed | High |
| Constructors | Invalid qubit counts now return error | Medium |
| Errors | New error types with additional fields | Low |
| Concurrency | Shared-state detection in parallel execution | Medium |
| Backend | Sparse backend has explicit capability limits | Medium |

## Gate Application Changes

### The Core Change

**Before (v1.x):**
```go
// DEPRECATED: This pattern cannot handle entanglement correctly
type Gate interface {
    Apply(q Qubit) error
    Name() string
    Matrix() [][]complex128
}
```

**After (v2.0):**
```go
// Gate provides metadata only; application is via QuantumState
type Gate interface {
    Name() string
    Matrix() [][]complex128
}
```

### Single-Qubit Gates

**Before:**
```go
qubit := qubit.New()
h := gates.NewHadamard()
err := h.Apply(qubit)  // REMOVED
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
// INCORRECT: Does not handle entanglement
control := qubit.New()
target := qubit.New()
cnot := gates.NewCNOT()
err := cnot.ApplyControlled(control, target)  // REMOVED
```

**After:**
```go
// CORRECT: Properly handles entanglement
state := state.New(2)  // 2-qubit state
cnot := gates.NewCNOT()
err := state.ApplyGate(cnot, 0, 1)  // control=0, target=1
```

### Bell State Creation

**Before (Incorrect):**
```go
q0 := qubit.New()
q1 := qubit.New()
h := gates.NewHadamard()
h.Apply(q0)  // Does not create entanglement
cnot := gates.NewCNOT()
cnot.ApplyControlled(q0, q1)  // WRONG: Not a true Bell state
```

**After (Correct):**
```go
state := state.New(2)
h := gates.NewHadamard()
state.ApplyGate(h, 0)        // |+⟩|0⟩ = (|0⟩+|1⟩)|0⟩/√2
cnot := gates.NewCNOT()
state.ApplyGate(cnot, 0, 1)  // (|00⟩+|11⟩)/√2 - true Bell state
```

## Constructor Behavior Changes

### Invalid Qubit Counts

All constructors now consistently return errors for invalid qubit counts instead of silently coercing to 1.

**Before (v1.x):**
```go
// Dense state: silently coerced to 1 qubit
s := state.New(0)    // Creates 1-qubit state
s := state.New(-5)   // Creates 1-qubit state

// Sparse state: silently coerced to 1 qubit
s := sparsestate.New(0)  // Creates 1-qubit state
```

**After (v2.0):**
```go
// All constructors return InvalidQubitCountError
s, err := state.New(0)    // Returns error
s, err := state.New(-5)   // Returns error
s, err := state.New(1)    // OK: Creates 1-qubit state
```

### Handling Constructor Errors

```go
s, err := state.New(numQubits)
if err != nil {
    if invalidQubits, ok := err.(*quantum.InvalidQubitCountError); ok {
        fmt.Printf("Invalid qubit count %d: %s\n",
            invalidQubits.Count, invalidQubits.Message)
    }
    return err
}
```

## Error Type Changes

### NormalizationError

The `NormalizationError` now reports both the attempted and current sums.

**Before:**
```go
type NormalizationError struct {
    Sum float64  // Post-rollback sum (misleading)
}
```

**After:**
```go
type NormalizationError struct {
    AttemptedSum float64  // The invalid sum that was attempted
    CurrentSum   float64  // The valid sum after rollback
}

func (e *NormalizationError) Error() string {
    return fmt.Sprintf("normalization error: attempted sum %.6f, current sum %.6f",
        e.AttemptedSum, e.CurrentSum)
}
```

**Migration:**
```go
// Before
if normErr, ok := err.(*quantum.NormalizationError); ok {
    fmt.Printf("Sum was %f\n", normErr.Sum)  // Always 0 now
}

// After
if normErr, ok := err.(*quantum.NormalizationError); ok {
    fmt.Printf("Attempted: %f, Current: %f\n",
        normErr.AttemptedSum, normErr.CurrentSum)
}
```

### New Error Types

| Error Type | Purpose | Fields |
|------------|---------|--------|
| `InvalidQubitCountError` | Invalid constructor input | `Count`, `Message` |
| `SharedStateError` | Parallel execution safety | `DuplicateIndices` |
| `UnsupportedOperationError` | Backend capability limits | `Operation`, `Backend`, `Alternative` |
| `InvalidGateMatrixError` | Gate matrix validation | `GateName`, `Rows`, `Cols` |

## Concurrency Contract Changes

### ExecuteAllParallel Safety

The parallel execution function now enforces state independence.

**Before (v1.x):**
```go
// Silent corruption if states are shared
ExecuteAllParallel(circuit, state1, state1, state2)  // Bug!
```

**After (v2.0):**
```go
// Explicit error on shared state
err := ExecuteAllParallel(circuit, state1, state1, state2)
// Returns: SharedStateError{DuplicateIndices: [0, 1]}
```

### Handling SharedStateError

```go
err := circuit.ExecuteAllParallel(c, states...)
if err != nil {
    if sharedErr, ok := err.(*quantum.SharedStateError); ok {
        fmt.Printf("Duplicate state indices: %v\n", sharedErr.DuplicateIndices)
        // Fix: ensure each state is a unique instance
    }
    return err
}
```

## Backend Capability Model

### Sparse Backend Limits

The sparse backend now explicitly declares its capability limits.

| Gate Type | Dense Backend | Sparse Backend |
|-----------|---------------|----------------|
| Single-qubit (2x2) | Supported | Supported |
| Two-qubit (4x4) | Supported | Supported |
| Three+ qubit (8x8+) | Supported | **Not Supported** |

### Checking Capabilities

```go
// Check before executing 3+ qubit gates
if caps, ok := backend.(quantum.BackendCapabilities); ok {
    if !caps.SupportsGateQubits(3) {
        fmt.Printf("Backend only supports up to %d-qubit gates\n",
            caps.MaxGateQubits())
        // Use dense backend instead
    }
}
```

### Handling UnsupportedOperationError

```go
err := sparseState.ApplyGate(toffoli, 0, 1, 2)
if err != nil {
    if unsupported, ok := err.(*quantum.UnsupportedOperationError); ok {
        fmt.Printf("%s does not support %s\n",
            unsupported.Backend, unsupported.Operation)
        fmt.Printf("Use %s instead\n", unsupported.Alternative)
    }
    return err
}
```

## Files Requiring Migration

| File | Changes Required |
|------|------------------|
| `quantum/quantum_test.go` | Update tests to use `state.ApplyGate` |
| `internal/examples/tgate.go` | Rewrite using state vector |
| `internal/examples/hadamard.go` | Rewrite using state vector |
| `internal/examples/bell.go` | Rewrite using state vector |
| `internal/examples/visualization.go` | Rewrite using state vector |

## References

- `docs/adr/0001-state-vector-first-api.md` - API direction rationale
- `docs/adr/0002-concurrency-contract.md` - Parallel execution contract
- `docs/adr/0003-sparse-backend-capability.md` - Backend capability model
- `docs/adr/0004-diagnostics-constructor-consistency.md` - Error and constructor changes
- `docs/deprecation-policy-v2.md` - Detailed removal list
- `docs/compatibility-policy.md` - Project compatibility stance
