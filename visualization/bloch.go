package visualization

import (
	"fmt"
	"math/cmplx"

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
		Z: cmplx.Abs(alpha)*cmplx.Abs(alpha) - cmplx.Abs(beta)*cmplx.Abs(beta),
	}
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
