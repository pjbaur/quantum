package algorithm

import (
	"math"
	"math/rand"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// bellState returns |Phi+> = (|00> + |11>)/sqrt(2) on qubits 0 and 1.
func bellState(t *testing.T) quantum.QuantumState {
	t.Helper()
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New(2): %v", err)
	}
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("Hadamard: %v", err)
	}
	if err := s.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("CNOT: %v", err)
	}
	return s
}

func TestChshCorrelationMatchesCosineOnBellState(t *testing.T) {
	s := bellState(t)
	cases := []struct{ thetaA, thetaB float64 }{
		{0, 0},
		{0, math.Pi / 4},
		{math.Pi / 2, math.Pi / 4},
		{math.Pi / 2, -math.Pi / 4},
		{math.Pi / 3, math.Pi / 6},
	}
	for _, c := range cases {
		got, err := ChshCorrelation(s, c.thetaA, c.thetaB)
		if err != nil {
			t.Fatalf("ChshCorrelation(%g, %g): %v", c.thetaA, c.thetaB, err)
		}
		want := math.Cos(c.thetaA - c.thetaB)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("ChshCorrelation(%g, %g) = %g, want %g", c.thetaA, c.thetaB, got, want)
		}
	}
}

func TestChshSExactViolatesBoundOnBellState(t *testing.T) {
	s := bellState(t)
	got, err := ChshSExact(s)
	if err != nil {
		t.Fatalf("ChshSExact: %v", err)
	}
	if math.Abs(got-2*math.Sqrt2) > 1e-9 {
		t.Errorf("ChshSExact on Bell state = %g, want 2*sqrt(2) = %g", got, 2*math.Sqrt2)
	}
}

func TestChshSExactStaysUnderBoundOnProductState(t *testing.T) {
	// |00> is a product state: E(thetaA, thetaB) = cos(thetaA)cos(thetaB),
	// so S = sqrt(2) < 2 — no violation without entanglement.
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New(2): %v", err)
	}
	got, err := ChshSExact(s)
	if err != nil {
		t.Fatalf("ChshSExact: %v", err)
	}
	if math.Abs(got-math.Sqrt2) > 1e-9 {
		t.Errorf("ChshSExact on |00> = %g, want sqrt(2)", got)
	}
	if got >= 2 {
		t.Errorf("ChshSExact on |00> = %g, must stay under the classical bound 2", got)
	}
}

func TestChshSSampledSeededLandsNearTsirelson(t *testing.T) {
	s := bellState(t)
	rng := rand.New(rand.NewSource(42))
	got, err := ChshSSampled(s, 400, rng)
	if err != nil {
		t.Fatalf("ChshSSampled: %v", err)
	}
	if got < 2 || got > 2*math.Sqrt2+0.5 {
		t.Errorf("ChshSSampled(400 shots, seed 42) = %g, want within [2, 2*sqrt(2)+0.5]", got)
	}
}

func TestChshCorrelationLeavesOriginalStateUntouched(t *testing.T) {
	s := bellState(t)
	before := []complex128{s.Amplitude(0), s.Amplitude(1), s.Amplitude(2), s.Amplitude(3)}
	if _, err := ChshCorrelation(s, math.Pi/2, math.Pi/4); err != nil {
		t.Fatalf("ChshCorrelation: %v", err)
	}
	for i, a := range before {
		if s.Amplitude(i) != a {
			t.Fatalf("amplitude %d changed: got %v, want %v", i, s.Amplitude(i), a)
		}
	}
}

func TestChshCorrelationRejectsBadStates(t *testing.T) {
	if _, err := ChshCorrelation(nil, 0, 0); err == nil {
		t.Error("ChshCorrelation(nil) must error")
	}
	var typedNil *state.State
	if _, err := ChshCorrelation(typedNil, 0, 0); err == nil {
		t.Error("ChshCorrelation(typed nil) must error, not panic")
	}
	oneQubit, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New(1): %v", err)
	}
	if _, err := ChshCorrelation(oneQubit, 0, 0); err == nil {
		t.Error("ChshCorrelation on 1-qubit state must error")
	}
	threeQubit, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New(3): %v", err)
	}
	if _, err := ChshCorrelation(threeQubit, 0, 0); err == nil {
		t.Error("ChshCorrelation on 3-qubit state must error: CHSH needs exactly 2")
	}
}
