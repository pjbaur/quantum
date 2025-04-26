# Schrödinger's Gopher

This project provides a basic simulation of quantum computing concepts using the Go programming language. It focuses on the fundamental building block of quantum computing: the qubit, and implements core operations like the Hadamard gate and measurement.

## Features

- Single and multi-qubit simulation
- Hadamard, Pauli (X, Y, Z), S, T, and CNOT gates
- Measurement and probability calculations
- Demonstrations of superposition, entanglement, and quantum teleportation
- Modular, testable Go code

## Getting Started

### Prerequisites

- Go (version 1.18 or later)

### Installation

1. Clone the repository:

    ```bash
    git clone https://github.com/pjbaur/quantum.git
    cd quantum
    ```

2. Run the main program:

    ```bash
    go run main.go
    ```

    This will display a menu of available quantum computing demonstrations, including Hadamard, T-gate, and Bell state examples.

3. Run the tests:

    ```bash
    go test
    ```

    This will run all unit tests in [`quantum_test.go`](quantum_test.go).

## Usage

The main entry point is [`main.go`](main.go), which provides a command-line interface to run various quantum computing demonstrations. Example usage:

```bash
go run main.go hadamard   # Run Hadamard gate demonstrations
go run main.go tgate      # Run T-gate demonstrations
go run main.go bell       # Run Bell state demonstrations
go run main.go all        # Run all demonstrations sequentially
```

### Project Structure

- [`main.go`](main.go): Entry point and CLI for running demonstrations.
- [`qubit/qubit.go`](qubit/qubit.go): Single qubit representation and operations.
- [`state/state.go`](state/state.go): Multi-qubit quantum state and gate application.
- [`gates/gates.go`](gates/gates.go): Definitions of quantum gates (H, X, Y, Z, S, T, CNOT, etc).
- [`measurement/measurement.go`](measurement/measurement.go): Measurement operations for qubits and quantum states.
- [`examples/`](examples/): Example programs and demonstrations:
  - [`hadamard.go`](examples/hadamard.go): Hadamard gate and superposition.
  - [`tgate.go`](examples/tgate.go): T-gate and phase operations.
  - [`bell.go`](examples/bell.go): Bell states, entanglement, and teleportation.

## Testing

Run all tests with:

```bash
go test
```

See [`quantum_test.go`](quantum_test.go) for comprehensive unit tests covering qubit operations, gates, measurement, and multi-qubit states.

---

For more details on quantum gates and their matrix representations, see [QUANTUM-HELP.md](QUANTUM-HELP.md).

## Expansion

To extend this further, you could:
- Add more gates (X, Y, Z, CNOT, etc.)
- Implement entanglement operations
- Add error checking for invalid qubit indices
- **Error Handling**: Check normalization (\( |\alpha|^2 + |\beta|^2 = 1 \)) after operations.
- Add methods to access individual qubit states
- Quantum Circuit abstraction: Add a circuit model to compose operations more easily.
- Gate decomposition: Support for decomposing complex operations into your basic gates.
- Density matrix representation: For mixed states and noisy simulations.
- Performance optimizations: Consider sparse representations for states with many zeros.
- Visualization tools: Add methods to visualize quantum states (Bloch sphere for single qubits).

## Refactoring

quantum/
├── gates/
│   ├── gates.go        # Gate definitions (H, T, etc.)
│   └── operations.go   # Gate application logic
├── qubit/
│   └── qubit.go        # Single qubit representation
├── state/
│   └── state.go        # Multi-qubit state representation  
├── measurement/
│   └── measurement.go  # Measurement operations
├── examples/
│   ├── hadamard.go     # Hadamard examples
│   ├── tgate.go        # T-gate examples
│   └── bell.go         # Bell state examples
└── main.go             # Entry point

Key Benefits:

Separation of concerns: Each file would handle a specific aspect of quantum simulation
Better testability: You could write focused tests for each component
Easier maintenance: Smaller files are easier to understand and modify
Better collaboration: Multiple developers could work on different parts simultaneously
Clearer imports: Dependencies between components would be more explicit


Implementation Approach:

Start by identifying logical groupings in your code
Move related functions and types into their own files
Ensure each file has a clear purpose and responsibility
Maintain consistent naming conventions across files
Use interfaces where appropriate to define clear boundaries


Additional Improvements:

Add interfaces for circuit design patterns
Create a separate package for common quantum algorithms
Add a visualization package for quantum states



This approach would make your code more manageable as you implement additional gates (like S, X, Y, Z gates) or more advanced quantum algorithms in the future.