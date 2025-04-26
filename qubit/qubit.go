package qubit

import (
	"math"
	"math/cmplx"
	"math/rand"

	"github.com/pjbaur/quantum"
)

// Qubit implements the quantum.Qubit interface
type Qubit struct {
	alpha complex128
	beta  complex128
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
	probSum := math.Pow(cmplx.Abs(alpha), 2) + math.Pow(cmplx.Abs(beta), 2)

	// Allow a small floating-point error margin
	if math.Abs(probSum-1.0) > 1e-10 {
		return &quantum.NormalizationError{Sum: probSum}
	}

	q.alpha = alpha
	q.beta = beta
	return nil
}

// Probability0 returns the probability of measuring |0⟩
func (q *Qubit) Probability0() float64 {
	return math.Pow(cmplx.Abs(q.alpha), 2)
}

// Probability1 returns the probability of measuring |1⟩
func (q *Qubit) Probability1() float64 {
	return math.Pow(cmplx.Abs(q.beta), 2)
}

// Measure collapses the qubit to either |0⟩ or |1⟩
func (q *Qubit) Measure() int {
	prob0 := q.Probability0()
	if rand.Float64() < prob0 {
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

// Clone creates a copy of this qubit
func (q *Qubit) Clone() quantum.Qubit {
	return &Qubit{
		alpha: q.alpha,
		beta:  q.beta,
	}
}

// IsNormalized checks if the qubit is properly normalized
func (q *Qubit) IsNormalized() bool {
	sum := math.Pow(cmplx.Abs(q.alpha), 2) + math.Pow(cmplx.Abs(q.beta), 2)
	return math.Abs(sum-1.0) <= 1e-10
}
