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
	// theta = 0: X on qubit 1 seeds q1=1, Ry(0) is identity, and the CNOT
	// (control q0=0) does not fire, so the state stays at basis index 2
	// (q0=0, q1=1). Energy is the diagonal element H[2][2] (the g4 term
	// has zero diagonal). Compare against the Jacobi-built matrix's [2][2]
	// entry.
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
	want := h2Matrix(t)[2][2]
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("Energy(theta=0) = %v, want matrix[2][2] = %v", got, want)
	}
}

func TestH2AnsatzReachesGroundSector(t *testing.T) {
	// Sweep theta over a fine grid and take the minimum ansatz energy:
	// proves the ansatz spans the ground state's sector without relying
	// on any optimizer. 256 steps rather than the suggested 64: the
	// energy's curvature near the minimum is ~0.82, so a 64-point grid
	// can miss the minimum by ~2e-3 Ha, over the 1e-3 tolerance. The
	// tolerance itself is unchanged.
	tmpl := algorithm.H2Ansatz()
	h := algorithm.H2Hamiltonian()
	best := math.Inf(1)
	const steps = 256
	for i := 0; i <= steps; i++ {
		theta := -math.Pi + 2*math.Pi*float64(i)/steps
		c, err := tmpl.Bind(parameterized.Params{"theta": theta})
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
		e, err := h.Energy(s)
		if err != nil {
			t.Fatalf("Energy: %v", err)
		}
		if e < best {
			best = e
		}
	}
	ground := h2GroundEnergy(t)
	if diff := math.Abs(best - ground); diff > 1e-3 {
		t.Fatalf("ansatz minimum over grid = %v, Jacobi ground energy = %v (diff %v, want <= 1e-3)", best, ground, diff)
	}
}
