package quantum_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

func TestExpectationDenseSparseAgree(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	sparse, err := sparsestate.New(2)
	if err != nil {
		t.Fatalf("sparsestate.New: %v", err)
	}
	bellState(t, dense)
	bellState(t, sparse)

	axes := []quantum.PauliAxis{quantum.PauliI, quantum.PauliX, quantum.PauliY, quantum.PauliZ}
	for _, a0 := range axes {
		for _, a1 := range axes {
			d, err := quantum.Expectation(dense, []quantum.PauliAxis{a0, a1})
			if err != nil {
				t.Fatalf("dense Expectation(%d,%d): %v", a0, a1, err)
			}
			s, err := quantum.Expectation(sparse, []quantum.PauliAxis{a0, a1})
			if err != nil {
				t.Fatalf("sparse Expectation(%d,%d): %v", a0, a1, err)
			}
			if math.Abs(d-s) > 1e-10 {
				t.Errorf("axes (%d,%d): dense %v vs sparse %v", a0, a1, d, s)
			}
		}
	}
}

func TestSampleExpectationBellXXExactlyOne(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	s := bellState(t, dense)

	// H on both qubits maps the Bell state to itself, so every outcome has
	// even X-parity: the estimate is exactly +1 whatever the draws.
	got, err := quantum.SampleExpectation(s, []quantum.PauliAxis{quantum.PauliX, quantum.PauliX},
		2, &sequenceRand{draws: []float64{0.25, 0.75}})
	if err != nil {
		t.Fatalf("SampleExpectation: %v", err)
	}
	if got != 1.0 {
		t.Errorf("SampleExpectation <XX> = %v, want exactly 1", got)
	}
}

func TestSampleExpectationBellZXParities(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	s := bellState(t, dense)

	// Measuring Z on q0 and X on q1 rotates only q1: outcomes become the
	// four states (|0+> + |1->)/sqrt(2) with equal probability, whose
	// parities are +1, -1, -1, +1 in basis order. One draw per bucket
	// cancels to exactly zero.
	got, err := quantum.SampleExpectation(s, []quantum.PauliAxis{quantum.PauliZ, quantum.PauliX},
		4, &sequenceRand{draws: []float64{0.1, 0.35, 0.6, 0.85}})
	if err != nil {
		t.Fatalf("SampleExpectation: %v", err)
	}
	if got != 0.0 {
		t.Errorf("SampleExpectation <ZX> = %v, want exactly 0", got)
	}
}

func TestSampleExpectationYAxis(t *testing.T) {
	plusY, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := plusY.SetAmplitudes([]complex128{complex(0.7071067811865476, 0), complex(0, 0.7071067811865476)}); err != nil {
		t.Fatalf("SetAmplitudes: %v", err)
	}

	got, err := quantum.SampleExpectation(plusY, []quantum.PauliAxis{quantum.PauliY}, 1,
		&sequenceRand{draws: []float64{0.3}})
	if err != nil {
		t.Fatalf("SampleExpectation: %v", err)
	}
	if got != 1.0 {
		t.Errorf("SampleExpectation <Y> on |+_y> = %v, want exactly 1", got)
	}

	minusY, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := minusY.SetAmplitudes([]complex128{complex(0.7071067811865476, 0), complex(0, -0.7071067811865476)}); err != nil {
		t.Fatalf("SetAmplitudes: %v", err)
	}

	got, err = quantum.SampleExpectation(minusY, []quantum.PauliAxis{quantum.PauliY}, 1,
		&sequenceRand{draws: []float64{0.3}})
	if err != nil {
		t.Fatalf("SampleExpectation: %v", err)
	}
	if got != -1.0 {
		t.Errorf("SampleExpectation <Y> on |-_y> = %v, want exactly -1", got)
	}
}

func TestSampleExpectationAgreesWithExact(t *testing.T) {
	dense, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := dense.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplyGate H: %v", err)
	}

	// <Z> on |+> is exactly 0; with 10000 seeded shots the estimator's
	// standard deviation is 0.01, so 0.05 is a 5-sigma window.
	got, err := quantum.SampleExpectation(dense, []quantum.PauliAxis{quantum.PauliZ},
		10000, rand.New(rand.NewSource(7)))
	if err != nil {
		t.Fatalf("SampleExpectation: %v", err)
	}
	if math.Abs(got) > 0.05 {
		t.Errorf("SampleExpectation <Z> on |+> = %v, want within 0.05 of 0", got)
	}
}

func TestSampleExpectationIsNonDestructive(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	s := bellState(t, dense)
	before0, before3 := s.Amplitude(0), s.Amplitude(3)

	if _, err := quantum.SampleExpectation(s, []quantum.PauliAxis{quantum.PauliX, quantum.PauliX},
		10, &sequenceRand{draws: repeatedDraws(0.25, 10)}); err != nil {
		t.Fatalf("SampleExpectation: %v", err)
	}

	if got := s.Amplitude(0); got != before0 {
		t.Errorf("amplitude 0 changed: %v -> %v", before0, got)
	}
	if got := s.Amplitude(3); got != before3 {
		t.Errorf("amplitude 3 changed: %v -> %v", before3, got)
	}
}

func TestSampleExpectationPropagatesSampleErrors(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	s := bellState(t, dense)

	_, err = quantum.SampleExpectation(s, []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ},
		0, &sequenceRand{draws: []float64{0.5}})
	if _, ok := err.(*quantum.InvalidShotCountError); !ok {
		t.Errorf("shots=0 error = %T (%v), want *InvalidShotCountError", err, err)
	}

	_, err = quantum.SampleExpectation(s, []quantum.PauliAxis{quantum.PauliZ}, 1,
		&sequenceRand{draws: []float64{0.5}})
	if _, ok := err.(*quantum.IncompatibleQubitCountError); !ok {
		t.Errorf("axis-length error = %T (%v), want *IncompatibleQubitCountError", err, err)
	}
}
