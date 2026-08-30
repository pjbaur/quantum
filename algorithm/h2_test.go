package algorithm_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)

func TestH2AnsatzShape(t *testing.T) {
	tmpl := algorithm.H2Ansatz()
	if tmpl.NumQubits() != 2 {
		t.Fatalf("NumQubits = %d, want 2", tmpl.NumQubits())
	}
	names := tmpl.ParamNames()
	if len(names) != 1 || names[0] != "theta" {
		t.Fatalf("ParamNames = %v, want [theta]", names)
	}
}

func TestH2GroundEnergyInLiteratureRange(t *testing.T) {
	ground := h2GroundEnergy(t)
	// Total H2 ground energy at R = 0.735 A is about -1.857 Ha. The Jacobi
	// verifier is authoritative; this range check catches a wrong coefficient
	// convention (wrong basis / missing nuclear term) by a wide margin.
	if math.Abs(ground-(-1.857)) > 0.3 {
		t.Fatalf("H2 ground energy = %v, want within 0.3 of -1.857 Ha; check the coefficient convention (O'Malley et al., PRA 93, 052337 (2016))", ground)
	}
}

func TestH2HamiltonianEnergyAtZeroParams(t *testing.T) {
	// theta = 0: ansatz is identity+CNOT, state stays |00>; energy is the
	// diagonal element H[0][0] (the g4 term has zero diagonal). Compare
	// against the Jacobi-built matrix's [0][0] entry.
	tmpl := algorithm.H2Ansatz()
	c, err := tmpl.Bind(parameterized.Params{"theta": 0})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := algorithm.H2Hamiltonian().Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	want := h2Matrix(t)[0][0]
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("Energy(|00>) = %v, want matrix[0][0] = %v", got, want)
	}
}
