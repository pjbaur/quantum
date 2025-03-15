# Quantum Computing Simulation in Go

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
