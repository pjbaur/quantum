package sparsestate

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

const tolerance = 1e-10

// mockThreeQubitGate is a test gate that operates on 3 qubits.
type mockThreeQubitGate struct{}

func newMockThreeQubitGate() *mockThreeQubitGate {
	return &mockThreeQubitGate{}
}

func (g *mockThreeQubitGate) Name() string {
	return "Mock3Qubit"
}

func (g *mockThreeQubitGate) Matrix() [][]complex128 {
	// 8x8 identity matrix (3-qubit gate)
	matrix := make([][]complex128, 8)
	for i := range matrix {
		matrix[i] = make([]complex128, 8)
		matrix[i][i] = 1
	}
	return matrix
}

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
			dense, err := state.New(tt.numQubits)
			if err != nil {
				t.Fatalf("dense state.New failed: %v", err)
			}
			sparse, err := New(tt.numQubits)
			if err != nil {
				t.Fatalf("sparse New failed: %v", err)
			}

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
	dense, err := state.New(3)
	if err != nil {
		t.Fatalf("dense state.New failed: %v", err)
	}
	sparse, err := New(3)
	if err != nil {
		t.Fatalf("sparse New failed: %v", err)
	}

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
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("dense state.New failed: %v", err)
	}
	sparse, err := New(2)
	if err != nil {
		t.Fatalf("sparse New failed: %v", err)
	}

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

type denseTwoQubitGate struct {
	matrix [][]complex128
}

func newDenseTwoQubitGate() *denseTwoQubitGate {
	return &denseTwoQubitGate{
		matrix: [][]complex128{
			{1, 0, 0, 0},
			{0, 0, 1, 0},
			{0, 1, 0, 0},
			{0, 0, 0, -1},
		},
	}
}

func (g *denseTwoQubitGate) Name() string {
	return "DenseTwoQubit"
}

func (g *denseTwoQubitGate) Matrix() [][]complex128 {
	return g.matrix
}

func TestSparseTwoQubitTargetOrderingMatchesDense(t *testing.T) {
	tests := []struct {
		name      string
		gate      quantum.Gate
		targets   []int
		numQubits int
		setup     func(quantum.QuantumState) error
	}{
		{
			name:      "cnot reversed non-adjacent targets",
			gate:      gates.NewCNOT(),
			targets:   []int{2, 0},
			numQubits: 3,
			setup: func(qs quantum.QuantumState) error {
				if err := qs.ApplyGate(gates.NewHadamard(), 2); err != nil {
					return err
				}
				return qs.ApplyGate(gates.NewPauliX(), 0)
			},
		},
		{
			name:      "swap reversed targets",
			gate:      gates.NewSwap(),
			targets:   []int{2, 0},
			numQubits: 3,
			setup: func(qs quantum.QuantumState) error {
				if err := qs.ApplyGate(gates.NewHadamard(), 2); err != nil {
					return err
				}
				return qs.ApplyGate(gates.NewPauliX(), 0)
			},
		},
		{
			name:      "generic two-qubit gate with non-sorted targets",
			gate:      newDenseTwoQubitGate(),
			targets:   []int{3, 1},
			numQubits: 4,
			setup: func(qs quantum.QuantumState) error {
				if err := qs.ApplyGate(gates.NewHadamard(), 1); err != nil {
					return err
				}
				return qs.ApplyGate(gates.NewPauliX(), 3)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dense, err := state.New(tt.numQubits)
			if err != nil {
				t.Fatalf("dense state.New failed: %v", err)
			}
			sparse, err := New(tt.numQubits)
			if err != nil {
				t.Fatalf("sparse New failed: %v", err)
			}

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

func TestSparseSetAmplitudeNormalization(t *testing.T) {
	sparse, err := New(2)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

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

func TestSparseBackendCapabilities(t *testing.T) {
	sparse, err := New(3)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Test SupportsGateQubits
	if !sparse.SupportsGateQubits(1) {
		t.Error("expected sparse backend to support 1-qubit gates")
	}
	if !sparse.SupportsGateQubits(2) {
		t.Error("expected sparse backend to support 2-qubit gates")
	}
	if sparse.SupportsGateQubits(3) {
		t.Error("expected sparse backend to NOT support 3-qubit gates")
	}
	if sparse.SupportsGateQubits(4) {
		t.Error("expected sparse backend to NOT support 4-qubit gates")
	}
	if sparse.SupportsGateQubits(0) {
		t.Error("expected sparse backend to NOT support 0-qubit gates")
	}

	// Test MaxGateQubits
	if max := sparse.MaxGateQubits(); max != 2 {
		t.Errorf("expected MaxGateQubits=2, got %d", max)
	}
}

func TestSparseUnsupportedGateError(t *testing.T) {
	sparse, err := New(3)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	gate := newMockThreeQubitGate()

	err = sparse.ApplyGate(gate, 0, 1, 2)
	if err == nil {
		t.Fatal("expected error for 3-qubit gate on sparse backend")
	}

	var unsupportedErr *quantum.UnsupportedOperationError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected UnsupportedOperationError, got %T: %v", err, err)
	}

	// Verify error message contains useful information
	if unsupportedErr.Backend != "sparse" {
		t.Errorf("expected Backend='sparse', got '%s'", unsupportedErr.Backend)
	}
	if unsupportedErr.Alternative == "" {
		t.Error("expected Alternative to be non-empty")
	}
}

func TestSparseGateInterfaceAssertion(t *testing.T) {
	// Verify State implements both QuantumState and BackendCapabilities
	s1, _ := New(1)
	var _ quantum.QuantumState = s1
	var _ quantum.BackendCapabilities = s1
}

func TestNewInvalidQubitCount(t *testing.T) {
	tests := []struct {
		name      string
		numQubits int
		errCheck  func(error) bool
	}{
		{
			name:      "zero qubits",
			numQubits: 0,
			errCheck: func(err error) bool {
				var targetErr *quantum.InvalidQubitCountError
				return errors.As(err, &targetErr) &&
					targetErr.Requested == 0 &&
					targetErr.Reason == "must be positive"
			},
		},
		{
			name:      "negative qubits",
			numQubits: -1,
			errCheck: func(err error) bool {
				var targetErr *quantum.InvalidQubitCountError
				return errors.As(err, &targetErr) &&
					targetErr.Requested == -1 &&
					targetErr.Reason == "must be positive"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.numQubits)
			if err == nil {
				t.Fatalf("expected error for %d qubits, got nil", tt.numQubits)
			}
			if !tt.errCheck(err) {
				t.Fatalf("unexpected error: %T: %v", err, err)
			}
		})
	}
}

func TestNewValidQubitCount(t *testing.T) {
	s, err := New(3)
	if err != nil {
		t.Fatalf("expected no error for 3 qubits, got: %v", err)
	}
	if s.NumQubits() != 3 {
		t.Fatalf("expected 3 qubits, got %d", s.NumQubits())
	}
}
