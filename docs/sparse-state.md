# Sparse State Prototype (Workstream 2)

## Criteria for Sparse State Use

Sparse state representation is most beneficial when:

- Qubit count is high enough that 2^n amplitudes are impractical to store.
- The number of non-zero amplitudes is far smaller than the full basis size.
- Gate sequences preserve sparsity (e.g., X, Z, CNOT, permutations), or sparsity only grows slowly.

Practical rule of thumb for this prototype:

- Prefer sparse state when non-zero amplitude count <= 1% of the full basis size.
- Avoid sparse state when applying dense-generating gates (e.g., Hadamard across many qubits).

## Prototype Scope

The sparse state prototype is implemented in the internal `sparsestate` package and supports:

- Single-qubit gates via generic 2x2 matrix application.
- Two-qubit gates via 4x4 matrix application (with CNOT optimized by index permutation).
- Measurement, cloning, and amplitude/probability queries.

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

## Adoption Path (No Public API Changes)

- Keep sparse state internal until gates beyond single-qubit and CNOT are supported.
- Add a factory or internal toggle in the `state` package once criteria and benchmarks justify.
- If adopted, keep the `quantum.QuantumState` interface unchanged and choose dense vs sparse internally based on the criteria above.
