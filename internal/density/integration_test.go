package density_test

import (
	"errors"
	"math"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/density"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// bellCircuit builds the two-qubit Bell circuit used throughout: H on
// qubit 0, then CNOT with qubit 0 controlling qubit 1.
func bellCircuit(t *testing.T) *circuit.Circuit {
	t.Helper()
	c, err := circuit.New(2)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("AddGate(H): %v", err)
	}
	if err := c.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("AddGate(CNOT): %v", err)
	}
	return c
}

// bellPair returns a density matrix holding a Bell pair, built by
// executing the circuit — the interchangeability being tested.
func bellPair(t *testing.T) *density.Matrix {
	t.Helper()
	m, err := density.New(2)
	if err != nil {
		t.Fatalf("density.New: %v", err)
	}
	if err := bellCircuit(t).Execute(m); err != nil {
		t.Fatalf("Execute on density: %v", err)
	}
	return m
}

func TestCircuitExecutesInterchangeably(t *testing.T) {
	c := bellCircuit(t)

	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(dense); err != nil {
		t.Fatalf("Execute on dense: %v", err)
	}
	m := bellPair(t)

	for basis := 0; basis < 4; basis++ {
		if diff := math.Abs(m.Probability(basis) - dense.Probability(basis)); diff > 1e-12 {
			t.Errorf("P(%d): density %g vs dense %g", basis,
				m.Probability(basis), dense.Probability(basis))
		}
	}
}

func TestExpectationOnPureDensityState(t *testing.T) {
	m := bellPair(t)
	for _, tc := range []struct {
		name string
		axes []quantum.PauliAxis
		want float64
	}{
		{"ZZ", []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ}, 1},
		{"XX", []quantum.PauliAxis{quantum.PauliX, quantum.PauliX}, 1},
	} {
		got, err := quantum.Expectation(m, tc.axes)
		if err != nil {
			t.Fatalf("Expectation(%s): %v", tc.name, err)
		}
		if math.Abs(got-tc.want) > 1e-10 {
			t.Errorf("⟨%s⟩ = %g, want %g", tc.name, got, tc.want)
		}
	}
}

func TestHelpersRejectMixedDensityState(t *testing.T) {
	m := bellPair(t)
	if err := m.ApplyDepolarizing(0, 0.5); err != nil {
		t.Fatalf("ApplyDepolarizing: %v", err)
	}

	var unnormalized *quantum.UnnormalizedStateError

	if _, err := quantum.Sample(m, 10, nil); !errors.As(err, &unnormalized) {
		t.Errorf("Sample on mixed state: err = %v, want *UnnormalizedStateError", err)
	}
	if _, err := quantum.Expectation(m, []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ}); !errors.As(err, &unnormalized) {
		t.Errorf("Expectation on mixed state: err = %v, want *UnnormalizedStateError", err)
	}
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if _, err := quantum.Fidelity(dense, m); !errors.As(err, &unnormalized) {
		t.Errorf("Fidelity with mixed state: err = %v, want *UnnormalizedStateError", err)
	}
}

func TestFidelityDenseVsPureDensityIsOne(t *testing.T) {
	c := bellCircuit(t)
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(dense); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m := bellPair(t)

	got, err := quantum.Fidelity(dense, m)
	if err != nil {
		t.Fatalf("Fidelity: %v", err)
	}
	if math.Abs(got-1) > 1e-10 {
		t.Errorf("Fidelity(dense Bell, density Bell) = %g, want 1", got)
	}
}

func TestQFTRefusesDensityBackend(t *testing.T) {
	m, err := density.New(2)
	if err != nil {
		t.Fatalf("density.New: %v", err)
	}
	var unsupported *quantum.UnsupportedOperationError
	if err := quantum.QFT(m); !errors.As(err, &unsupported) {
		t.Errorf("QFT on density backend: err = %v, want *UnsupportedOperationError", err)
	}
}

func TestSampleAgreesWithDenseOnPureState(t *testing.T) {
	c := bellCircuit(t)
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(dense); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m := bellPair(t)

	// Identical fixed draws must produce identical histograms, because a
	// pure density state reconstructs the same probabilities.
	draws := []float64{0.1, 0.6, 0.4, 0.9, 0.2, 0.7}
	denseHist, err := quantum.Sample(dense, len(draws), &fixedSource{draws: draws})
	if err != nil {
		t.Fatalf("Sample(dense): %v", err)
	}
	densityHist, err := quantum.Sample(m, len(draws), &fixedSource{draws: draws})
	if err != nil {
		t.Fatalf("Sample(density): %v", err)
	}
	if len(denseHist) != len(densityHist) {
		t.Fatalf("histograms differ: dense %v, density %v", denseHist, densityHist)
	}
	for k, v := range denseHist {
		if densityHist[k] != v {
			t.Errorf("histogram[%q]: dense %d, density %d", k, v, densityHist[k])
		}
	}
}

// fixedSource replays a fixed sequence of draws.
type fixedSource struct {
	draws []float64
	next  int
}

func (s *fixedSource) Float64() float64 {
	v := s.draws[s.next]
	s.next++
	return v
}
