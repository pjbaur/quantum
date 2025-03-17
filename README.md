# Schrödinger's Gopher

This project provides a basic simulation of quantum computing concepts using the Go programming language. It focuses on the fundamental building block of quantum computing: the qubit, and implements core operations like the Hadamard gate and measurement.

## Features

-   **Qubit Representation:** Represents a qubit using complex numbers for alpha and beta amplitudes.
-   **Hadamard Gate:** Implements the Hadamard gate, a fundamental quantum gate that puts a qubit into superposition.
-   **Measurement:** Simulates the measurement of a qubit, collapsing it into either the |0> or |1> state with probabilities determined by the qubit's amplitudes.
-   **Testing:** Includes comprehensive unit tests to ensure the correctness of the implemented operations.

## Getting Started

### Prerequisites

-   Go (version 1.18 or later)

### Installation

1.  Clone the repository:

    ```bash
    git clone https://github.com/pjbaur/quantum.git
    cd quantum
    ```

2.  Run the main program:

    ```bash
    go run quantum.go
    ```

3. Run the tests:

    ```bash
    go test
    ```

## Usage

### `quantum.go`

The `quantum.go` file contains the main program, which demonstrates the basic usage of the qubit, Hadamard gate, and measurement.

```go
package main

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"
)

// ... (Qubit struct and methods) ...

func main() {
	trials := 10
	zeros := 0
	ones := 0

	for i := 0; i < trials; i++ {
		qubit := NewQubit()
		fmt.Printf("Initial state: Alpha=%.3f, Beta=%.3f\n", real(qubit.Alpha), real(qubit.Beta))

		qubit.ApplyHadamard()
		fmt.Printf("After Hadamard: Alpha=%.3f, Beta=%.3f\n", real(qubit.Alpha), real(qubit.Beta))

		result := qubit.Measure()
		fmt.Printf("Measured: %d\n", result)
		if result == 0 {
			zeros++
		} else {
			ones++
		}
	}

	fmt.Printf("\nResults: %d zeros, %d ones\n", zeros, ones)
}

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