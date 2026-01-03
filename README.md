# Schrödinger's Gopher

This project provides a basic simulation of quantum computing concepts using the Go programming language. It focuses on the fundamental building block of quantum computing: the qubit, and implements core operations like the Hadamard gate, multi-qubit state evolution, and measurement.

## Features

- Single- and multi-qubit simulation with a shared `quantum` interface layer
- Hadamard, Pauli (X, Y, Z), S, T, CNOT, and SWAP gates
- Measurement and probability calculations for qubits and quantum states
- Circuit abstraction for sequencing gate operations
- Demonstrations of superposition, entanglement, and quantum teleportation
- Modular, testable Go code
- Little-endian qubit indexing (qubit 0 is the least-significant bit)

## Getting Started

### Prerequisites

- Go (version 1.21 or later)

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
go run ./cmd/quantum all        # Run all demonstrations sequentially
```

### Project Structure

- [`cmd/quantum/main.go`](cmd/quantum/main.go): Entry point and CLI for running demonstrations.
- [`circuit/`](circuit/): Circuit abstraction for sequencing gate operations.
- [`gates/gates.go`](gates/gates.go): Definitions of quantum gates (H, X, Y, Z, S, T, CNOT, SWAP, etc).
- [`quantum/`](quantum/): Core interfaces and error types shared across packages.
- [`qubit/qubit.go`](qubit/qubit.go): Single qubit representation and operations.
- [`state/state.go`](state/state.go): Multi-qubit quantum state and gate application.
- [`internal/examples/`](internal/examples/): Example programs and demonstrations:
  - [`hadamard.go`](internal/examples/hadamard.go): Hadamard gate and superposition.
  - [`tgate.go`](internal/examples/tgate.go): T-gate and phase operations.
  - [`bell.go`](internal/examples/bell.go): Bell states, entanglement, and teleportation.
  - [`algorithm.go`](internal/examples/algorithm.go): Deutsch-Jozsa and Grover algorithms.
- Measurement helpers currently live on `state.State` and `qubit.Qubit`; a dedicated `measurement` package is TODO.

## Testing

Run all tests with:

```bash
go test ./...
```

Tests live in [`quantum/quantum_test.go`](quantum/quantum_test.go), [`state/state_test.go`](state/state_test.go), and [`circuit/circuit_test.go`](circuit/circuit_test.go), covering qubit operations, gates, measurement, circuits, and multi-qubit states.

---

For more details on quantum gates and their matrix representations, see [`docs/QUANTUM-HELP.md`](docs/QUANTUM-HELP.md).
