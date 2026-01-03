package circuit_test

import (
	"errors"
	"fmt"
	"math"
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
