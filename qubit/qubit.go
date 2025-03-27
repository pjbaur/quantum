package qubit

import (
	"math/cmplx"
	"math/rand"

	"github.com/pjbaur/quantum/gates"
)

// Qubit represents a single quantum bit with complex amplitudes
type Qubit struct {
	Alpha complex128 // Amplitude for |0⟩ state
	Beta  complex128 // Amplitude for |1⟩ state
}

// ApplyMatrix applies a 2x2 unitary matrix to the qubit's state
func (q *Qubit) ApplyMatrix(matrix [2][2]complex128) {
	newAlpha := matrix[0][0]*q.Alpha + matrix[0][1]*q.Beta
	newBeta := matrix[1][0]*q.Alpha + matrix[1][1]*q.Beta

	q.Alpha = newAlpha
	q.Beta = newBeta
}

// ApplyHadamard applies the Hadamard gate to the qubit
func (q *Qubit) ApplyHadamard() {
	q.ApplyMatrix(gates.H)
}

// ApplyX applies the Pauli-X gate to the qubit
func (q *Qubit) ApplyX() {
	q.ApplyMatrix(gates.X)
}

// ApplyZ applies the Pauli-Z gate to the qubit
func (q *Qubit) ApplyZ() {
	q.ApplyMatrix(gates.Z)
}

// ApplyT applies the T gate to the qubit
func (q *Qubit) ApplyT() {
	q.ApplyMatrix(gates.T)
}

// ApplyS applies the S gate to the qubit
func (q *Qubit) ApplyS() {
	q.ApplyMatrix(gates.S)
}

// ApplyY applies the Pauli-Y gate to the qubit
func (q *Qubit) ApplyY() {
	q.ApplyMatrix(gates.Y)
}

// ApplyCNot applies the CNOT gate to the qubit
func (q *Qubit) ApplyCNot(control int) {
	q.ApplyMatrix(gates.CNot(control))
}

// ApplyCZ applies the CZ gate to the qubit
func (q *Qubit) ApplyCZ(control int) {
	q.ApplyMatrix(gates.CZ(control))
}

// NewQubit creates a new qubit initialized to |0⟩ state
func NewQubit() *Qubit {
	return &Qubit{
		Alpha: 1.0 + 0i,
		Beta:  0.0 + 0i,
	}
}

// NewQubitFromValues creates a new qubit with specified amplitude values
func NewQubitFromValues(alpha, beta complex128) *Qubit {
	return &Qubit{
		Alpha: alpha,
		Beta:  beta,
	}
}

// Measure collapses the qubit to either |0⟩ or |1⟩ based on probability
// Returns the result (0 or 1) and updates the qubit's state
func (q *Qubit) Measure() int {
	prob0 := cmplx.Abs(q.Alpha) * cmplx.Abs(q.Alpha)
	if rand.Float64() < prob0 {
		q.Alpha = 1.0 + 0i
		q.Beta = 0.0 + 0i
		return 0
	}
	q.Alpha = 0.0 + 0i
	q.Beta = 1.0 + 0i
	return 1
}

// Probability0 returns the probability of measuring |0⟩
func (q *Qubit) Probability0() float64 {
	return cmplx.Abs(q.Alpha) * cmplx.Abs(q.Alpha)
}

// Probability1 returns the probability of measuring |1⟩
func (q *Qubit) Probability1() float64 {
	return cmplx.Abs(q.Beta) * cmplx.Abs(q.Beta)
}

// IsNormalized checks if the qubit state is properly normalized
func (q *Qubit) IsNormalized() bool {
	sum := q.Probability0() + q.Probability1()
	return cmplx.Abs(complex(sum, 0)-complex(1.0, 0)) < 1e-10
}

// Clone creates a copy of the qubit
func (q *Qubit) Clone() *Qubit {
	return &Qubit{
		Alpha: q.Alpha,
		Beta:  q.Beta,
	}
}
