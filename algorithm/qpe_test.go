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
