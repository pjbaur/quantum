// Package gates contains the definition of quantum gates used in the simulator.
package gates

import (
	"math"
)

// HadamardGate implements the Hadamard gate
type HadamardGate struct{}

// NewHadamard creates a new Hadamard gate
func NewHadamard() *HadamardGate {
	return &HadamardGate{}
}

// Name returns the name of the gate
func (g *HadamardGate) Name() string {
	return "Hadamard"
}

// Matrix returns the matrix representation of the gate
func (g *HadamardGate) Matrix() [][]complex128 {
	return [][]complex128{
		{1.0 / complex(math.Sqrt(2), 0), 1.0 / complex(math.Sqrt(2), 0)},
		{1.0 / complex(math.Sqrt(2), 0), -1.0 / complex(math.Sqrt(2), 0)},
	}
}

// PauliXGate implements the Pauli-X (NOT) gate
type PauliXGate struct{}

// NewPauliX creates a new Pauli-X gate
func NewPauliX() *PauliXGate {
	return &PauliXGate{}
}

// Name returns the name of the gate
func (g *PauliXGate) Name() string {
	return "PauliX"
}

// Matrix returns the matrix representation of the gate
func (g *PauliXGate) Matrix() [][]complex128 {
	return [][]complex128{
		{0, 1},
		{1, 0},
	}
}

// PauliYGate implements the Pauli-Y gate
type PauliYGate struct{}

// NewPauliY creates a new Pauli-Y gate
func NewPauliY() *PauliYGate {
	return &PauliYGate{}
}

// Name returns the name of the gate
func (g *PauliYGate) Name() string {
	return "PauliY"
}

// Matrix returns the matrix representation of the gate
func (g *PauliYGate) Matrix() [][]complex128 {
	return [][]complex128{
		{0, complex(0, -1)},
		{complex(0, 1), 0},
	}
}

// PauliZGate implements the Pauli-Z gate
type PauliZGate struct{}

// NewPauliZ creates a new Pauli-Z gate
func NewPauliZ() *PauliZGate {
	return &PauliZGate{}
}

// Name returns the name of the gate
func (g *PauliZGate) Name() string {
	return "PauliZ"
}

// Matrix returns the matrix representation of the gate
func (g *PauliZGate) Matrix() [][]complex128 {
	return [][]complex128{
		{1, 0},
		{0, -1},
	}
}

// SGate implements the S (phase) gate
type SGate struct{}

// NewS creates a new S gate
func NewS() *SGate {
	return &SGate{}
}

// Name returns the name of the gate
func (g *SGate) Name() string {
	return "S"
}

// Matrix returns the matrix representation of the gate
func (g *SGate) Matrix() [][]complex128 {
	return [][]complex128{
		{1, 0},
		{0, complex(0, 1)},
	}
}

// TGate implements the T (π/8) gate
type TGate struct{}

// NewT creates a new T gate
func NewT() *TGate {
	return &TGate{}
}

// Name returns the name of the gate
func (g *TGate) Name() string {
	return "T"
}

// Matrix returns the matrix representation of the gate
func (g *TGate) Matrix() [][]complex128 {
	return [][]complex128{
		{1, 0},
		{0, complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))},
	}
}

// CNOTGate implements the Controlled-NOT gate
type CNOTGate struct{}

// NewCNOT creates a new CNOT gate
func NewCNOT() *CNOTGate {
	return &CNOTGate{}
}

// Name returns the name of the gate
func (g *CNOTGate) Name() string {
	return "CNOT"
}

// Matrix returns the matrix representation of the gate
// Note: This is a 4x4 matrix for a 2-qubit operation
func (g *CNOTGate) Matrix() [][]complex128 {
	return [][]complex128{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 0, 1},
		{0, 0, 1, 0},
	}
}

// SwapGate implements the SWAP gate which exchanges two qubits
type SwapGate struct{}

// NewSwap creates a new SWAP gate
func NewSwap() *SwapGate {
	return &SwapGate{}
}

// Name returns the name of the gate
func (g *SwapGate) Name() string {
	return "SWAP"
}

// Matrix returns the matrix representation of the gate
func (g *SwapGate) Matrix() [][]complex128 {
	return [][]complex128{
		{1, 0, 0, 0},
		{0, 0, 1, 0},
		{0, 1, 0, 0},
		{0, 0, 0, 1},
	}
}
