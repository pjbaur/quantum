package quantum

import (
	"errors"
	"math"
	"math/cmplx"
)

// PauliAxis identifies one factor of a Pauli string: the single-qubit
// operator I, X, Y, or Z acting on one qubit of a register.
type PauliAxis uint8

// The four Pauli operators, ordered so validity is a <= PauliZ.
const (
	PauliI PauliAxis = iota
	PauliX
	PauliY
	PauliZ
)

// expectationInvSqrt2 is 1/√2 for the basis-rotation matrices below.
var expectationInvSqrt2 = 1 / math.Sqrt(2)

// rotationGate is a fixed single-qubit matrix satisfying Gate. The quantum
// package cannot import gates (gates imports quantum), so the two rotations
// basis-change measurement needs are defined here. The matrices are
// read-only by every ApplyGate path.
type rotationGate struct {
	name   string
	matrix [][]complex128
}

func (g rotationGate) Name() string           { return g.name }
func (g rotationGate) Matrix() [][]complex128 { return g.matrix }

// hadamardRotation maps the X eigenbasis to the Z eigenbasis.
func hadamardRotation() Gate {
	v := complex(expectationInvSqrt2, 0)
	return rotationGate{name: "Hadamard", matrix: [][]complex128{{v, v}, {v, -v}}}
}

// sdagRotation is S† = diag(1, -i). Applied before the Hadamard it maps the
// Y eigenbasis to the Z eigenbasis.
func sdagRotation() Gate {
	return rotationGate{name: "Sdg", matrix: [][]complex128{{1, 0}, {0, -1i}}}
}

// Expectation returns the exact expectation value ⟨ψ|P|ψ⟩ of the Pauli
// string P = axes[0] ⊗ axes[1] ⊗ ... on state s, where axes[k] acts on
// qubit k (qubit 0 is the least significant bit of the basis index).
//
// The value is computed from the full amplitude vector as
// Σᵢ conj(aᵢ) · phase(i) · a_{i⊕mask}, where mask holds the qubits with an
// X or Y factor and phase(i) is the product of (-1) per set Z-qubit bit and
// ±i per Y-qubit bit. For a Hermitian Pauli string the result is real; the
// imaginary residue of the float sum is discarded, so callers get a clean
// float64. Cost is O(2ⁿ) time with no matrix construction.
//
// Returns an error if s is nil or a typed nil, if len(axes) does not equal
// the qubit count (IncompatibleQubitCountError), if an axis is not one of
// the four Pauli operators (InvalidPauliAxisError), or if the state's
// probabilities do not sum to 1 within the backends' 1e-10 tolerance
// (UnnormalizedStateError; a NaN sum fails the same check).
func Expectation(s QuantumState, axes []PauliAxis) (float64, error) {
	if isNilQuantumState(s) {
		return 0, errors.New("state must not be nil")
	}
	numQubits, err := validatePauliAxes(s, axes)
	if err != nil {
		return 0, err
	}

	// Qubits with an X or Y factor are flipped; I and Z keep the bit.
	mask := 0
	for k, axis := range axes {
		if axis == PauliX || axis == PauliY {
			mask |= 1 << k
		}
	}

	size := 1 << numQubits
	sum := 0.0
	expectation := complex(0, 0)
	for i := 0; i < size; i++ {
		amp := s.Amplitude(i)
		sum += Probability(amp)

		// P maps |src⟩ to phase(src)|i⟩ where src ⊕ mask = i, so the
		// phase comes from the source index's bits, not the
		// destination's. Getting this backwards flips the sign of every
		// string with an odd number of Y factors.
		src := i ^ mask
		phase := complex(1, 0)
		for k, axis := range axes {
			bit := src&(1<<k) != 0
			switch axis {
			case PauliZ:
				if bit {
					phase = -phase
				}
			case PauliY:
				if bit {
					phase *= -1i
				} else {
					phase *= 1i
				}
			}
		}
		expectation += cmplx.Conj(amp) * phase * s.Amplitude(src)
	}

	// Negated comparison so a NaN probability sum is rejected too.
	if !(math.Abs(sum-1.0) <= sampleNormalizationTolerance) {
		return 0, &UnnormalizedStateError{Sum: sum}
	}
	return real(expectation), nil
}

// SampleExpectation estimates ⟨ψ|P|ψ⟩ for the Pauli string described by
// axes from `shots` measurement outcomes, the way a real device would:
// each qubit with an X or Y factor is rotated into the Z basis (H for X,
// S† then H for Y), the rotated state is sampled non-destructively, and
// the outcomes' parity over the non-identity qubits is averaged as ±1.
//
// The state itself is never modified: rotations run on a Clone. rng feeds
// Sample, so a seeded source makes the estimate reproducible; a nil rng
// uses the global math/rand source. Error contract is Expectation's
// (nil state, axis length, invalid axis) plus Sample's (shot count,
// normalization).
//
// The estimate converges to the exact value at rate O(1/√shots) and
// matches Expectation exactly for Pauli strings whose rotated outcomes
// have deterministic parity.
func SampleExpectation(s QuantumState, axes []PauliAxis, shots int, rng RandomSource) (float64, error) {
	if isNilQuantumState(s) {
		return 0, errors.New("state must not be nil")
	}
	numQubits, err := validatePauliAxes(s, axes)
	if err != nil {
		return 0, err
	}

	rotated := s.Clone()
	for k, axis := range axes {
		switch axis {
		case PauliX:
			if err := rotated.ApplyGate(hadamardRotation(), k); err != nil {
				return 0, err
			}
		case PauliY:
			if err := rotated.ApplyGate(sdagRotation(), k); err != nil {
				return 0, err
			}
			if err := rotated.ApplyGate(hadamardRotation(), k); err != nil {
				return 0, err
			}
		}
	}

	counts, err := Sample(rotated, shots, rng)
	if err != nil {
		return 0, err
	}

	// Key layout is %0*b: character 0 is the highest-index qubit, so
	// qubit k lives at character numQubits-1-k.
	total := 0
	for key, count := range counts {
		parity := 1
		for k, axis := range axes {
			if axis != PauliI && key[numQubits-1-k] == '1' {
				parity = -parity
			}
		}
		total += parity * count
	}
	return float64(total) / float64(shots), nil
}

// validatePauliAxes checks that axes names exactly one operator per qubit
// of s and returns the qubit count.
func validatePauliAxes(s QuantumState, axes []PauliAxis) (int, error) {
	numQubits := s.NumQubits()
	if len(axes) != numQubits {
		return 0, &IncompatibleQubitCountError{
			Expected: numQubits,
			Actual:   len(axes),
		}
	}
	for k, axis := range axes {
		if axis > PauliZ {
			return 0, &InvalidPauliAxisError{Qubit: k, Axis: axis}
		}
	}
	return numQubits, nil
}
