package algorithm

import (
	"github.com/pjbaur/quantum/quantum"
)

// HamiltonianTerm is one weighted Pauli string.
type HamiltonianTerm struct {
	Coeff float64
	Axes  []quantum.PauliAxis
}

// Hamiltonian is a sum of weighted Pauli strings:
//
//	H = sum_i c_i * P_i
//
// Terms are stored in insertion order so Energy's floating-point summation
// is deterministic for a given construction sequence.
type Hamiltonian struct {
	terms []HamiltonianTerm
}

// NewHamiltonian returns an empty Hamiltonian (the zero operator).
func NewHamiltonian() *Hamiltonian {
	return &Hamiltonian{}
}

// AddTerm appends coeff * (axes as a Pauli string) and returns h for
// chaining. Empty axes denote the identity, contributing coeff directly.
// Coefficients are not validated: NaN and Inf flow into Energy results,
// where they surface as non-finite energies the caller can detect.
func (h *Hamiltonian) AddTerm(coeff float64, axes ...quantum.PauliAxis) *Hamiltonian {
	h.terms = append(h.terms, HamiltonianTerm{Coeff: coeff, Axes: axes})
	return h
}

// Energy returns the exact expectation value <psi|H|psi> computed term by
// term via quantum.Expectation. Identity terms (no axes) add their
// coefficient directly.
func (h *Hamiltonian) Energy(s quantum.QuantumState) (float64, error) {
	total := 0.0
	for _, term := range h.terms {
		if len(term.Axes) == 0 {
			total += term.Coeff
			continue
		}
		value, err := quantum.Expectation(s, term.Axes)
		if err != nil {
			return 0, err
		}
		total += term.Coeff * value
	}
	return total, nil
}
