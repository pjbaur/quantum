package sparsestate

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)

func TestSparseResetReturnsStateToZero(t *testing.T) {
	s, err := New(3)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Spread across five basis states via SetAmplitudes (5 x 0.2 = 1).
	v := complex(math.Sqrt(0.2), 0)
	if err := s.SetAmplitudes([]complex128{v, v, v, v, v, 0, 0, 0}); err != nil {
		t.Fatalf("SetAmplitudes: %v", err)
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
	if p := s.Probability(0); math.Abs(p-1) > 1e-10 {
		t.Errorf("P(000) after Reset = %v, want 1", p)
	}

	// The map must actually be empty apart from basis 0, not just reading
	// as zero: measure again and confirm the state behaves fresh.
	got, err := s.Measure(1)
	if err != nil {
		t.Fatalf("Measure after Reset: %v", err)
	}
	if got != 0 {
		t.Errorf("Measure on reset state = %d, want 0 deterministically", got)
	}
}

func TestSparseResetMatchesFreshState(t *testing.T) {
	s, err := New(3)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplyGate H: %v", err)
	}
	if _, err := s.Measure(1); err != nil {
		t.Fatalf("Measure: %v", err)
	}
	s.Reset()

	fresh, err := New(3)
	if err != nil {
		t.Fatalf("New: %v", err)
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

func TestSparseStateImplementsResetter(t *testing.T) {
	s, err := New(1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var qs quantum.QuantumState = s
	if _, ok := qs.(quantum.Resetter); !ok {
		t.Error("*sparsestate.State does not implement quantum.Resetter")
	}
}
