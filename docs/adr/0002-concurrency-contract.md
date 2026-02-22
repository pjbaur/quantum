# ADR-0002: Concurrency Contract for Parallel Execution

## Status

Accepted

## Context

The `circuit.ExecuteAllParallel` function enables concurrent circuit execution but currently has an undefined safety contract.

### Current Implementation (`circuit/parallel.go`)

```go
// Execution pairs a circuit with the state it should operate on.
// Circuits and states must be independent to safely execute in parallel.
type Execution struct {
    Circuit *Circuit
    State   quantum.QuantumState
}

func ExecuteAllParallel(executions []Execution, opts ParallelOptions) error {
    // No validation that states are independent
    // Workers share access to Execution structs
}
```

### Problems

1. **No Runtime Enforcement**: The comment states "must be independent" but there is no validation.
2. **Silent Data Corruption**: If callers pass the same `State` pointer in multiple executions, concurrent writes cause race conditions with undefined behavior.
3. **No Diagnostic Help**: Misuse is difficult to debug because failures are non-deterministic.

### Two Possible Approaches

| Approach | Pros | Cons |
|----------|------|------|
| **Fail-Fast**: Detect shared pointers and error immediately | Clear errors, predictable behavior | Small runtime overhead for pointer tracking |
| **Auto-Clone**: Clone states internally before parallel execution | Transparent to callers | Hidden memory cost, may mask caller bugs |

## Decision

We adopt **Fail-Fast with Shared-State Detection**.

### Contract Definition

1. **Each `Execution.State` must be a unique pointer** - no two executions may share the same underlying state.
2. **Detection**: Before parallel execution begins, validate that all state pointers are unique.
3. **Error Behavior**: Return a dedicated error type with clear diagnostic information.

### Error Type

```go
// SharedStateError indicates that the same quantum state was passed
// to multiple parallel executions, which would cause data races.
type SharedStateError struct {
    DuplicateIndices []int  // Indices of executions sharing state
}

func (e *SharedStateError) Error() string {
    return fmt.Sprintf("parallel execution requires independent states: "+
        "executions %v share the same state pointer", e.DuplicateIndices)
}
```

### Implementation Requirements

1. **Pre-execution validation**: Check for duplicate `State` pointers before spawning workers.
2. **Fast path for valid input**: Minimal overhead when all states are independent.
3. **Clear error messages**: Include the indices of conflicting executions.

### Performance Tradeoff

- **Overhead**: O(n) map of pointer identities before parallel execution starts.
- **Impact**: Negligible compared to quantum state operations.
- **Benefit**: Catches programmer errors early with actionable messages.

## Consequences

### Positive

1. **Safety**: Data races are detected before they can corrupt simulation results.
2. **Debugging**: Clear error messages identify exactly which executions conflict.
3. **Documentation**: The error type serves as inline documentation of the contract.
4. **Performance**: No hidden cloning overhead.

### Negative

1. **Breaking Change**: Code that accidentally worked due to lucky timing will now fail.
2. **Migration**: Callers with shared-state bugs must fix their code.

### Neutral

The `Circuit` pointer is not checked because circuits are read-only during execution. Multiple executions can safely share the same circuit.

## Implementation Plan

1. Add `SharedStateError` type to `quantum/errortypes.go`.
2. Add validation function in `circuit/parallel.go`:
   ```go
   func validateIndependentStates(executions []Execution) error {
       seen := make(map[quantum.QuantumState]int)
       for i, exec := range executions {
           if exec.State == nil {
               continue // nil states are caught by executeOne
           }
           if firstIdx, exists := seen[exec.State]; exists {
               return &SharedStateError{DuplicateIndices: []int{firstIdx, i}}
           }
           seen[exec.State] = i
       }
       return nil
   }
   ```
3. Call validation at the start of `ExecuteAllParallel`.
4. Add tests for shared-state detection.

## Decision Log

- 2026-02-21: Initial decision accepted
- Referenced by: `docs/critical-review-implementation-plan-2026-02-21.md` Phase 0.2
