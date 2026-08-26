package visualization

import (
	"errors"
	"fmt"
	"math/cmplx"

	"github.com/pjbaur/quantum/internal/density"
	"github.com/pjbaur/quantum/quantum"
)

// BlochVector represents a single-qubit state as a vector on the Bloch sphere.
type BlochVector struct {
	X float64
	Y float64
	Z float64
}

// BlochVectorFromQubit computes the Bloch vector for a single qubit.
func BlochVectorFromQubit(q quantum.Qubit) BlochVector {
	if q == nil {
		return BlochVector{}
	}
	alpha := q.Alpha()
	beta := q.Beta()
	product := cmplx.Conj(alpha) * beta

	return BlochVector{
		X: 2 * real(product),
		Y: 2 * imag(product),
		Z: quantum.Probability(alpha) - quantum.Probability(beta),
	}
}

// BlochVectorFromState computes the Bloch vector of the qubit at target
// inside the multi-qubit pure state s, tracing out every other qubit.
//
// A qubit entangled with the rest of the register reduces to a mixed state,
// so the vector lies inside the unit sphere rather than on it: each half of
// a Bell pair reduces to the origin.
//
// Returns QubitsOutOfRangeError if target does not name a qubit of s.
func BlochVectorFromState(s quantum.QuantumState, target int) (BlochVector, error) {
	if s == nil {
		return BlochVector{}, errors.New("state must not be nil")
	}

	// Reject a bad target before density.FromState allocates 4ⁿ elements.
	// A non-positive qubit count is left to FromState, which owns that error.
	if numQubits := s.NumQubits(); numQubits > 0 && (target < 0 || target >= numQubits) {
		return BlochVector{}, &quantum.QubitsOutOfRangeError{
			Index:    target,
			MaxIndex: numQubits - 1,
		}
	}

	matrix, err := density.FromState(s)
	if err != nil {
		return BlochVector{}, err
	}

	x, y, z, err := matrix.ReducedBlochVector(target)
	if err != nil {
		return BlochVector{}, err
	}

	return BlochVector{X: x, Y: y, Z: z}, nil
}

// FormatBlochVector renders a simple ASCII representation of a Bloch vector.
func FormatBlochVector(v BlochVector, precision int) string {
	if precision <= 0 {
		precision = 4
	}
	return fmt.Sprintf("x=%.*f y=%.*f z=%.*f", precision, v.X, precision, v.Y, precision, v.Z)
}

// BlochCSV returns comma-separated coordinates for plotting.
func BlochCSV(v BlochVector, precision int) string {
	if precision <= 0 {
		precision = 4
	}
	return fmt.Sprintf("%.*f,%.*f,%.*f", precision, v.X, precision, v.Y, precision, v.Z)
}
