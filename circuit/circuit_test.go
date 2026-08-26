package circuit_test

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

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

// maxQubitsCapBackend wraps a dense state but reports a caller-supplied
// MaxGateQubits bound while always reporting SupportsGateQubits as true.
// This isolates the MaxGateQubits half of BackendCapabilities so it can be
// tested independently of SupportsGateQubits. It also records whether
// ApplyGate was invoked, so tests can confirm a capability failure is
// caught before any state mutation is attempted.
type maxQubitsCapBackend struct {
	*state.State
	maxQubits       int
	applyGateCalled bool
}

func (b *maxQubitsCapBackend) SupportsGateQubits(qubitCount int) bool {
	return true
}

func (b *maxQubitsCapBackend) MaxGateQubits() int {
	return b.maxQubits
}

func (b *maxQubitsCapBackend) ApplyGate(gate quantum.Gate, targets ...int) error {
	b.applyGateCalled = true
	return b.State.ApplyGate(gate, targets...)
}

func TestNewCircuitInvalidQubits(t *testing.T) {
	_, err := circuit.New(0)
	if err == nil {
		t.Fatalf("expected error for invalid qubit count")
	}

	var targetErr *quantum.InvalidQubitCountError
	if !errors.As(err, &targetErr) {
		t.Fatalf("expected InvalidQubitCountError, got %T", err)
	}
	if targetErr.Requested != 0 {
		t.Fatalf("expected Requested=0, got %d", targetErr.Requested)
	}
}

func TestAddGateTargetValidation(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = c.AddGate(gates.NewHadamard(), 1)
	if err == nil {
		t.Fatalf("expected error for out-of-range target")
	}

	var rangeErr *quantum.QubitsOutOfRangeError
	if !errors.As(err, &rangeErr) {
		t.Fatalf("expected QubitsOutOfRangeError, got %T", err)
	}
}

func TestAddGateTargetCountMismatch(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = c.AddGate(gates.NewHadamard(), 0, 1)
	if err == nil {
		t.Fatalf("expected error for invalid target count")
	}

	var gateErr *quantum.InvalidGateApplicationError
	if !errors.As(err, &gateErr) {
		t.Fatalf("expected InvalidGateApplicationError, got %T", err)
	}
}

func TestExecuteAppliesOperations(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := c.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prob := s.Probability(1); math.Abs(prob-1.0) > 1e-10 {
		t.Fatalf("expected |1⟩ probability 1.0, got %v", prob)
	}
}

func TestExecuteMultiQubitCircuits(t *testing.T) {
	invSqrt2 := 1 / math.Sqrt(2)
	tests := []struct {
		name      string
		numQubits int
		setup     func(*circuit.Circuit) error
		want      map[int]complex128
	}{
		{
			name:      "bell state on two qubits",
			numQubits: 2,
			setup: func(c *circuit.Circuit) error {
				if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
					return err
				}
				return c.AddGate(gates.NewCNOT(), 0, 1)
			},
			want: map[int]complex128{
				0: complex(invSqrt2, 0),
				3: complex(invSqrt2, 0),
			},
		},
		{
			name:      "non-adjacent CNOT wiring",
			numQubits: 3,
			setup: func(c *circuit.Circuit) error {
				if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
					return err
				}
				return c.AddGate(gates.NewCNOT(), 0, 2)
			},
			want: map[int]complex128{
				0: complex(invSqrt2, 0),
				5: complex(invSqrt2, 0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := circuit.New(tt.numQubits)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := tt.setup(c); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			s, err := state.New(tt.numQubits)
			if err != nil {
				t.Fatalf("state.New failed: %v", err)
			}
			if err := c.Execute(s); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			totalStates := 1 << tt.numQubits
			for i := 0; i < totalStates; i++ {
				wantAmp, ok := tt.want[i]
				if !ok {
					wantAmp = 0
				}
				got := s.Amplitude(i)
				if cmplx.Abs(got-wantAmp) > 1e-10 {
					t.Fatalf("state %d amplitude = %v, want %v", i, got, wantAmp)
				}
			}
		})
	}
}

func TestComposeAndAppend(t *testing.T) {
	c1, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c2, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := c1.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c2.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	composed, err := c1.Compose(c2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	if err := composed.Execute(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prob := s.Probability(1); math.Abs(prob-0.0) > 1e-10 {
		t.Fatalf("expected |1⟩ probability 0.0 after double X, got %v", prob)
	}

	if err := c1.Append(c2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, err = state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	if err := c1.Execute(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prob := s.Probability(1); math.Abs(prob-0.0) > 1e-10 {
		t.Fatalf("expected |1⟩ probability 0.0 after append, got %v", prob)
	}
}

func TestExecuteQubitMismatch(t *testing.T) {
	c, err := circuit.New(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	err = c.Execute(s)
	if err == nil {
		t.Fatalf("expected error for mismatched qubit count")
	}

	var countErr *quantum.IncompatibleQubitCountError
	if !errors.As(err, &countErr) {
		t.Fatalf("expected IncompatibleQubitCountError, got %T", err)
	}
}

func ExampleCircuit() {
	c, _ := circuit.New(1)
	_ = c.AddGate(gates.NewHadamard(), 0)
	_ = c.AddGate(gates.NewPauliX(), 0)

	s, _ := state.New(1)
	_ = c.Execute(s)
}

func ExampleCircuit_bellState() {
	c, _ := circuit.New(2)
	_ = c.AddGate(gates.NewHadamard(), 0)
	_ = c.AddGate(gates.NewCNOT(), 0, 1)

	s, _ := state.New(2)
	_ = c.Execute(s)
}

func ExampleCircuit_nonAdjacentCNOT() {
	c, _ := circuit.New(3)
	_ = c.AddGate(gates.NewHadamard(), 0)
	_ = c.AddGate(gates.NewCNOT(), 0, 2)

	s, _ := state.New(3)
	_ = c.Execute(s)
}

func TestExecuteWithSparseBackendCapabilityCheck(t *testing.T) {
	// Test that sparse backend rejects 3-qubit gates during circuit execution
	c, err := circuit.New(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add a 3-qubit gate
	if err := c.AddGate(newMockThreeQubitGate(), 0, 1, 2); err != nil {
		t.Fatalf("unexpected error adding gate: %v", err)
	}

	// Sparse backend should fail with UnsupportedOperationError
	sparse, err := sparsestate.New(3)
	if err != nil {
		t.Fatalf("sparsestate.New failed: %v", err)
	}
	err = c.Execute(sparse)
	if err == nil {
		t.Fatal("expected error for 3-qubit gate on sparse backend")
	}

	var unsupportedErr *quantum.UnsupportedOperationError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected UnsupportedOperationError, got %T: %v", err, err)
	}
}

func TestExecuteWithDenseBackendCapabilityCheck(t *testing.T) {
	// Test that dense backend accepts all gate sizes
	c, err := circuit.New(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add a 3-qubit gate
	if err := c.AddGate(newMockThreeQubitGate(), 0, 1, 2); err != nil {
		t.Fatalf("unexpected error adding gate: %v", err)
	}

	// Dense backend should succeed
	dense, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	err = c.Execute(dense)
	if err != nil {
		t.Fatalf("unexpected error on dense backend: %v", err)
	}
}

func TestCheckBackendCapabilities(t *testing.T) {
	// Test explicit capability checking
	c, err := circuit.New(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := c.AddGate(newMockThreeQubitGate(), 0, 1, 2); err != nil {
		t.Fatalf("unexpected error adding gate: %v", err)
	}

	// Check sparse backend - should fail
	sparse, err := sparsestate.New(3)
	if err != nil {
		t.Fatalf("sparsestate.New failed: %v", err)
	}
	err = c.CheckBackendCapabilities(sparse)
	if err == nil {
		t.Fatal("expected error checking sparse backend capabilities")
	}

	var unsupportedErr *quantum.UnsupportedOperationError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected UnsupportedOperationError, got %T: %v", err, err)
	}

	// Check dense backend - should succeed
	dense, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	err = c.CheckBackendCapabilities(dense)
	if err != nil {
		t.Fatalf("unexpected error checking dense backend capabilities: %v", err)
	}
}

func TestCheckBackendCapabilitiesNil(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = c.CheckBackendCapabilities(nil)
	if err == nil {
		t.Fatal("expected error for nil capabilities")
	}
}

func TestCircuitWithMixedGatesCapabilityCheck(t *testing.T) {
	// Test circuit with mix of 1, 2, and 3-qubit gates
	c, err := circuit.New(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(newMockThreeQubitGate(), 0, 1, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sparse should fail on the 3-qubit gate
	sparse, err := sparsestate.New(3)
	if err != nil {
		t.Fatalf("sparsestate.New failed: %v", err)
	}
	err = c.Execute(sparse)
	if err == nil {
		t.Fatal("expected error for 3-qubit gate on sparse backend")
	}

	var unsupportedErr *quantum.UnsupportedOperationError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected UnsupportedOperationError, got %T: %v", err, err)
	}

	// Dense should succeed
	dense, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	err = c.Execute(dense)
	if err != nil {
		t.Fatalf("unexpected error on dense backend: %v", err)
	}
}

// TestExecuteEnforcesMaxGateQubits verifies that MaxGateQubits is enforced
// as a hard upper bound on gate width even when SupportsGateQubits reports
// support for that width, and that a violation is caught before any state
// mutation is attempted.
func TestExecuteEnforcesMaxGateQubits(t *testing.T) {
	tests := []struct {
		name      string
		maxQubits int
		wantErr   bool
	}{
		{name: "gate size below max", maxQubits: 4, wantErr: false},
		{name: "gate size equals max (boundary)", maxQubits: 3, wantErr: false},
		{name: "gate size exceeds max", maxQubits: 2, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := circuit.New(3)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := c.AddGate(newMockThreeQubitGate(), 0, 1, 2); err != nil {
				t.Fatalf("unexpected error adding gate: %v", err)
			}

			dense, err := state.New(3)
			if err != nil {
				t.Fatalf("state.New failed: %v", err)
			}
			backend := &maxQubitsCapBackend{State: dense, maxQubits: tt.maxQubits}

			err = c.Execute(backend)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !backend.applyGateCalled {
					t.Fatal("expected ApplyGate to be called when gate size is within MaxGateQubits")
				}
				return
			}

			if err == nil {
				t.Fatal("expected error when gate size exceeds MaxGateQubits")
			}

			var unsupportedErr *quantum.UnsupportedOperationError
			if !errors.As(err, &unsupportedErr) {
				t.Fatalf("expected UnsupportedOperationError, got %T: %v", err, err)
			}
			if backend.applyGateCalled {
				t.Fatal("ApplyGate must not be called when the MaxGateQubits check fails (fail fast, no mutation)")
			}
		})
	}
}

// TestCheckBackendCapabilitiesMaxGateQubits exercises the MaxGateQubits
// check via CheckBackendCapabilities directly (no state mutation involved).
func TestCheckBackendCapabilitiesMaxGateQubits(t *testing.T) {
	tests := []struct {
		name      string
		maxQubits int
		wantErr   bool
	}{
		{
			name:      "gate size exceeds max is rejected",
			maxQubits: 2,
			wantErr:   true,
		},
		{
			name:      "gate size equal to max is allowed",
			maxQubits: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := circuit.New(3)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := c.AddGate(newMockThreeQubitGate(), 0, 1, 2); err != nil {
				t.Fatalf("unexpected error adding gate: %v", err)
			}

			dense, err := state.New(3)
			if err != nil {
				t.Fatalf("state.New failed: %v", err)
			}
			backend := &maxQubitsCapBackend{State: dense, maxQubits: tt.maxQubits}

			err = c.CheckBackendCapabilities(backend)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error when gate size exceeds MaxGateQubits")
			}

			var unsupportedErr *quantum.UnsupportedOperationError
			if !errors.As(err, &unsupportedErr) {
				t.Fatalf("expected UnsupportedOperationError, got %T: %v", err, err)
			}
		})
	}
}
