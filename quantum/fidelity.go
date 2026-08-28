package quantum

import (
	"errors"
	"math/cmplx"
)

// Fidelity returns F(|ψ⟩, |φ⟩) = |⟨ψ|φ⟩|² for two pure states, a real
// number in [0, 1] that is 1 exactly when the states differ only by a
// global phase and 0 when they are orthogonal. It is symmetric in its
// arguments and ignores relative phase, which is what makes it the right
// closeness measure for verifying state preparation (teleportation output
// vs input, cross-backend agreement) where the phase carries no meaning.
//
// The overlap is computed directly from both amplitude vectors in O(2ⁿ)
// with no matrix construction. Neither state is modified.
//
// Returns an error if either state is nil or a typed nil ("first"/"second"
// names the argument), if the qubit counts differ
// (IncompatibleQubitCountError), or if either state's probabilities do not
// sum to 1 within the backends' 1e-10 tolerance (UnnormalizedStateError; a
// NaN sum fails the same check).
func Fidelity(a, b QuantumState) (float64, error) {
	if isNilQuantumState(a) {
		return 0, errors.New("first state must not be nil")
	}
	if isNilQuantumState(b) {
		return 0, errors.New("second state must not be nil")
	}

	numQubits := a.NumQubits()
	if other := b.NumQubits(); other != numQubits {
		return 0, &IncompatibleQubitCountError{
			Expected: numQubits,
			Actual:   other,
		}
	}

	size := 1 << numQubits
	sumA, sumB := 0.0, 0.0
	overlap := complex(0, 0)
	for i := 0; i < size; i++ {
		av, bv := a.Amplitude(i), b.Amplitude(i)
		sumA += Probability(av)
		sumB += Probability(bv)
		overlap += cmplx.Conj(av) * bv
	}

	// Negated comparisons so a NaN probability sum is rejected too.
	if !IsNormalizedSum(sumA) {
		return 0, &UnnormalizedStateError{Sum: sumA}
	}
	if !IsNormalizedSum(sumB) {
		return 0, &UnnormalizedStateError{Sum: sumB}
	}

	// |z|² written out: the direct re²+im² avoids cmplx.Abs's square root
	// for the same reason quantum.Probability does.
	return real(overlap)*real(overlap) + imag(overlap)*imag(overlap), nil
}
