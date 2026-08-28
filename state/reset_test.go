package state_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

func TestResetReturnsStateToZero(t *testing.T) {
	s, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	for _, q := range []int{0, 1, 2} {
		if err := s.ApplyGate(gates.NewHadamard(), q); err != nil {
			t.Fatalf("ApplyGate H on %d: %v", q, err)
		}
	}
	if p := s.Probability(0); math.Abs(p-0.125) > tolerance {
		t.Fatalf("setup: P(000) = %v, want 0.125", p)
	}

	s.Reset()

	for i := 1; i < 8; i++ {
		if amp := s.Amplitude(i); amp != 0 {
			t.Errorf("amplitude %d after Reset = %v, want 0", i, amp)
		}
	}
	if amp := s.Amplitude(0); amp != 1 {
		t.Errorf("amplitude 0 after Reset = %v, want 1", amp)
	}
	if p := s.Probability(0); math.Abs(p-1) > tolerance {
		t.Errorf("P(000) after Reset = %v, want 1", p)
	}
	if n := s.NumQubits(); n != 3 {
		t.Errorf("NumQubits after Reset = %d, want 3", n)
	}
}

func TestResetKeepsRandSource(t *testing.T) {
	s := plusState(t)
	// Two draws: 0.7 makes the first measurement collapse to 1, and 0.3
	// must still be consumed by the second measurement after Reset —
	// proving the injected source survived the reset.
	s.SetRandSource(&stubRandSource{values: []float64{0.7, 0.3}})

	got, err := s.Measure(0)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if got != 1 {
		t.Fatalf("first Measure = %d, want 1 (draw 0.7 >= P(0) = 0.5)", got)
	}

	s.Reset()
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplyGate H: %v", err)
	}
	got, err = s.Measure(0)
	if err != nil {
		t.Fatalf("Measure after Reset: %v", err)
	}
	if got != 0 {
		t.Errorf("Measure after Reset = %d, want 0 (draw 0.3 < P(0) = 0.5)", got)
	}
}

func TestResetMatchesFreshState(t *testing.T) {
	s, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	// Dirty the state with gates and a measurement so scratch buffers
	// carry stale data.
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplyGate H: %v", err)
	}
	if err := s.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("ApplyGate CNOT: %v", err)
	}
	if _, err := s.Measure(2); err != nil {
		t.Fatalf("Measure: %v", err)
	}
	s.Reset()

	fresh, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	for _, st := range []struct {
		name string
		s    quantum.QuantumState
	}{{"reset", s}, {"fresh", fresh}} {
		if err := st.s.ApplyGate(gates.NewHadamard(), 1); err != nil {
			t.Fatalf("%s: ApplyGate H: %v", st.name, err)
		}
		if err := st.s.ApplyGate(gates.NewCNOT(), 1, 2); err != nil {
			t.Fatalf("%s: ApplyGate CNOT: %v", st.name, err)
		}
	}

	for i := 0; i < 8; i++ {
		if s.Amplitude(i) != fresh.Amplitude(i) {
			t.Errorf("amplitude %d: reset-state %v vs fresh %v",
				i, s.Amplitude(i), fresh.Amplitude(i))
		}
	}
}

func TestStateImplementsResetter(t *testing.T) {
	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	var qs quantum.QuantumState = s
	if _, ok := qs.(quantum.Resetter); !ok {
		t.Error("*state.State does not implement quantum.Resetter")
	}
}
