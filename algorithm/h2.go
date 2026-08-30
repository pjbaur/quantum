package algorithm

import (
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)

// H2 coefficients for the two-qubit reduced Hamiltonian at the equilibrium
// bond length R = 0.735 Angstrom, in Hartree. Source: O'Malley et al.,
// "Scalable Quantum Simulation of Molecular Energies", PRA 93, 052337
// (2016), XZX (parity) basis reduction. g0 includes the nuclear repulsion
// and electronic constants; the g4 term couples the two parity sectors.
//
// The g4 coupling is carried by Y0Y1, the convention that passed the
// literature check in TestH2GroundEnergyInLiteratureRange: the Jacobi
// verifier in jacobi_test.go diagonalizes these exact terms to a ground
// energy of -1.8573 Ha, within 0.3 of the published -1.857. (An X0X1
// coupling is spectrally identical here — both strings connect the same
// pairs of basis states — but Y0Y1 follows the source's Eq. 4.)
//
// If TestH2GroundEnergyInLiteratureRange fails, the convention (which Pauli
// string carries g4, or the qubit ordering of Z terms) differs from the
// source's basis. Diagnose with the Jacobi verifier, not by tuning numbers.
const (
	h2G0 = -1.052373245772859
	h2G1 = 0.39793742484318045
	h2G2 = -0.39793742484318045
	h2G3 = -0.01128010425623538
	h2G4 = 0.18093119978423156
)

// H2Terms returns the H2 Pauli terms with axes ordered qubit 0 first,
// matching quantum.Expectation's convention.
func H2Terms() []HamiltonianTerm {
	return []HamiltonianTerm{
		{Coeff: h2G0},
		{Coeff: h2G1, Axes: []quantum.PauliAxis{quantum.PauliZ, quantum.PauliI}},
		{Coeff: h2G2, Axes: []quantum.PauliAxis{quantum.PauliI, quantum.PauliZ}},
		{Coeff: h2G3, Axes: []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ}},
		{Coeff: h2G4, Axes: []quantum.PauliAxis{quantum.PauliY, quantum.PauliY}},
	}
}

// H2Hamiltonian returns the two-qubit reduced H2 Hamiltonian at R = 0.735 A.
func H2Hamiltonian() *Hamiltonian {
	h := NewHamiltonian()
	for _, term := range H2Terms() {
		h.AddTerm(term.Coeff, term.Axes...)
	}
	return h
}

// H2Ansatz returns the canonical UCC-inspired H2 ansatz template:
// Ry(theta) on qubit 0, then CNOT with control 0 and target 1.
func H2Ansatz() *parameterized.Template {
	t := parameterized.NewTemplate(2)
	if err := t.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		panic(err) // targets are static; an error here is a programming bug
	}
	if err := t.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	return t
}
