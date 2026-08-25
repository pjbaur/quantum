package qubit

import (
	"math"
	"math/rand"

	"github.com/pjbaur/quantum/quantum"
)

// Qubit implements the quantum.Qubit interface
type Qubit struct {
	alpha      complex128
	beta       complex128
	randSource quantum.RandomSource
}

// SetRandSource sets the randomness source used by Measure, enabling
// reproducible measurements from a seeded generator. A nil source
// restores the default (the global math/rand source).
func (q *Qubit) SetRandSource(src quantum.RandomSource) {
	q.randSource = src
}

func (q *Qubit) randFloat64() float64 {
	if q.randSource != nil {
		return q.randSource.Float64()
	}
	return rand.Float64()
}

// New creates a new qubit in the |0⟩ state
func New() *Qubit {
	return &Qubit{
		alpha: 1.0,
		beta:  0.0,
	}
}

// NewWithValues creates a new qubit with specific amplitudes
// Returns error if amplitudes would not result in a normalized state
func NewWithValues(alpha, beta complex128) (*Qubit, error) {
	q := &Qubit{}
	if err := q.Set(alpha, beta); err != nil {
		return nil, err
	}
	return q, nil
}

// Alpha returns the amplitude of the |0⟩ state
func (q *Qubit) Alpha() complex128 {
	return q.alpha
}

// Beta returns the amplitude of the |1⟩ state
func (q *Qubit) Beta() complex128 {
	return q.beta
}

// Set updates the amplitudes of the qubit
func (q *Qubit) Set(alpha, beta complex128) error {
	// Calculate probability sum to check normalization
	probSum := quantum.Probability(alpha) + quantum.Probability(beta)

	// Allow a small floating-point error margin
	if math.Abs(probSum-1.0) > 1e-6 {
		currentSum := quantum.Probability(q.alpha) + quantum.Probability(q.beta)
		return &quantum.NormalizationError{
			AttemptedSum: probSum,
			CurrentSum:   currentSum,
		}
	}

	q.alpha = alpha
	q.beta = beta
	return nil
}

// Probability0 returns the probability of measuring |0⟩
func (q *Qubit) Probability0() float64 {
	return quantum.Probability(q.alpha)
}

// Probability1 returns the probability of measuring |1⟩
func (q *Qubit) Probability1() float64 {
	return quantum.Probability(q.beta)
}

// Measure collapses the qubit to either |0⟩ or |1⟩
func (q *Qubit) Measure() int {
	prob0 := q.Probability0()
	if q.randFloat64() < prob0 {
		// Collapse to |0⟩
		q.alpha = 1.0
		q.beta = 0.0
		return 0
	} else {
		// Collapse to |1⟩
		q.alpha = 0.0
		q.beta = 1.0
		return 1
	}
}

// Clone creates a copy of this qubit. The clone shares the original's
// randomness source (if any), so seeded pipelines stay deterministic.
func (q *Qubit) Clone() quantum.Qubit {
	return &Qubit{
		alpha:      q.alpha,
		beta:       q.beta,
		randSource: q.randSource,
	}
}

// IsNormalized checks if the qubit is properly normalized
func (q *Qubit) IsNormalized() bool {
	sum := quantum.Probability(q.alpha) + quantum.Probability(q.beta)
	return math.Abs(sum-1.0) <= 1e-10
}
