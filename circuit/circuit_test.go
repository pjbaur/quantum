package circuit_test

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

func TestNewCircuitInvalidQubits(t *testing.T) {
	_, err := circuit.New(0)
	if err == nil {
		t.Fatalf("expected error for invalid qubit count")
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

	s := state.New(1)
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

			s := state.New(tt.numQubits)
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

	s := state.New(1)
	if err := composed.Execute(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prob := s.Probability(1); math.Abs(prob-0.0) > 1e-10 {
		t.Fatalf("expected |1⟩ probability 0.0 after double X, got %v", prob)
	}

	if err := c1.Append(c2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s = state.New(1)
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

	s := state.New(1)
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

	s := state.New(1)
	_ = c.Execute(s)

	fmt.Printf("%.2f %.2f\n", s.Probability(0), s.Probability(1))
	// Output: 0.50 0.50
}

func ExampleCircuit_bellState() {
	c, _ := circuit.New(2)
	_ = c.AddGate(gates.NewHadamard(), 0)
	_ = c.AddGate(gates.NewCNOT(), 0, 1)

	s := state.New(2)
	_ = c.Execute(s)

	fmt.Printf("%.2f %.2f\n", s.Probability(0), s.Probability(3))
	// Output: 0.50 0.50
}

func ExampleCircuit_nonAdjacentCNOT() {
	c, _ := circuit.New(3)
	_ = c.AddGate(gates.NewHadamard(), 0)
	_ = c.AddGate(gates.NewCNOT(), 0, 2)

	s := state.New(3)
	_ = c.Execute(s)

	fmt.Printf("%.2f %.2f\n", s.Probability(0), s.Probability(5))
	// Output: 0.50 0.50
}
