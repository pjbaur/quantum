// Package gates contains the definition of quantum gates used in the simulator.
package gates

import (
	"math"

	"github.com/pjbaur/quantum/quantum"
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

// Apply applies the gate to the given qubit
func (g *HadamardGate) Apply(q quantum.Qubit) error {
	alpha := q.Alpha()
	beta := q.Beta()

	newAlpha := (alpha + beta) / complex(math.Sqrt(2), 0)
	newBeta := (alpha - beta) / complex(math.Sqrt(2), 0)

	return q.Set(newAlpha, newBeta)
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

// Apply applies the gate to the given qubit
func (g *PauliXGate) Apply(q quantum.Qubit) error {
	alpha := q.Alpha()
	beta := q.Beta()

	// X gate swaps the amplitudes
	return q.Set(beta, alpha)
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

// Apply applies the gate to the given qubit
func (g *PauliYGate) Apply(q quantum.Qubit) error {
	alpha := q.Alpha()
	beta := q.Beta()

	// Y gate swaps the amplitudes with a phase shift
	return q.Set(complex(0, -1)*beta, complex(0, 1)*alpha)
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

// Apply applies the gate to the given qubit
func (g *PauliZGate) Apply(q quantum.Qubit) error {
	alpha := q.Alpha()
	beta := q.Beta()

	// Z gate flips the phase of |1⟩
	return q.Set(alpha, -beta)
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

// Apply applies the gate to the given qubit
func (g *SGate) Apply(q quantum.Qubit) error {
	alpha := q.Alpha()
	beta := q.Beta()

	// S gate adds a π/2 phase to |1⟩
	return q.Set(alpha, complex(0, 1)*beta)
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

// Apply applies the gate to the given qubit
func (g *TGate) Apply(q quantum.Qubit) error {
	alpha := q.Alpha()
	beta := q.Beta()

	// T gate adds a π/4 phase to |1⟩
	phase := complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))
	return q.Set(alpha, phase*beta)
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

// Apply applies the gate to a single qubit
// This method will return an error as CNOT requires two qubits
func (g *CNOTGate) Apply(q quantum.Qubit) error {
	return &quantum.InvalidGateApplicationError{
		Gate:        g.Name(),
		RequiredLen: 2,
		ActualLen:   1,
	}
}

// ApplyControlled applies the CNOT gate to a target qubit based on a control qubit
// This is an additional method specific to multi-qubit gates
func (g *CNOTGate) ApplyControlled(control, target quantum.Qubit) error {
	// If control is |1⟩, apply X to target
	controlProb1 := control.Probability1()

	// If control has non-zero probability of being |1⟩, we need to check
	// If it's a pure |1⟩ state, just apply X to target
	if controlProb1 > 0.999 {
		xGate := NewPauliX()
		return xGate.Apply(target)
	} else if controlProb1 < 0.001 {
		// If control is |0⟩, do nothing to target
		return nil
	} else {
		// For superpositions, we should use a quantum state with both qubits
		// This simplified implementation won't handle entanglement correctly
		return &quantum.InvalidGateApplicationError{
			Gate:        g.Name(),
			RequiredLen: 2,
			ActualLen:   1,
		}
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

// Apply applies the gate to a single qubit
// This method will return an error as SWAP requires two qubits
func (g *SwapGate) Apply(q quantum.Qubit) error {
	return &quantum.InvalidGateApplicationError{
		Gate:        g.Name(),
		RequiredLen: 2,
		ActualLen:   1,
	}
}

// ApplySwap swaps the states of two qubits
func (g *SwapGate) ApplySwap(q1, q2 quantum.Qubit) error {
	// Save the original values
	alpha1 := q1.Alpha()
	beta1 := q1.Beta()
	alpha2 := q2.Alpha()
	beta2 := q2.Beta()

	// Set the new values
	if err := q1.Set(alpha2, beta2); err != nil {
		return err
	}
	if err := q2.Set(alpha1, beta1); err != nil {
		return err
	}

	return nil
}
