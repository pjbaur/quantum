package algorithm

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// applyInverseQFTSub runs the gate decomposition on a fresh n-qubit state.
func applyInverseQFTSub(t *testing.T, n int, prepared func(*state.State)) *state.State {
	t.Helper()
	s, err := state.New(n)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	prepared(s)
	c, err := circuit.New(n)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := appendInverseQFTSub(c, n); err != nil {
		t.Fatalf("appendInverseQFTSub: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return s
}

// TestSubregisterIQFTMatchesQuantumInverseQFT is the convention-pinning
// test: the gate decomposition on qubits 0..n-1 must reproduce
// quantum.InverseQFT amplitudes exactly, for several widths and inputs.
func TestSubregisterIQFTMatchesQuantumInverseQFT(t *testing.T) {
	for n := 1; n <= 4; n++ {
		inputs := []func(*state.State){
			// |0...0>
			func(s *state.State) {},
			// uniform |+...+>
			func(s *state.State) {
				for q := 0; q < n; q++ {
					if err := s.ApplyGate(gates.NewHadamard(), q); err != nil {
						t.Fatalf("Hadamard: %v", err)
					}
				}
			},
		}
		// one non-trivial basis state |x> per width
		x := (1 << n) - 3
		if x < 0 {
			x = 1
		}
		inputs = append(inputs, func(s *state.State) {
			for q := 0; q < n; q++ {
				if x&(1<<q) != 0 {
					if err := s.ApplyGate(gates.NewPauliX(), q); err != nil {
						t.Fatalf("PauliX: %v", err)
					}
				}
			}
		})
		for _, prepare := range inputs {
			got := applyInverseQFTSub(t, n, prepare)

			want, err := state.New(n)
			if err != nil {
				t.Fatalf("state.New: %v", err)
			}
			prepare(want)
			if err := quantum.InverseQFT(want); err != nil {
				t.Fatalf("InverseQFT: %v", err)
			}

			for i := 0; i < (1 << n); i++ {
				g := got.Amplitude(i)
				w := want.Amplitude(i)
				if math.Abs(real(g)-real(w)) > 1e-9 || math.Abs(imag(g)-imag(w)) > 1e-9 {
					t.Fatalf("n=%d input %d: amplitude %d = %v, want %v", n, x, i, g, w)
				}
			}
		}
	}
}

// oneQubitEigenstate returns a 1-qubit state with bit flipped to |1> if set.
func oneQubitEigenstate(t *testing.T, flip bool) quantum.QuantumState {
	t.Helper()
	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New(1): %v", err)
	}
	if flip {
		if err := s.ApplyGate(gates.NewPauliX(), 0); err != nil {
			t.Fatalf("PauliX: %v", err)
		}
	}
	return s
}

func TestEstimatePhaseExactEighths(t *testing.T) {
	for k := 0; k < 8; k++ {
		u := gates.NewPhase(2 * math.Pi * float64(k) / 8)
		best, phase, err := EstimatePhase(u, oneQubitEigenstate(t, true), 3)
		if err != nil {
			t.Fatalf("k=%d: EstimatePhase: %v", k, err)
		}
		if best != k {
			t.Errorf("k=%d: bestCount = %d", k, best)
		}
		if math.Abs(phase-float64(k)/8) > 1e-12 {
			t.Errorf("k=%d: phaseTurns = %g", k, phase)
		}
		probs, err := PhaseProbabilities(u, oneQubitEigenstate(t, true), 3)
		if err != nil {
			t.Fatalf("k=%d: PhaseProbabilities: %v", k, err)
		}
		if probs[k] < 1-1e-9 {
			t.Errorf("k=%d: peak probability = %g, want within 1e-9 of 1", k, probs[k])
		}
	}
}

func TestEstimatePhaseNonRepresentableThird(t *testing.T) {
	u := gates.NewPhase(2 * math.Pi / 3)
	best, _, err := EstimatePhase(u, oneQubitEigenstate(t, true), 3)
	if err != nil {
		t.Fatalf("EstimatePhase: %v", err)
	}
	// 3/8 = 0.375 is the closest eighth to 1/3 = 0.333...
	if best != 3 {
		t.Errorf("bestCount = %d, want 3 (closest eighth to 1/3)", best)
	}
}

func TestEstimatePhaseZeroPhaseEigenstate(t *testing.T) {
	u := gates.NewPhase(2 * math.Pi * 3 / 8)
	best, phase, err := EstimatePhase(u, oneQubitEigenstate(t, false), 3)
	if err != nil {
		t.Fatalf("EstimatePhase: %v", err)
	}
	if best != 0 || phase != 0 {
		t.Errorf("best=%d phase=%g, want 0/0 (|0> is a +1 eigenstate of any Phase gate)", best, phase)
	}
}

func TestEstimatePhaseRejectsBadInputs(t *testing.T) {
	u := gates.NewPhase(math.Pi / 4)
	eig := oneQubitEigenstate(t, true)
	if _, _, err := EstimatePhase(nil, eig, 3); err == nil {
		t.Error("nil gate must error")
	}
	if _, _, err := EstimatePhase(u, nil, 3); err == nil {
		t.Error("nil eigenstate must error")
	}
	if _, _, err := EstimatePhase(u, eig, 0); err == nil {
		t.Error("numCounting < 1 must error")
	}
	twoQubit, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New(2): %v", err)
	}
	if _, _, err := EstimatePhase(u, twoQubit, 3); err == nil {
		t.Error("2-qubit eigenstate must error")
	}
}
