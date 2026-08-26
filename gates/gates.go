// Package gates contains the definition of quantum gates used in the simulator.
package gates

import (
	"fmt"
	"math"
)

// invSqrt2 is 1/√2, computed with math.Sqrt at runtime rather than the
// math.Sqrt2 constant so it stays bit-identical to the per-entry expressions
// it replaced, which the golden-value tests compare exactly. It is kept as a
// float64 so negated entries are built as complex(-invSqrt2, 0) with a +0
// imaginary part — negating a complex128 would flip it to -0, which the CLI
// output tests observe through Sprintf.
var invSqrt2 = 1 / math.Sqrt(2)

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
		{complex(invSqrt2, 0), complex(invSqrt2, 0)},
		{complex(invSqrt2, 0), complex(-invSqrt2, 0)},
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

// NewToffoli creates a new Toffoli (CCNOT) gate (8x8, 3-qubit).
// The two most-significant basis bits control a flip of the least-significant
// one, so applied as ApplyGate(gate, c1, c2, target) the first two targets are
// the controls: the gate follows the same targets[0]-is-most-significant
// convention as CNOT.
func NewToffoli() *MatrixGate {
	return mustGate("Toffoli", [][]complex128{
		{1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 0, 0, 0, 1, 0},
	})
}

// NewRx creates a rotation of theta radians about the Bloch sphere's X axis:
// Rx(θ) = cos(θ/2)·I − i·sin(θ/2)·X. Rx(π) is X up to a global phase of −i.
//
// The angle is part of the gate's name ("Rx(0.5)"), so rotations by different
// angles stay distinguishable — including in a Registry, which keys on name.
// theta itself is not validated: a non-finite angle yields a non-finite
// matrix, since gate matrices are checked for shape, not for finiteness.
func NewRx(theta float64) *MatrixGate {
	sin, cos := math.Sincos(theta / 2)
	return mustGate(fmt.Sprintf("Rx(%g)", theta), [][]complex128{
		{complex(cos, 0), complex(0, -sin)},
		{complex(0, -sin), complex(cos, 0)},
	})
}

// NewRy creates a rotation of theta radians about the Bloch sphere's Y axis:
// Ry(θ) = cos(θ/2)·I − i·sin(θ/2)·Y. Ry(π) is Y up to a global phase of −i.
// theta is not validated; see NewRx.
func NewRy(theta float64) *MatrixGate {
	sin, cos := math.Sincos(theta / 2)
	return mustGate(fmt.Sprintf("Ry(%g)", theta), [][]complex128{
		{complex(cos, 0), complex(-sin, 0)},
		{complex(sin, 0), complex(cos, 0)},
	})
}

// NewRz creates a rotation of theta radians about the Bloch sphere's Z axis:
// Rz(θ) = diag(e^{-iθ/2}, e^{iθ/2}). Rz(π) is Z up to a global phase of −i,
// and Rz(θ) is NewPhase(θ) up to a global phase of e^{-iθ/2}: the two differ
// only in how they split the phase between the |0⟩ and |1⟩ branches.
// theta is not validated; see NewRx.
func NewRz(theta float64) *MatrixGate {
	sin, cos := math.Sincos(theta / 2)
	return mustGate(fmt.Sprintf("Rz(%g)", theta), [][]complex128{
		{complex(cos, -sin), 0},
		{0, complex(cos, sin)},
	})
}

// NewPhase creates a phase-shift gate that leaves |0⟩ alone and sends
// |1⟩ to e^{iφ}|1⟩. It generalizes the fixed phase gates: NewPhase(π) is Z,
// NewPhase(π/2) is S, and NewPhase(π/4) is T.
// phi is not validated; see NewRx.
func NewPhase(phi float64) *MatrixGate {
	sin, cos := math.Sincos(phi)
	return mustGate(fmt.Sprintf("Phase(%g)", phi), [][]complex128{
		{1, 0},
		{0, complex(cos, sin)},
	})
}
