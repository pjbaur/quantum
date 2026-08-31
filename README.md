# Schrödinger's Gopher

This project provides a basic simulation of quantum computing concepts using the Go programming language. It focuses on the fundamental building block of quantum computing: the qubit, and implements core operations like the Hadamard gate, multi-qubit state evolution, and measurement.

## Features

- Single- and multi-qubit simulation with a shared `quantum` interface layer
- Hadamard, Pauli (X, Y, Z), S, T, CNOT, SWAP, and Toffoli gates
- Parameterized gates: `Rx`/`Ry`/`Rz` rotations, `Phase`, and a controlled-U construction that turns any gate into its controlled form
- Measurement and probability calculations for qubits and quantum states
- Reproducible measurements: inject a seeded source with `SetRandSource` (defaults to `math/rand`'s global source)
- Circuit abstraction for sequencing gate operations
- Parallel batch execution for independent circuit runs with configurable worker limits
- Demonstrations of superposition, entanglement, and quantum teleportation
- Visualization helpers for state tables and Bloch vectors
- Modular, testable Go code
- Little-endian qubit indexing (qubit 0 is the least-significant bit)

## Getting Started

### Prerequisites

- Go (version 1.25 or later)

### Installation

1. Clone the repository:

    ```bash
    git clone https://github.com/pjbaur/quantum.git
    cd quantum
    ```

2. Run the main program:

    ```bash
    go run ./cmd/quantum
    ```

    This will display a menu of available quantum computing demonstrations, including Hadamard, T-gate, and Bell state examples.

3. Run the tests:

    ```bash
    go test ./...
    ```

    This will run all unit tests across the module.

## Usage

The main entry point is [`cmd/quantum/main.go`](cmd/quantum/main.go), which provides a command-line interface to run various quantum computing demonstrations. Example usage (from the repository root):

```bash
go run ./cmd/quantum hadamard   # Run Hadamard gate demonstrations
go run ./cmd/quantum tgate      # Run T-gate demonstrations
go run ./cmd/quantum bell       # Run Bell state demonstrations
go run ./cmd/quantum algorithm  # Run Deutsch-Jozsa and Grover demonstrations
go run ./cmd/quantum visual     # Run visualization demonstrations
go run ./cmd/quantum noise      # Run noise channel demonstrations (density matrices)
go run ./cmd/quantum gates      # Run gate catalog and decomposition demonstrations
go run ./cmd/quantum gate       # List the gates in the built-in registry
go run ./cmd/quantum gate CNOT  # Print one registered gate's qubit count and matrix
go run ./cmd/quantum all        # Run all demonstrations sequentially
```

### Visualization

Use the `visualization` package to format multi-qubit state tables and export Bloch vectors for plotting. Example usage:

```go
opts := visualization.DefaultStateViewOptions()
opts.MinProbability = 0.001
fmt.Println(visualization.FormatStateView(state, opts))

vector := visualization.BlochVectorFromQubit(qubit)
fmt.Println(visualization.FormatBlochVector(vector, 4))
fmt.Println(visualization.BlochCSV(vector, 4))
```

### Project Structure

- [`cmd/quantum/main.go`](cmd/quantum/main.go): Entry point and CLI for running demonstrations.
- [`algorithm/`](algorithm/): Algorithm drivers (Deutsch-Jozsa, Grover, teleportation) plus Pauli-sum Hamiltonians and the VQE loop with parameter-shift gradients.
- [`circuit/`](circuit/): Circuit abstraction for sequencing gate operations.
- [`gates/`](gates/): Data-driven gate definitions (H, X, Y, Z, S, T, CNOT, SWAP, Toffoli), the parameterized `Rx`/`Ry`/`Rz`/`Phase` constructors, `NewControlled` for controlled-U, `MatrixGate` for custom gates, the built-in registry, and SWAP decomposition.
- [`parameterized/`](parameterized/): Circuit templates with named symbolic parameters; `Bind` materializes a concrete circuit.
- [`quantum/`](quantum/): Core interfaces and error types shared across packages.
- [`qubit/qubit.go`](qubit/qubit.go): Single qubit representation and operations.
- [`state/state.go`](state/state.go): Multi-qubit quantum state and gate application.
- [`visualization/`](visualization/): Text-based state views and Bloch vector export helpers.
- [`internal/examples/`](internal/examples/): Example programs and demonstrations:
  - [`hadamard.go`](internal/examples/hadamard.go): Hadamard gate and superposition.
  - [`tgate.go`](internal/examples/tgate.go): T-gate and phase operations.
  - [`bell.go`](internal/examples/bell.go): Bell states, entanglement, and teleportation.
  - [`algorithm.go`](internal/examples/algorithm.go): Deutsch-Jozsa and Grover algorithms, plus Grover's diffusion operator built from gates and compared against the algorithm's fast path.
  - [`visualization.go`](internal/examples/visualization.go): State table and Bloch vector outputs.
  - [`noise.go`](internal/examples/noise.go): Noise channels on the density-matrix backend.
  - [`gates.go`](internal/examples/gates.go): Gate catalog and SWAP decomposition demonstration.
- [`internal/density/`](internal/density/): Density-matrix backend for mixed states and noise channels (dephasing, amplitude damping, depolarizing).
- [`internal/backendmath/`](internal/backendmath/): State-vector math shared by the dense and sparse backends: target validation, combo-mask construction, and measurement collapse.
- Measurement helpers live on `state.State` and `qubit.Qubit`.

### Parallel Circuit Execution

Independent circuits that operate on distinct quantum states can be executed in
parallel. Use `MaxParallelism` to cap concurrency.

```go
executions := []circuit.Execution{
	{Circuit: c1, State: s1},
	{Circuit: c2, State: s2},
}
err := circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 4})
```

## Testing

Run all tests with:

```bash
go test ./...
```

Tests live in [`qubit/qubit_test.go`](qubit/qubit_test.go), [`quantum/quantum_test.go`](quantum/quantum_test.go), [`state/state_test.go`](state/state_test.go), [`circuit/circuit_test.go`](circuit/circuit_test.go), and [`gates/`](gates/) (`gates_test.go`, `matrixgate_test.go`, `matrix_test.go`, `registry_test.go`, `decompose_test.go`), covering qubit operations, gates and gate validation, the built-in gate registry, SWAP decomposition, measurement, circuits, and multi-qubit states.

## Benchmarking
```bash
go test ./state -bench Benchmark -benchmem -run '^$'

mkdir -p CHANGES/profiles

go test ./circuit -bench BenchmarkCircuitExecute \
    -benchmem \
    -run '^$' \
    -cpuprofile CHANGES/profiles/ws1-circuit-cpu.pprof \
    -memprofile CHANGES/profiles/ws1-circuit-mem.pprof
```

Inspect profiles with `go tool pprof CHANGES/profiles/ws1-circuit-cpu.pprof`

Profiles are local artifacts and are not tracked in git.

Run full state/sparse comparison suite: go test ./state ./internal/sparsestate

## Demos

### Visualization
`go run ./cmd/quantum visual`

## Versioning

Releases are tagged in the v0.x series (current: `v0.3.0`). The "v2" in
[`docs/MIGRATION-v2.md`](docs/MIGRATION-v2.md) and
[`docs/deprecation-policy-v2.md`](docs/deprecation-policy-v2.md) names the
current API generation, not a Go module major version.

[`CHANGELOG.md`](CHANGELOG.md) records what changed in each release, including
the unreleased work already on `main`.

## License

MIT — see [LICENSE](LICENSE).

---

For more details on quantum gates and their matrix representations, see [`docs/QUANTUM-HELP.md`](docs/QUANTUM-HELP.md).
