package circuit_test

import (
	"math"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

func TestExecuteAllAppliesCircuits(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s1, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	s2, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	states := []*state.State{s1, s2}
	executions := []circuit.Execution{
		{Circuit: c, State: states[0]},
		{Circuit: c, State: states[1]},
	}

	if err := circuit.ExecuteAll(executions); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, s := range states {
		if prob := s.Probability(1); math.Abs(prob-1.0) > 1e-10 {
			t.Fatalf("execution %d probability = %v, want 1.0", i, prob)
		}
	}
}

func TestExecuteAllParallelAppliesCircuits(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s1, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	s2, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	s3, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	states := []*state.State{s1, s2, s3}
	executions := []circuit.Execution{
		{Circuit: c, State: states[0]},
		{Circuit: c, State: states[1]},
		{Circuit: c, State: states[2]},
	}

	err = circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, s := range states {
		if prob := s.Probability(1); math.Abs(prob-1.0) > 1e-10 {
			t.Fatalf("execution %d probability = %v, want 1.0", i, prob)
		}
	}
}

func TestExecuteAllParallelReportsError(t *testing.T) {
	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	executions := []circuit.Execution{
		{Circuit: nil, State: s},
	}

	err = circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 2})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "execution 0") {
		t.Fatalf("expected execution index in error, got %v", err)
	}
}
