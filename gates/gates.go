// Package gates contains the definition of quantum gates used in the simulator.
package gates

import (
	"fmt"
	"math"
)

// mustGate builds a built-in gate and panics if its matrix table is invalid.
// A panic here is a programmer error in this package, caught by tests.
func mustGate(name string, matrix [][]complex128) *MatrixGate {
	gate, err := NewMatrixGate(name, matrix)
	if err != nil {
		panic(fmt.Sprintf("invalid built-in gate %q: %v", name, err))
	}
	return gate
}

// NewHadamard creates a new Hadamard gate.
func NewHadamard() *MatrixGate {
	return mustGate("Hadamard", [][]complex128{
		{1.0 / complex(math.Sqrt(2), 0), 1.0 / complex(math.Sqrt(2), 0)},
		{1.0 / complex(math.Sqrt(2), 0), -1.0 / complex(math.Sqrt(2), 0)},
	})
}

// NewPauliX creates a new Pauli-X (NOT) gate.
func NewPauliX() *MatrixGate {
	return mustGate("PauliX", [][]complex128{
		{0, 1},
		{1, 0},
	})
}

// NewPauliY creates a new Pauli-Y gate.
func NewPauliY() *MatrixGate {
	return mustGate("PauliY", [][]complex128{
		{0, complex(0, -1)},
		{complex(0, 1), 0},
	})
}

// NewPauliZ creates a new Pauli-Z gate.
func NewPauliZ() *MatrixGate {
	return mustGate("PauliZ", [][]complex128{
		{1, 0},
		{0, -1},
	})
}

// NewS creates a new S (phase) gate.
func NewS() *MatrixGate {
	return mustGate("S", [][]complex128{
		{1, 0},
		{0, complex(0, 1)},
	})
}

// NewT creates a new T (π/8) gate.
func NewT() *MatrixGate {
	return mustGate("T", [][]complex128{
		{1, 0},
		{0, complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))},
	})
}

// NewCNOT creates a new Controlled-NOT gate (4x4, 2-qubit).
func NewCNOT() *MatrixGate {
	return mustGate("CNOT", [][]complex128{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 0, 1},
		{0, 0, 1, 0},
	})
}

// NewSwap creates a new SWAP gate which exchanges two qubits.
func NewSwap() *MatrixGate {
	return mustGate("SWAP", [][]complex128{
		{1, 0, 0, 0},
		{0, 0, 1, 0},
		{0, 1, 0, 0},
		{0, 0, 0, 1},
	})
}
