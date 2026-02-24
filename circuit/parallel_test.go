package circuit_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
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

	states := []*state.State{
		state.New(1),
		state.New(1),
	}
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

	states := []*state.State{
		state.New(1),
		state.New(1),
		state.New(1),
	}
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
	s := state.New(1)
	executions := []circuit.Execution{
		{Circuit: nil, State: s},
	}

	err := circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 2})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "execution 0") {
		t.Fatalf("expected execution index in error, got %v", err)
	}
}

func TestExecuteAllParallelDetectsSharedState(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create a single state and reuse it (this is the bug we're detecting)
	sharedState := state.New(1)
	executions := []circuit.Execution{
		{Circuit: c, State: sharedState},
		{Circuit: c, State: sharedState}, // Same state pointer!
	}

	err = circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 2})
	if err == nil {
		t.Fatalf("expected SharedStateError for shared state")
	}

	// Verify it's the right error type
	var sharedErr *quantum.SharedStateError
	if !errors.As(err, &sharedErr) {
		t.Fatalf("expected SharedStateError, got %T: %v", err, err)
	}

	// Verify error contains duplicate indices
	if len(sharedErr.DuplicateIndices) != 2 {
		t.Fatalf("expected 2 duplicate indices, got %d", len(sharedErr.DuplicateIndices))
	}
	if sharedErr.DuplicateIndices[0] != 0 || sharedErr.DuplicateIndices[1] != 1 {
		t.Fatalf("expected indices [0, 1], got %v", sharedErr.DuplicateIndices)
	}

	// Verify error message is actionable
	if !strings.Contains(err.Error(), "executions [0 1]") {
		t.Fatalf("expected error message to contain 'executions [0 1]', got: %v", err)
	}
}

func TestExecuteAllParallelDetectsSharedStateLater(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create states where first and third share the same pointer
	sharedState := state.New(1)
	executions := []circuit.Execution{
		{Circuit: c, State: state.New(1)},
		{Circuit: c, State: state.New(1)},
		{Circuit: c, State: sharedState},
		{Circuit: c, State: sharedState}, // Duplicate of index 2
	}

	err = circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 2})
	if err == nil {
		t.Fatalf("expected SharedStateError for shared state")
	}

	var sharedErr *quantum.SharedStateError
	if !errors.As(err, &sharedErr) {
		t.Fatalf("expected SharedStateError, got %T: %v", err, err)
	}

	// Should detect indices 2 and 3 as duplicates
	if sharedErr.DuplicateIndices[0] != 2 || sharedErr.DuplicateIndices[1] != 3 {
		t.Fatalf("expected indices [2, 3], got %v", sharedErr.DuplicateIndices)
	}
}

func TestExecuteAllParallelAllowsNilState(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nil states should not cause shared-state detection to fail
	// (they will fail later in executeOne)
	executions := []circuit.Execution{
		{Circuit: c, State: nil},
		{Circuit: c, State: nil},
	}

	err = circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 2})
	// Should get nil state error, not shared state error
	if err == nil {
		t.Fatalf("expected error for nil state")
	}
	var sharedErr *quantum.SharedStateError
	if errors.As(err, &sharedErr) {
		t.Fatalf("should not get SharedStateError for nil states, got: %v", err)
	}
}

func TestExecuteAllParallelAllowsSharedCircuit(t *testing.T) {
	// Multiple executions can safely share the same circuit (it's read-only)
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	executions := []circuit.Execution{
		{Circuit: c, State: state.New(1)}, // Same circuit
		{Circuit: c, State: state.New(1)}, // Same circuit
		{Circuit: c, State: state.New(1)}, // Same circuit
	}

	err = circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 2})
	if err != nil {
		t.Fatalf("shared circuit should be allowed: %v", err)
	}
}

func TestExecuteAllSerialAllowsSharedState(t *testing.T) {
	// ExecuteAll (serial) should not check for shared state since it's safe
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sharedState := state.New(1)
	executions := []circuit.Execution{
		{Circuit: c, State: sharedState},
		{Circuit: c, State: sharedState},
	}

	err = circuit.ExecuteAll(executions)
	if err != nil {
		t.Fatalf("serial execution should allow shared state: %v", err)
	}
}
