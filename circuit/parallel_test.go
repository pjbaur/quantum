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

// uncomparableState is a deliberately uncomparable quantum.QuantumState
// implementation: it is used as a struct value (not a pointer) and holds a
// slice field, so comparing two values of this type panics. It exists to
// verify that validateIndependentStates rejects such states with a typed
// error instead of panicking when it keys its seen-state map.
type uncomparableState struct {
	amplitudes []complex128
}

func (u uncomparableState) NumQubits() int                                      { return 1 }
func (u uncomparableState) Amplitude(basisState int) complex128                 { return u.amplitudes[basisState] }
func (u uncomparableState) SetAmplitude(basisState int, value complex128) error { return nil }
func (u uncomparableState) ApplyGate(gate quantum.Gate, targets ...int) error   { return nil }
func (u uncomparableState) Measure(qubitIndex int) (int, error)                 { return 0, nil }
func (u uncomparableState) Probability(basisState int) float64                  { return 0 }
func (u uncomparableState) Clone() quantum.QuantumState                         { return u }

func TestExecuteAllParallelValidatesStateComparability(t *testing.T) {
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

	tests := []struct {
		name       string
		executions []circuit.Execution
		checkErr   func(t *testing.T, err error)
	}{
		{
			name: "pointer states still validate as before",
			executions: []circuit.Execution{
				{Circuit: c, State: s1},
				{Circuit: c, State: s2},
			},
			checkErr: func(t *testing.T, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
		{
			name: "uncomparable state returns typed error instead of panicking",
			executions: []circuit.Execution{
				{Circuit: c, State: uncomparableState{amplitudes: []complex128{1, 0}}},
				{Circuit: c, State: uncomparableState{amplitudes: []complex128{1, 0}}},
			},
			checkErr: func(t *testing.T, err error) {
				var uncomparableErr *quantum.UncomparableStateError
				if !errors.As(err, &uncomparableErr) {
					t.Fatalf("expected UncomparableStateError, got %T: %v", err, err)
				}
				if uncomparableErr.Index != 0 {
					t.Fatalf("expected index 0, got %d", uncomparableErr.Index)
				}
				if !strings.Contains(err.Error(), "uncomparableState") {
					t.Fatalf("expected error message to name the type, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ExecuteAllParallel panicked: %v", r)
				}
			}()
			err := circuit.ExecuteAllParallel(tt.executions, circuit.ParallelOptions{MaxParallelism: 2})
			tt.checkErr(t, err)
		})
	}
}

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

func TestExecuteAllParallelDetectsSharedState(t *testing.T) {
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create a single state and reuse it (this is the bug we're detecting)
	sharedState, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
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
	sharedState, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	s1, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	s2, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	executions := []circuit.Execution{
		{Circuit: c, State: s1},
		{Circuit: c, State: s2},
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
	executions := []circuit.Execution{
		{Circuit: c, State: s1}, // Same circuit
		{Circuit: c, State: s2}, // Same circuit
		{Circuit: c, State: s3}, // Same circuit
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

	sharedState, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	executions := []circuit.Execution{
		{Circuit: c, State: sharedState},
		{Circuit: c, State: sharedState},
	}

	err = circuit.ExecuteAll(executions)
	if err != nil {
		t.Fatalf("serial execution should allow shared state: %v", err)
	}
}

// TestExecuteAllParallelRaceDetection tests that concurrent execution with
// independent states does not cause data races. Run with `go test -race`.
func TestExecuteAllParallelRaceDetection(t *testing.T) {
	// Create a shared circuit (read-only, safe to share)
	c, err := circuit.New(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create many independent states - this tests concurrent access
	const numStates = 100
	executions := make([]circuit.Execution, numStates)
	for i := 0; i < numStates; i++ {
		s, err := state.New(1)
		if err != nil {
			t.Fatalf("state.New failed: %v", err)
		}
		executions[i] = circuit.Execution{Circuit: c, State: s}
	}

	// Execute with high parallelism to stress-test for races
	err = circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all states were modified correctly
	for i, exec := range executions {
		// After Hadamard, probability of measuring |0> should be 0.5
		prob := exec.State.Probability(0)
		if math.Abs(prob-0.5) > 1e-10 {
			t.Fatalf("state %d: probability of |0> = %v, want 0.5", i, prob)
		}
	}
}

// TestExecuteAllParallelConcurrentReads tests that multiple concurrent
// executions don't cause races when reading circuit data.
func TestExecuteAllParallelConcurrentReads(t *testing.T) {
	// Create a more complex circuit to test concurrent reads
	c, err := circuit.New(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const numStates = 50
	executions := make([]circuit.Execution, numStates)
	for i := 0; i < numStates; i++ {
		s, err := state.New(2)
		if err != nil {
			t.Fatalf("state.New failed: %v", err)
		}
		executions[i] = circuit.Execution{Circuit: c, State: s}
	}

	err = circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Bell state was created in each execution
	for i, exec := range executions {
		// For Bell state, probability of measuring |00> should be 0.5
		prob00 := exec.State.Probability(0)
		if math.Abs(prob00-0.5) > 1e-10 {
			t.Fatalf("state %d: probability of |00> = %v, want 0.5", i, prob00)
		}
	}
}

// TestExecuteAllParallelStressTest is designed to trigger potential race
// conditions when run with `go test -race`.
func TestExecuteAllParallelStressTest(t *testing.T) {
	// Create multiple different circuits
	circuits := make([]*circuit.Circuit, 5)
	for i := range circuits {
		c, err := circuit.New(2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := c.AddGate(gates.NewPauliX(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		circuits[i] = c
	}

	// Create executions with independent states
	const numExecutions = 200
	executions := make([]circuit.Execution, numExecutions)
	for i := 0; i < numExecutions; i++ {
		s, err := state.New(2)
		if err != nil {
			t.Fatalf("state.New failed: %v", err)
		}
		// Use different circuits to test concurrent circuit reads
		executions[i] = circuit.Execution{Circuit: circuits[i%len(circuits)], State: s}
	}

	err := circuit.ExecuteAllParallel(executions, circuit.ParallelOptions{MaxParallelism: 16})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
