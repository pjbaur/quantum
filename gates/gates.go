// Package gates contains the definition of quantum gates used in the simulator.
package gates

import (
	"math"
	"math/cmplx"
)

// Matrix2x2 represents a 2x2 complex matrix used for single-qubit gates.
type Matrix2x2 [2][2]complex128

// Common quantum gates as constants
var (
	// Identity gate
	I = Matrix2x2{
		{complex(1, 0), complex(0, 0)},
		{complex(0, 0), complex(1, 0)},
	}

	// Pauli-X gate (NOT gate)
	X = Matrix2x2{
		{complex(0, 0), complex(1, 0)},
		{complex(1, 0), complex(0, 0)},
	}

	// Pauli-Y gate
	Y = Matrix2x2{
		{complex(0, 0), complex(0, -1)},
		{complex(0, 1), complex(0, 0)},
	}

	// Pauli-Z gate
	Z = Matrix2x2{
		{complex(1, 0), complex(0, 0)},
		{complex(0, 0), complex(-1, 0)},
	}

	// Hadamard gate
	H = Matrix2x2{
		{complex(1.0/math.Sqrt(2), 0), complex(1.0/math.Sqrt(2), 0)},
		{complex(1.0/math.Sqrt(2), 0), complex(-1.0/math.Sqrt(2), 0)},
	}

	// T gate (π/4 phase shift)
	T = Matrix2x2{
		{complex(1, 0), complex(0, 0)},
		{complex(0, 0), complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))},
	}

	// S gate (π/2 phase shift, equivalent to T²)
	S = Matrix2x2{
		{complex(1, 0), complex(0, 0)},
		{complex(0, 0), complex(0, 1)},
	}

	// CNOT gate
	CNot = func(control int) Matrix2x2 {
		// CNOT = |0⟩⟨0| ⊗ I + |1⟩⟨1| ⊗ X
		var result Matrix2x2
		switch control {
		case 0:
			result = Matrix2x2{
				{complex(1, 0), complex(0, 0)},
				{complex(0, 0), complex(0, 0)},
			}
		case 1:
			result = Matrix2x2{
				{complex(0, 0), complex(0, 0)},
				{complex(0, 0), complex(1, 0)},
			}
		}
		return result.Multiply(X)
	}

	// CZ gate
	CZ = func(control int) Matrix2x2 {
		// CZ = |0⟩⟨0| ⊗ I + |1⟩⟨1| ⊗ Z
		var result Matrix2x2
		switch control {
		case 0:
			result = Matrix2x2{
				{complex(1, 0), complex(0, 0)},
				{complex(0, 0), complex(0, 0)},
			}
		case 1:
			result = Matrix2x2{
				{complex(0, 0), complex(0, 0)},
				{complex(0, 0), complex(-1, 0)},
			}
		}
		return result.Multiply(Z)
	}

	// T-Dagger gate (inverse of T)
	TDagger = T.Conjugate()
)

// Conjugate returns the conjugate of a 2x2 matrix
func (m Matrix2x2) Conjugate() Matrix2x2 {
	var result Matrix2x2
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			result[i][j] = cmplx.Conj(m[i][j])
		}
	}
	return result
}

// Multiply multiplies two 2x2 matrices and returns the result
func (m Matrix2x2) Multiply(other Matrix2x2) Matrix2x2 {
	var result Matrix2x2
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			sum := complex(0, 0)
			for k := 0; k < 2; k++ {
				sum += m[i][k] * other[k][j]
			}
			result[i][j] = sum
		}
	}
	return result
}

// Apply applies the gate matrix to a state vector [α, β]
func (m Matrix2x2) Apply(alpha, beta complex128) (complex128, complex128) {
	newAlpha := m[0][0]*alpha + m[0][1]*beta
	newBeta := m[1][0]*alpha + m[1][1]*beta
	return newAlpha, newBeta
}

// CreateRotationGate creates a rotation gate around the specified axis
// axis: "x", "y", or "z"
// theta: rotation angle in radians
func CreateRotationGate(axis string, theta float64) Matrix2x2 {
	cosTheta2 := complex(math.Cos(theta/2), 0)
	sinTheta2 := complex(math.Sin(theta/2), 0)

	var result Matrix2x2

	switch axis {
	case "x":
		// Rx(θ) = cos(θ/2)I - i·sin(θ/2)X
		result[0][0] = cosTheta2
		result[0][1] = complex(0, -1) * sinTheta2
		result[1][0] = complex(0, -1) * sinTheta2
		result[1][1] = cosTheta2
	case "y":
		// Ry(θ) = cos(θ/2)I - i·sin(θ/2)Y
		result[0][0] = cosTheta2
		result[0][1] = -sinTheta2
		result[1][0] = sinTheta2
		result[1][1] = cosTheta2
	case "z":
		// Rz(θ) = cos(θ/2)I - i·sin(θ/2)Z
		result[0][0] = cmplx.Exp(complex(0, -theta/2))
		result[0][1] = complex(0, 0)
		result[1][0] = complex(0, 0)
		result[1][1] = cmplx.Exp(complex(0, theta/2))
	default:
		// Default to identity if invalid axis
		return I
	}

	return result
}

// CreatePhaseGate creates a phase gate with the specified phase
// phase: rotation angle in radians
func CreatePhaseGate(phase float64) Matrix2x2 {
	return Matrix2x2{
		{complex(1, 0), complex(0, 0)},
		{complex(0, 0), cmplx.Exp(complex(0, phase))},
	}
}
