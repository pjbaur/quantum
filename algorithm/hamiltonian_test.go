package algorithm_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// bellState prepares (|00>+|11>)/sqrt2 on a fresh 2-qubit state.
func bellState(t *testing.T) *state.State {
	t.Helper()
	c, err := circuit.New(2)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("AddGate H: %v", err)
	}
	if err := c.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("AddGate CNOT: %v", err)
	}
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return s
}

func TestHamiltonianEnergyIdentityOnly(t *testing.T) {
	h := algorithm.NewHamiltonian().AddTerm(-1.25)
	s, _ := state.New(2)
	got, err := h.Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	if got != -1.25 {
		t.Fatalf("Energy = %v, want -1.25", got)
	}
}

func TestHamiltonianEnergyKnownValues(t *testing.T) {
	// Bell state: <ZZ>=1, <XX>=1, <YY>=-1.
	s := bellState(t)

	h := algorithm.NewHamiltonian().
		AddTerm(2.0, quantum.PauliZ, quantum.PauliZ).
		AddTerm(1.5, quantum.PauliX, quantum.PauliX).
		AddTerm(-0.5, quantum.PauliY, quantum.PauliY)
	got, err := h.Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	want := 2.0*1 + 1.5*1 + (-0.5)*(-1) // 4.0
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("Energy = %v, want %v", got, want)
	}
}

func TestHamiltonianEnergySingleQubit(t *testing.T) {
	// |+> on qubit 0 of a 2-qubit register: <XI> = 1.
	c, err := circuit.New(2)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("AddGate H: %v", err)
	}
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	h := algorithm.NewHamiltonian().AddTerm(3.0, quantum.PauliX, quantum.PauliI)
	got, err := h.Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	if math.Abs(got-3.0) > 1e-12 {
		t.Fatalf("Energy = %v, want 3.0", got)
	}
}

func TestHamiltonianEnergyPropagatesExpectationErrors(t *testing.T) {
	h := algorithm.NewHamiltonian().AddTerm(1.0, quantum.PauliZ)
	if _, err := h.Energy(nil); err == nil {
		t.Fatal("Energy(nil) succeeded, want error")
	}
	// Axis length mismatch: 3 axes on a 2-qubit state.
	s, _ := state.New(2)
	h2 := algorithm.NewHamiltonian().AddTerm(1.0, quantum.PauliZ, quantum.PauliZ, quantum.PauliZ)
	if _, err := h2.Energy(s); err == nil {
		t.Fatal("Energy with 3 axes on 2 qubits succeeded, want error")
	}
}
