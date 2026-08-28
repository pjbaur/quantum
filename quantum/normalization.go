package quantum

import (
	"fmt"
	"math"
)

// NormalizationTolerance is the window around 1 within which a state's
// probability sum counts as normalized. Both state backends enforce it on
// every amplitude write, and Sample, Expectation, and Fidelity require it
// of the states they are given. One declaration replaces the five that
// used to cross-reference each other by comment.
const NormalizationTolerance = 1e-10

// IsNormalizedSum reports whether sum is the probability sum of a
// normalized state. A NaN sum is not: it compares false against the
// tolerance, which is what makes NaN-amplitude states fail the checks
// that matter.
func IsNormalizedSum(sum float64) bool {
	return math.Abs(sum-1.0) <= NormalizationTolerance
}

// CheckNormalization returns nil when attemptedSum is a normalized
// probability sum, or the NormalizationError a rejected amplitude write
// reports: the sum the write would have produced alongside the sum the
// state was rolled back to.
func CheckNormalization(attemptedSum, currentSum float64) error {
	if IsNormalizedSum(attemptedSum) {
		return nil
	}
	return &NormalizationError{
		AttemptedSum: attemptedSum,
		CurrentSum:   currentSum,
	}
}

// ValidateAmplitudeVector checks a candidate amplitude vector for a
// size-wide register and returns its probability sum. The length check
// runs first; the amplitudes are then scanned in order, and a non-finite
// amplitude is caught explicitly rather than by the sum — a NaN amplitude
// makes the sum NaN, and NaN fails every comparison. A vector that is
// wrong in more than one way therefore reports the same failure either
// backend reports today.
func ValidateAmplitudeVector(values []complex128, size int) (float64, error) {
	if len(values) != size {
		return 0, fmt.Errorf("values slice length %d does not match state size %d", len(values), size)
	}

	sum := 0.0
	for i, v := range values {
		if !IsFiniteAmplitude(v) {
			return 0, &NonFiniteAmplitudeError{BasisState: i, Value: v}
		}
		sum += Probability(v)
	}
	return sum, nil
}
