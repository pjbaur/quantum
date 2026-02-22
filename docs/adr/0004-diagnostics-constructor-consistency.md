# ADR-0004: Diagnostics and Constructor Consistency

## Status

Accepted

## Context

The codebase has inconsistent behavior in two critical areas: normalization error reporting and constructor behavior for invalid inputs.

### Problem 1: Normalization Diagnostics Report Wrong Value

When `SetAmplitude` would result in a non-normalized state, the state is rolled back but the error reports the **post-rollback** sum, not the **attempted** sum.

**Dense State** (`state/state.go:66-74`):
```go
oldValue := s.amplitudes[basisState]
s.amplitudes[basisState] = value

if !s.isNormalized() {
    s.amplitudes[basisState] = oldValue  // Rollback
    return &quantum.NormalizationError{Sum: s.probabilitySum()}  // Reports post-rollback!
}
```

**Sparse State** (`internal/sparsestate/state.go:57-67`):
```go
oldValue, had := s.amplitudes[basisState]
s.setAmplitudeUnsafe(basisState, value)

if !s.isNormalized() {
    // Rollback...
    return &quantum.NormalizationError{Sum: s.probabilitySum()}  // Reports post-rollback!
}
```

The reported `Sum` is always ~1.0 (the valid state), which provides no debugging information about what the caller actually attempted.

### Problem 2: Constructor Behavior Inconsistency

| Constructor | Input `numQubits <= 0` | Behavior |
|-------------|------------------------|----------|
| `circuit.New(numQubits)` | Returns error | `"numQubits must be positive"` |
| `state.New(numQubits)` | Coerces to 1 | Silently changes 0 or negative to 1 |
| `sparsestate.New(numQubits)` | Coerces to 1 | Silently changes 0 or negative to 1 |

The circuit constructor is strict, but state constructors silently "fix" invalid input. This is inconsistent and can mask bugs.

### Current Error Type

```go
type NormalizationError struct {
    Sum float64
}

func (e *NormalizationError) Error() string {
    return fmt.Sprintf("quantum state is not normalized (sum of probabilities = %f, should be 1.0)", e.Sum)
}
```

The field is named `Sum` but doesn't indicate whether it's the attempted or current sum.

## Decision

### Decision 1: Report Attempted Sum in Normalization Errors

Extend `NormalizationError` to capture both values:

```go
type NormalizationError struct {
    AttemptedSum float64  // The sum that would have resulted from the change
    CurrentSum   float64  // The sum after rollback (should be ~1.0)
}

func (e *NormalizationError) Error() string {
    return fmt.Sprintf("normalization violation: attempted change would result in "+
        "probability sum %f (must be 1.0); state rolled back to valid sum %f",
        e.AttemptedSum, e.CurrentSum)
}
```

**Implementation Change**:

```go
// Capture attempted sum BEFORE rollback
attemptedSum := s.probabilitySum()

// Rollback
s.amplitudes[basisState] = oldValue

// Return error with both values
return &quantum.NormalizationError{
    AttemptedSum: attemptedSum,
    CurrentSum:   s.probabilitySum(),
}
```

### Decision 2: Standardize Constructor Behavior

All constructors should **return an error** for invalid qubit counts. Silent coercion is removed.

**New Behavior**:

| Constructor | Input `numQubits <= 0` | Behavior |
|-------------|------------------------|----------|
| `circuit.New(numQubits)` | Returns error | No change |
| `state.New(numQubits)` | **Returns error** | Previously coerced |
| `sparsestate.New(numQubits)` | **Returns error** | Previously coerced |

**Standardized Error Type**:

```go
// InvalidQubitCountError indicates that an invalid number of qubits was specified.
type InvalidQubitCountError struct {
    Requested int
    Reason    string
}

func (e *InvalidQubitCountError) Error() string {
    return fmt.Sprintf("invalid qubit count %d: %s", e.Requested, e.Reason)
}
```

**Constructor Signatures**:

```go
// circuit/circuit.go
func New(numQubits int) (*Circuit, error)  // No change

// state/state.go
func New(numQubits int) (*State, error)  // Was: *State (no error)

// internal/sparsestate/state.go
func New(numQubits int) (*State, error)  // Was: *State (no error)
```

### Decision 3: No Backward Compatibility Gate

The coercion-to-1 behavior is **removed entirely**. This is a breaking change justified by:

1. **No external consumers**: Per `docs/compatibility-policy.md`, we have no external API consumers.
2. **Bug masking**: Silent coercion hides programmer errors.
3. **Consistency**: All constructors should behave the same way.

## Consequences

### Positive

1. **Better Debugging**: `NormalizationError` now shows what was actually attempted.
2. **Consistency**: All constructors have the same validation behavior.
3. **Early Bug Detection**: Invalid qubit counts fail fast with clear errors.
4. **Self-Documenting**: Error messages explain the constraint.

### Negative

1. **Breaking Changes**:
   - `state.New()` and `sparsestate.New()` now return `(T, error)` instead of `T`.
   - Code that passed `0` or negative values will now fail.
2. **Migration**: Call sites must handle the new error return.

### Neutral

The `NormalizationError.Sum` field is deprecated. Existing code that reads it will compile but get the zero value. This is acceptable because:
- There are no external consumers.
- The field was never useful (always reported ~1.0).

## Migration Guide

### For NormalizationError

```go
// Before - Sum was always ~1.0 (useless)
if normErr, ok := err.(*quantum.NormalizationError); ok {
    fmt.Println("bad sum:", normErr.Sum)
}

// After - AttemptedSum shows what was tried
if normErr, ok := err.(*quantum.NormalizationError); ok {
    fmt.Printf("attempted: %f, current: %f\n",
        normErr.AttemptedSum, normErr.CurrentSum)
}
```

### For State Constructors

```go
// Before - No error handling needed
s := state.New(3)

// After - Must handle error
s, err := state.New(3)
if err != nil {
    // Handle invalid qubit count
}
```

## Implementation Plan

1. Update `NormalizationError` struct with `AttemptedSum` and `CurrentSum` fields.
2. Update error string formatting in `NormalizationError.Error()`.
3. Update `state/state.go:SetAmplitude` to capture `AttemptedSum` before rollback.
4. Update `internal/sparsestate/state.go:SetAmplitude` similarly.
5. Add `InvalidQubitCountError` type to `quantum/errortypes.go`.
6. Update `state/state.go:New` to return `(*State, error)`.
7. Update `internal/sparsestate/state.go:New` to return `(*State, error)`.
8. Update all call sites to handle the new error return.
9. Update tests to expect errors for invalid qubit counts.

## Decision Log

- 2026-02-21: Initial decision accepted
- Referenced by: `docs/critical-review-implementation-plan-2026-02-21.md` Phase 0.4
