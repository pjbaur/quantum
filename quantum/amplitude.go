package quantum

import "math/cmplx"

// IsFiniteAmplitude reports whether c is a usable amplitude, meaning neither
// of its components is NaN or infinite.
//
// State backends must check this explicitly, because a normalization check
// cannot: |NaN|² is NaN, and math.Abs(NaN-1) > tolerance is false, so a NaN
// amplitude slips through a comparison against the tolerance and poisons
// every later measurement and gate application.
func IsFiniteAmplitude(c complex128) bool {
	return !cmplx.IsNaN(c) && !cmplx.IsInf(c)
}
