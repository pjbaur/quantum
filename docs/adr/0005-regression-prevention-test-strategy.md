# ADR-0005: Regression Prevention Test Strategy

## Status

Accepted

## Context

The critical review identified significant test coverage gaps that increase the risk of regressions during the planned refactoring work:

1. **Missing Negative Path Tests**: Algorithms (`DeutschJozsa`, `Grover`) only test happy paths
2. **Missing Misuse Path Tests**: `ExecuteAllParallel` has no tests for shared-state misuse detection
3. **Missing Helper-Level Tests**: Visualization helpers (`FormatBlochVector`, `BlochCSV`) lack direct unit tests
4. **No Race Test Expectations**: Concurrency-sensitive packages have no documented race testing requirements

This ADR defines the minimum test matrix and expectations to prevent regressions.

## Decision

### 1. Negative Path Test Requirements

Every public function that returns an error must have tests covering all error paths.

#### Required Test Categories

| Category | Description | Assertion Requirement |
|----------|-------------|----------------------|
| **Invalid Input** | Bad parameters (nil, empty, out of range) | Assert specific error type via `errors.As` |
| **State Violation** | Operations that violate quantum state invariants | Assert error message contains relevant context |
| **Boundary Conditions** | Edge cases (0 qubits, max qubits, empty sets) | Assert behavior is documented (error or valid) |

#### Error Assertion Pattern

```go
// Required: Use errors.As to assert specific error types
var errType *quantum.SomeErrorType
if !errors.As(err, &errType) {
    t.Fatalf("expected SomeErrorType, got %T: %v", err, err)
}

// Required: Assert error contains actionable information
if !strings.Contains(err.Error(), "expected substring") {
    t.Fatalf("error message lacks context: %v", err)
}
```

### 2. Misuse Path Test Requirements

Functions with implicit caller contracts must have explicit misuse detection tests.

#### `ExecuteAllParallel` Misuse Tests

| Misuse Scenario | Expected Behavior | Test Requirement |
|-----------------|-------------------|------------------|
| Same state pointer in multiple executions | Return `SharedStateError` | Must detect and report duplicate indices |
| nil state in execution | Return indexed error | Must include execution index in message |
| nil circuit in execution | Return indexed error | Must include execution index in message |
| Empty executions slice | Return nil | Must handle gracefully |

#### Misuse Test Pattern

```go
func TestExecuteAllParallelSharedState(t *testing.T) {
    s := state.New(1)
    c, _ := circuit.New(1)

    executions := []circuit.Execution{
        {Circuit: c, State: s},
        {Circuit: c, State: s}, // Same pointer - misuse
    }

    err := circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{})

    var sharedErr *circuit.SharedStateError
    if !errors.As(err, &sharedErr) {
        t.Fatalf("expected SharedStateError, got %T", err)
    }
    if len(sharedErr.DuplicateIndices) < 2 {
        t.Fatalf("expected at least 2 duplicate indices, got %v", sharedErr.DuplicateIndices)
    }
}
```

### 3. Helper-Level Output Tests

Pure functions in utility packages require direct unit tests with table-driven cases.

#### Visualization Helper Test Matrix

| Function | Test Cases |
|----------|------------|
| `FormatBlochVector` | Zero vector, unit vectors (x/y/z), arbitrary values, precision=0, negative precision |
| `BlochCSV` | Zero vector, sign handling, precision formatting, delimiter correctness |
| `BlochVectorFromQubit` | Nil input (empty return), |0⟩ state, |1⟩ state, superposition states |

#### Helper Test Pattern

```go
func TestFormatBlochVector(t *testing.T) {
    tests := []struct {
        name      string
        vector    BlochVector
        precision int
        want      string
    }{
        {
            name:      "zero vector default precision",
            vector:    BlochVector{X: 0, Y: 0, Z: 0},
            precision: 4,
            want:      "x=0.0000 y=0.0000 z=0.0000",
        },
        {
            name:      "negative precision uses default",
            vector:    BlochVector{X: 1, Y: 0, Z: 0},
            precision: -1,
            want:      "x=1.0000 y=0.0000 z=0.0000",
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FormatBlochVector(tt.vector, tt.precision)
            if got != tt.want {
                t.Fatalf("FormatBlochVector() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

### 4. Race-Focused Test Command Expectations

Concurrency-sensitive packages must pass the race detector.

#### Concurrency-Sensitive Packages

| Package | Race-Sensitive Function | Reason |
|---------|------------------------|--------|
| `circuit` | `ExecuteAllParallel` | Concurrent goroutines access executions |
| `internal/sparsestate` | Any gate application | Internal state mutation |
| `state` | Any mutation operation | Non-thread-safe by design |

#### Race Test Requirements

1. **Test Construction**: Tests must create concurrent scenarios that the race detector can analyze
2. **Multiple Iterations**: Race conditions are probabilistic; tests must run enough iterations
3. **Command**: `go test -race ./...` must pass with zero reports

#### Race Test Pattern

```go
func TestExecuteAllParallelNoRace(t *testing.T) {
    // Create many independent executions to stress-test parallelism
    numExecutions := 100
    executions := make([]circuit.Execution, numExecutions)

    for i := 0; i < numExecutions; i++ {
        c, _ := circuit.New(1)
        _ = c.AddGate(gates.NewPauliX(), 0)
        executions[i] = circuit.Execution{
            Circuit: c,
            State:   state.New(1),
        }
    }

    // This test must pass under: go test -race
    err := circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 10})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

#### CI Race Test Expectations

```bash
# Required: Full race detection on all packages
go test -race ./...

# Recommended: Run race tests with increased iterations for flaky detection
go test -race -count=5 ./circuit/...
go test -race -count=5 ./internal/sparsestate/...
```

### 5. Test Organization Requirements

#### File Naming Conventions

| Test Type | File Pattern | Example |
|-----------|--------------|---------|
| Unit tests | `*_test.go` | `state_test.go` |
| Benchmark tests | `*_bench_test.go` | `state_bench_test.go` |
| Race-specific tests | Include in `*_test.go` with `// Race:` comment | `parallel_test.go` |

#### Table-Driven Test Requirements

- Use `t.Run()` for individual cases
- Include `name` field for test identification
- Use `t.Fatalf()` for failures with context
- Validate both success and error conditions in the same table where applicable

## Minimum New Test Matrix

### Algorithms Package (`algorithm/`)

| Function | Negative Path Tests Required |
|----------|------------------------------|
| `DeutschJozsa` | `numInputQubits <= 0`, `oracle == nil`, oracle returns invalid value |
| `Grover` | `numQubits <= 0`, empty `marked` slice, `marked` state out of range |

### Circuit Package (`circuit/`)

| Function | Negative/Misuse Path Tests Required |
|----------|-------------------------------------|
| `ExecuteAllParallel` | Shared state pointer, nil circuit, nil state, empty slice |
| `ExecuteAll` | Same as parallel (serial validation) |
| `New` | Invalid qubit count (per ADR-0004) |

### Visualization Package (`visualization/`)

| Function | Test Cases Required |
|----------|---------------------|
| `FormatBlochVector` | Zero vector, precision edge cases, formatting verification |
| `BlochCSV` | CSV format correctness, precision, sign handling |
| `BlochVectorFromQubit` | Nil input, basis states, superposition |

### State Package (`state/`, `internal/sparsestate/`)

| Function | Negative Path Tests Required |
|----------|------------------------------|
| `New` | Invalid qubit count (per ADR-0004) |
| `SetAmplitude` | Non-normalizing assignment (per ADR-0004 error info) |
| `ApplyGate` | Target out of range, wrong target count (existing) |

## Consequences

### Positive

1. **Regression Prevention**: Required tests catch errors before they reach production
2. **Documentation**: Tests serve as executable documentation of expected behavior
3. **Refactoring Safety**: Changes can be made confidently with test coverage
4. **Debugging**: Error-specific assertions make failures easier to diagnose

### Negative

1. **Initial Effort**: Writing comprehensive negative/misuse tests requires time
2. **Maintenance**: More tests mean more test code to maintain
3. **False Confidence**: Tests must accurately reflect invariants to be valuable

### Neutral

- Test coverage metrics should improve but are not the primary goal
- The focus is on meaningful tests, not coverage percentages

## Implementation Checklist

- [ ] Add negative path tests for `DeutschJozsa` (3+ error cases)
- [ ] Add negative path tests for `Grover` (3+ error cases)
- [ ] Add shared-state misuse test for `ExecuteAllParallel`
- [ ] Add race-focused test for `ExecuteAllParallel`
- [ ] Add direct unit tests for `FormatBlochVector` (4+ cases)
- [ ] Add direct unit tests for `BlochCSV` (4+ cases)
- [ ] Add nil input test for `BlochVectorFromQubit`
- [ ] Verify `go test -race ./...` passes

## Decision Log

- 2026-02-21: Initial decision accepted
- Referenced by: `docs/critical-review-implementation-plan-2026-02-21.md` Phase 0.6
