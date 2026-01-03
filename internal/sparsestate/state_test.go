package sparsestate

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

const tolerance = 1e-10

func TestSparseSingleQubitMatchesDense(t *testing.T) {
	tests := []struct {
		name      string
		gate      quantum.Gate
		targets   []int
		numQubits int
		setup     func(quantum.QuantumState) error
	}{
		{
			name:      "hadamard on middle qubit",
			gate:      gates.NewHadamard(),
			targets:   []int{1},
			numQubits: 3,
			setup: func(qs quantum.QuantumState) error {
				if err := qs.ApplyGate(gates.NewPauliX(), 0); err != nil {
					return err
				}
				return qs.ApplyGate(gates.NewPauliX(), 2)
			},
		},
		{
			name:      "phase gate on highest qubit",
			gate:      gates.NewS(),
			targets:   []int{2},
			numQubits: 3,
			setup: func(qs quantum.QuantumState) error {
				return qs.ApplyGate(gates.NewHadamard(), 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dense := state.New(tt.numQubits)
			sparse := New(tt.numQubits)

			if tt.setup != nil {
				if err := tt.setup(dense); err != nil {
					t.Fatalf("dense setup failed: %v", err)
				}
				if err := tt.setup(sparse); err != nil {
					t.Fatalf("sparse setup failed: %v", err)
				}
			}

			if err := dense.ApplyGate(tt.gate, tt.targets...); err != nil {
				t.Fatalf("dense ApplyGate failed: %v", err)
			}
			if err := sparse.ApplyGate(tt.gate, tt.targets...); err != nil {
				t.Fatalf("sparse ApplyGate failed: %v", err)
			}

			assertStatesMatch(t, dense, sparse)
		})
	}
}

func TestSparseCNOTMatchesDense(t *testing.T) {
	dense := state.New(3)
	sparse := New(3)

	if err := dense.ApplyGate(gates.NewHadamard(), 1); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewHadamard(), 1); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	if err := dense.ApplyGate(gates.NewCNOT(), 1, 2); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewCNOT(), 1, 2); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	assertStatesMatch(t, dense, sparse)
}

func TestSparseSwapMatchesDense(t *testing.T) {
	dense := state.New(2)
	sparse := New(2)

	if err := dense.ApplyGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	if err := dense.ApplyGate(gates.NewSwap(), 0, 1); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewSwap(), 0, 1); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	assertStatesMatch(t, dense, sparse)
}

func TestSparseSetAmplitudeNormalization(t *testing.T) {
	sparse := New(2)

	if err := sparse.SetAmplitude(0, complex(0.25, 0)); err == nil {
		t.Fatalf("expected normalization error, got nil")
	}
	if amp := sparse.Amplitude(0); cmplx.Abs(amp-1) > tolerance {
		t.Fatalf("expected amplitude to remain 1, got %v", amp)
	}
}

func assertStatesMatch(t *testing.T, dense quantum.QuantumState, sparse quantum.QuantumState) {
	t.Helper()

	if dense.NumQubits() != sparse.NumQubits() {
		t.Fatalf("mismatched qubit counts: %d vs %d", dense.NumQubits(), sparse.NumQubits())
	}

	states := 1 << dense.NumQubits()
	for i := 0; i < states; i++ {
		denseAmp := dense.Amplitude(i)
		sparseAmp := sparse.Amplitude(i)
		if cmplx.Abs(denseAmp-sparseAmp) > tolerance {
			t.Fatalf("amplitude mismatch at %d: dense=%v sparse=%v", i, denseAmp, sparseAmp)
		}
		denseProb := dense.Probability(i)
		sparseProb := sparse.Probability(i)
		if math.Abs(denseProb-sparseProb) > tolerance {
			t.Fatalf("probability mismatch at %d: dense=%v sparse=%v", i, denseProb, sparseProb)
		}
	}
}
