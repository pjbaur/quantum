# Sparse State Backend

## Overview

The sparse state backend provides memory-efficient quantum state representation for circuits with limited entanglement. It stores only non-zero amplitudes, making it suitable for large qubit counts where the dense backend would be impractical.

## Criteria for Sparse State Use

Sparse state representation is most beneficial when:

- Qubit count is high enough that 2^n amplitudes are impractical to store.
- The number of non-zero amplitudes is far smaller than the full basis size.
- Gate sequences preserve sparsity (e.g., X, Z, CNOT, permutations), or sparsity only grows slowly.

Practical rule of thumb:

- Prefer sparse state when non-zero amplitude count <= 1% of the full basis size.
- Avoid sparse state when applying dense-generating gates (e.g., Hadamard across many qubits).

## Backend Capabilities

The sparse backend has **explicit capability limits** that differ from the dense backend. Use the `BackendCapabilities` interface to check compatibility before execution.

### Supported Operations

| Operation | Support |
|-----------|---------|
| Single-qubit gates (2x2 matrix) | Supported |
| Two-qubit gates (4x4 matrix) | Supported (CNOT optimized) |
| Three+ qubit gates (8x8+ matrix) | **Not Supported** |
| Measurement | Supported |
| Cloning | Supported |
| Amplitude/probability queries | Supported |
| Bulk amplitude writes (`SetAmplitudes`) | Supported (amplitudes at or below 1e-12 are pruned, not stored) |

### Capability API

```go
// BackendCapabilities describes what operations a backend supports
type BackendCapabilities interface {
    // SupportsGateQubits returns whether this backend can apply gates
    // operating on the specified number of qubits.
    SupportsGateQubits(qubitCount int) bool

    // MaxGateQubits returns the maximum number of qubits a gate can operate on.
    // Returns 0 if there is no limit.
    MaxGateQubits() int
}
```

### Checking Capabilities

```go
sparse := sparsestate.New(10)
caps := sparse.(quantum.BackendCapabilities)

if !caps.SupportsGateQubits(3) {
    fmt.Printf("Sparse backend only supports up to %d-qubit gates\n",
        caps.MaxGateQubits())  // Output: 2
}
```

### Unsupported Operations

When attempting to apply a 3+ qubit gate, the sparse backend returns an `UnsupportedOperationError`:

```go
toffoli := gates.NewToffoli()  // 3-qubit gate
err := sparse.ApplyGate(toffoli, 0, 1, 2)
// Returns: &UnsupportedOperationError{
//     Operation: "3-qubit gate application",
//     Backend: "sparse",
//     Alternative: "dense state backend (state.State)",
// }
```

## When to Use Dense Backend Instead

Use the dense backend (`state.State`) when:

- Circuits contain 3+ qubit gates (Toffoli, Fredkin, etc.)
- Qubit count is small (< 25 qubits)
- Maximum feature compatibility is required

Use the sparse backend (`sparsestate.State`) when:

- Large qubit counts with limited entanglement
- Memory-constrained environments
- Circuits use only 1-2 qubit gates

## Accuracy and Performance Comparison

Accuracy is verified by tests that apply the same gate sequences to dense and sparse states and compare amplitudes and probabilities.

Benchmarks to compare sparse and dense performance are defined in `internal/sparsestate/bench_test.go` and can be run with:

```
go test ./internal/sparsestate -bench=Benchmark -run=^$
```

Benchmarks on Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz (darwin/amd64):

- Sparse vs dense single-qubit gate: 692.5 ns/op vs 35,504 ns/op
- Sparse vs dense CNOT: 543.5 ns/op vs 75,047 ns/op
- Sparse vs dense probability lookup: 46.89 ns/op vs 40.71 ns/op

## Implementation Details

The sparse state backend is implemented in the `internal/sparsestate` package.

- Single-qubit gates: Generic 2x2 matrix application
- Two-qubit gates: Generic 4x4 matrix application (CNOT optimized by index permutation)
- Measurement, cloning, and amplitude/probability queries: Full support
- Bulk amplitude writes: `SetAmplitudes` satisfies `quantum.BulkAmplitudeSetter`,
  validating length, finiteness, and normalization in that order exactly as the
  dense backend does. This is what lets the `algorithm` package run on either
  backend.

## References

- `docs/adr/0003-sparse-backend-capability.md` - Capability model decision
- `docs/MIGRATION-v2.md` - Migration guide for backend changes
- `internal/sparsestate/` - Implementation
