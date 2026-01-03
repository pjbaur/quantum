package quantum_test

import (
	"fmt"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/state"
)

// TestQubitInitialization tests the initialization of qubits
func TestQubitInitialization(t *testing.T) {
	// Test default initialization (|0⟩ state)
	q := qubit.New()
	if q.Alpha() != 1.0 || q.Beta() != 0.0 {
		t.Errorf("New qubit should be in |0⟩ state, got α=%v, β=%v", q.Alpha(), q.Beta())
	}

	// Test initialization with specific values
	tests := []struct {
		name      string
		alpha     complex128
		beta      complex128
		expectErr bool
	}{
		{"Valid |0⟩ state", 1.0, 0.0, false},
		{"Valid |1⟩ state", 0.0, 1.0, false},
		{"Valid |+⟩ state", complex(1/math.Sqrt2, 0), complex(1/math.Sqrt2, 0), false},
		{"Valid |-⟩ state", complex(1/math.Sqrt2, 0), complex(-1/math.Sqrt2, 0), false},
		{"Valid with phase", complex(0.5, 0), complex(0, math.Sqrt(3)/2), false}, // Approximately |0⟩ + i|1⟩, normalized
		{"Invalid - not normalized", 1.0, 1.0, true},
		{"Invalid - zero state", 0.0, 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := qubit.NewWithValues(tt.alpha, tt.beta)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				// Check if the error is of the expected type
				if _, ok := err.(*quantum.NormalizationError); !ok {
					t.Errorf("Expected NormalizationError but got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if q.Alpha() != tt.alpha || q.Beta() != tt.beta {
					t.Errorf("Expected α=%v, β=%v but got α=%v, β=%v",
						tt.alpha, tt.beta, q.Alpha(), q.Beta())
				}
			}
		})
	}
}

// TestQubitSet tests setting qubit values
func TestQubitSet(t *testing.T) {
	q := qubit.New()

	// Test valid state transitions
	validStates := []struct {
		alpha complex128
		beta  complex128
	}{
		{1.0, 0.0}, // |0⟩
		{0.0, 1.0}, // |1⟩
		{complex(1/math.Sqrt2, 0), complex(1/math.Sqrt2, 0)},  // |+⟩
		{complex(1/math.Sqrt2, 0), complex(-1/math.Sqrt2, 0)}, // |-⟩
		{complex(0, 1/math.Sqrt2), complex(0, 1/math.Sqrt2)},  // With phase
	}

	for i, state := range validStates {
		err := q.Set(state.alpha, state.beta)
		if err != nil {
			t.Errorf("Case %d: Unexpected error setting valid state: %v", i, err)
		}
		if q.Alpha() != state.alpha || q.Beta() != state.beta {
			t.Errorf("Case %d: Expected α=%v, β=%v but got α=%v, β=%v",
				i, state.alpha, state.beta, q.Alpha(), q.Beta())
		}
	}

	// Test invalid state transitions
	invalidStates := []struct {
		alpha complex128
		beta  complex128
	}{
		{2.0, 0.0},               // Not normalized
		{1.0, 1.0},               // Not normalized
		{0.5, 0.5},               // Not normalized
		{0.0, 0.0},               // Zero state
		{complex(0.7, 0.7), 0.0}, // Complex values not normalized
	}

	for i, state := range invalidStates {
		err := q.Set(state.alpha, state.beta)
		if err == nil {
			t.Errorf("Case %d: Expected error with invalid state but got nil", i)
		}
		// Check error type
		if _, ok := err.(*quantum.NormalizationError); !ok {
			t.Errorf("Case %d: Expected NormalizationError but got %T", i, err)
		}
	}
}

// TestQubitMeasurement tests qubit measurement operations
func TestQubitMeasurement(t *testing.T) {
	// Test |0⟩ state - should always measure 0
	q := qubit.New()
	result := q.Measure()
	if result != 0 {
		t.Errorf("Measuring |0⟩ state should give 0, got %d", result)
	}
	if q.Alpha() != 1.0 || q.Beta() != 0.0 {
		t.Errorf("After measuring |0⟩, qubit should remain |0⟩, got α=%v, β=%v", q.Alpha(), q.Beta())
	}

	// Test |1⟩ state - should always measure 1
	q, _ = qubit.NewWithValues(0.0, 1.0)
	result = q.Measure()
	if result != 1 {
		t.Errorf("Measuring |1⟩ state should give 1, got %d", result)
	}
	if q.Alpha() != 0.0 || q.Beta() != 1.0 {
		t.Errorf("After measuring |1⟩, qubit should remain |1⟩, got α=%v, β=%v", q.Alpha(), q.Beta())
	}

	// Test |+⟩ state - should collapse to either |0⟩ or |1⟩
	q, _ = qubit.NewWithValues(complex(1/math.Sqrt2, 0), complex(1/math.Sqrt2, 0))
	result = q.Measure()
	if result != 0 && result != 1 {
		t.Errorf("Measuring |+⟩ state should give 0 or 1, got %d", result)
	}
	// After measurement, should collapse to the basis state
	if result == 0 && (q.Alpha() != 1.0 || q.Beta() != 0.0) {
		t.Errorf("After measuring |+⟩ with result 0, qubit should be |0⟩, got α=%v, β=%v", q.Alpha(), q.Beta())
	}
	if result == 1 && (q.Alpha() != 0.0 || q.Beta() != 1.0) {
		t.Errorf("After measuring |+⟩ with result 1, qubit should be |1⟩, got α=%v, β=%v", q.Alpha(), q.Beta())
	}
}

// TestQubitProbabilities tests the probability calculation functions
func TestQubitProbabilities(t *testing.T) {
	tests := []struct {
		name  string
		alpha complex128
		beta  complex128
		prob0 float64
		prob1 float64
	}{
		{"State |0⟩", 1.0, 0.0, 1.0, 0.0},
		{"State |1⟩", 0.0, 1.0, 0.0, 1.0},
		{"State |+⟩", complex(1/math.Sqrt2, 0), complex(1/math.Sqrt2, 0), 0.5, 0.5},
		{"State with phase", complex(0.6, 0), complex(0, 0.8), 0.36, 0.64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, _ := qubit.NewWithValues(tt.alpha, tt.beta)

			// Check probability of |0⟩
			if math.Abs(q.Probability0()-tt.prob0) > 1e-10 {
				t.Errorf("Expected P(|0⟩) = %v, got %v", tt.prob0, q.Probability0())
			}

			// Check probability of |1⟩
			if math.Abs(q.Probability1()-tt.prob1) > 1e-10 {
				t.Errorf("Expected P(|1⟩) = %v, got %v", tt.prob1, q.Probability1())
			}

			// Check that probabilities sum to 1
			if math.Abs(q.Probability0()+q.Probability1()-1.0) > 1e-10 {
				t.Errorf("Probabilities should sum to 1, got P(0) + P(1) = %v",
					q.Probability0()+q.Probability1())
			}
		})
	}
}

// TestQubitClone tests the qubit cloning functionality
func TestQubitClone(t *testing.T) {
	// Create a qubit in an arbitrary state
	original, err := qubit.NewWithValues(complex(0.8, 0.0), complex(0.6, 0.0))
	if err != nil {
		t.Fatalf("Failed to create qubit: %v", err)
	}

	// Clone the qubit
	clone := original.Clone()

	// Check that the clone has the same values
	if clone.Alpha() != original.Alpha() || clone.Beta() != original.Beta() {
		t.Errorf("Clone should have same values as original. Got α=%v vs α=%v, β=%v vs β=%v",
			clone.Alpha(), original.Alpha(), clone.Beta(), original.Beta())
	}

	// Modify the original and check that clone doesn't change
	originalAlpha := original.Alpha()
	originalBeta := original.Beta()
	_ = original.Set(1.0, 0.0)

	if clone.Alpha() != originalAlpha || clone.Beta() != originalBeta {
		t.Errorf("Modifying original should not affect clone. Got α=%v vs α=%v, β=%v vs β=%v",
			clone.Alpha(), originalAlpha, clone.Beta(), originalBeta)
	}
}

// TestGateHadamard tests the Hadamard gate
func TestGateHadamard(t *testing.T) {
	h := gates.NewHadamard()

	// Check name
	if h.Name() != "Hadamard" {
		t.Errorf("Expected gate name 'Hadamard', got '%s'", h.Name())
	}

	// Check matrix representation
	matrix := h.Matrix()
	expectedMatrix := [][]complex128{
		{1 / math.Sqrt2, 1 / math.Sqrt2},
		{1 / math.Sqrt2, -1 / math.Sqrt2},
	}

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if cmplx.Abs(matrix[i][j]-expectedMatrix[i][j]) > 1e-10 {
				t.Errorf("Matrix element [%d][%d] incorrect: expected %v, got %v",
					i, j, expectedMatrix[i][j], matrix[i][j])
			}
		}
	}

	// Test Hadamard on |0⟩ (should give |+⟩)
	q := qubit.New()
	err := h.Apply(q)
	if err != nil {
		t.Errorf("Unexpected error applying H to |0⟩: %v", err)
	}

	expectedAlpha := complex(1/math.Sqrt2, 0)
	expectedBeta := complex(1/math.Sqrt2, 0)

	if cmplx.Abs(q.Alpha()-expectedAlpha) > 1e-10 || cmplx.Abs(q.Beta()-expectedBeta) > 1e-10 {
		t.Errorf("H|0⟩ should be |+⟩. Expected α=%v, β=%v, got α=%v, β=%v",
			expectedAlpha, expectedBeta, q.Alpha(), q.Beta())
	}

	// Test Hadamard on |1⟩ (should give |-⟩)
	q, _ = qubit.NewWithValues(0.0, 1.0)
	err = h.Apply(q)
	if err != nil {
		t.Errorf("Unexpected error applying H to |1⟩: %v", err)
	}

	expectedAlpha = complex(1/math.Sqrt2, 0)
	expectedBeta = complex(-1/math.Sqrt2, 0)

	if cmplx.Abs(q.Alpha()-expectedAlpha) > 1e-10 || cmplx.Abs(q.Beta()-expectedBeta) > 1e-10 {
		t.Errorf("H|1⟩ should be |-⟩. Expected α=%v, β=%v, got α=%v, β=%v",
			expectedAlpha, expectedBeta, q.Alpha(), q.Beta())
	}

	// Test Hadamard twice (should return to original state)
	q = qubit.New() // |0⟩
	_ = h.Apply(q)  // |+⟩
	_ = h.Apply(q)  // should be |0⟩ again

	if cmplx.Abs(q.Alpha()-1.0) > 1e-10 || cmplx.Abs(q.Beta()) > 1e-10 {
		t.Errorf("H²|0⟩ should be |0⟩. Got α=%v, β=%v", q.Alpha(), q.Beta())
	}
}

// TestGatePauliX tests the Pauli-X (NOT) gate
func TestGatePauliX(t *testing.T) {
	x := gates.NewPauliX()

	// Check name
	if x.Name() != "PauliX" {
		t.Errorf("Expected gate name 'PauliX', got '%s'", x.Name())
	}

	// Test X on |0⟩ (should give |1⟩)
	q := qubit.New()
	err := x.Apply(q)
	if err != nil {
		t.Errorf("Unexpected error applying X to |0⟩: %v", err)
	}

	if q.Alpha() != 0.0 || q.Beta() != 1.0 {
		t.Errorf("X|0⟩ should be |1⟩. Got α=%v, β=%v", q.Alpha(), q.Beta())
	}

	// Test X on |1⟩ (should give |0⟩)
	q, _ = qubit.NewWithValues(0.0, 1.0)
	err = x.Apply(q)
	if err != nil {
		t.Errorf("Unexpected error applying X to |1⟩: %v", err)
	}

	if q.Alpha() != 1.0 || q.Beta() != 0.0 {
		t.Errorf("X|1⟩ should be |0⟩. Got α=%v, β=%v", q.Alpha(), q.Beta())
	}

	// Test X gate on superposition |+⟩
	q, _ = qubit.NewWithValues(complex(1/math.Sqrt2, 0), complex(1/math.Sqrt2, 0))
	err = x.Apply(q)
	if err != nil {
		t.Errorf("Unexpected error applying X to |+⟩: %v", err)
	}

	// X|+⟩ = |+⟩
	if cmplx.Abs(q.Alpha()-complex(1/math.Sqrt2, 0)) > 1e-10 ||
		cmplx.Abs(q.Beta()-complex(1/math.Sqrt2, 0)) > 1e-10 {
		t.Errorf("X|+⟩ should be |+⟩. Got α=%v, β=%v", q.Alpha(), q.Beta())
	}
}

// TestGateT tests the T (π/8) gate
func TestGateT(t *testing.T) {
	tGate := gates.NewT()

	// Check name
	if tGate.Name() != "T" {
		t.Errorf("Expected gate name 'T', got '%s'", tGate.Name())
	}

	// Test T on |0⟩ (should remain |0⟩)
	q := qubit.New()
	err := tGate.Apply(q)
	if err != nil {
		t.Errorf("Unexpected error applying T to |0⟩: %v", err)
	}

	if q.Alpha() != 1.0 || q.Beta() != 0.0 {
		t.Errorf("T|0⟩ should be |0⟩. Got α=%v, β=%v", q.Alpha(), q.Beta())
	}

	// Test T on |1⟩ (should add phase e^(iπ/4))
	q, _ = qubit.NewWithValues(0.0, 1.0)
	err = tGate.Apply(q)
	if err != nil {
		t.Errorf("Unexpected error applying T to |1⟩: %v", err)
	}

	expectedPhase := complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))
	if cmplx.Abs(q.Alpha()) > 1e-10 || cmplx.Abs(q.Beta()-expectedPhase) > 1e-10 {
		t.Errorf("T|1⟩ should be e^(iπ/4)|1⟩. Got α=%v, β=%v", q.Alpha(), q.Beta())
	}

	// Test 8 applications of T (should return to original state)
	q, _ = qubit.NewWithValues(0.0, 1.0)
	for i := 0; i < 8; i++ {
		err = tGate.Apply(q)
		if err != nil {
			t.Errorf("Unexpected error in T gate application %d: %v", i, err)
		}
	}

	// Check that we're back to |1⟩ after 8 T gates (modulo global phase)
	// Due to floating point, we'll check magnitude of α is close to 0 and β close to 1
	if cmplx.Abs(q.Alpha()) > 1e-10 || math.Abs(cmplx.Abs(q.Beta())-1.0) > 1e-10 {
		t.Errorf("T⁸|1⟩ should be |1⟩. Got α=%v, β=%v", q.Alpha(), q.Beta())
	}
}

// TestQuantumState tests basic quantum state functionality
func TestQuantumState(t *testing.T) {
	// Create a 2-qubit system
	s := state.New(2)

	// Check initial state is |00⟩
	if s.NumQubits() != 2 {
		t.Errorf("Expected 2 qubits, got %d", s.NumQubits())
	}

	if s.Amplitude(0) != 1.0 {
		t.Errorf("Initial state should be |00⟩ (amplitude 1.0 at index 0), got %v", s.Amplitude(0))
	}

	for i := 1; i < 4; i++ {
		if s.Amplitude(i) != 0.0 {
			t.Errorf("Initial state should have amplitude 0 at index %d, got %v", i, s.Amplitude(i))
		}
	}

	// Test measurement of |00⟩ (should always be 0 for both qubits)
	bit0, err := s.Measure(0)
	if err != nil || bit0 != 0 {
		t.Errorf("Measuring qubit 0 of |00⟩ should give 0, got %d with error: %v", bit0, err)
	}

	bit1, err := s.Measure(1)
	if err != nil || bit1 != 0 {
		t.Errorf("Measuring qubit 1 of |00⟩ should give 0, got %d with error: %v", bit1, err)
	}
}

// TestQuantumStateGateApplication tests applying gates to a quantum state
func TestQuantumStateGateApplication(t *testing.T) {
	// Create a 2-qubit system
	s := state.New(2)

	// Apply Hadamard to first qubit (least significant bit)
	h := gates.NewHadamard()
	err := s.ApplyGate(h, 0)
	if err != nil {
		t.Errorf("Error applying Hadamard to qubit 0: %v", err)
	}

	// Should get (|00⟩ + |01⟩)/√2 in little-endian ordering
	expectedAmplitudes := []complex128{
		complex(1/math.Sqrt2, 0),
		complex(1/math.Sqrt2, 0),
		0,
		0,
	}

	for i, expected := range expectedAmplitudes {
		if cmplx.Abs(s.Amplitude(i)-expected) > 1e-10 {
			t.Errorf("Incorrect amplitude at index %d: expected %v, got %v",
				i, expected, s.Amplitude(i))
		}
	}

	// Apply Hadamard to second qubit
	err = s.ApplyGate(h, 1)
	if err != nil {
		t.Errorf("Error applying Hadamard to qubit 1: %v", err)
	}

	// Should get (|00⟩ + |01⟩ + |10⟩ + |11⟩)/2
	expectedAmplitudes = []complex128{
		0.5, 0.5, 0.5, 0.5,
	}

	for i, expected := range expectedAmplitudes {
		if cmplx.Abs(s.Amplitude(i)-complex(real(expected), 0)) > 1e-10 {
			t.Errorf("Incorrect amplitude at index %d: expected %v, got %v",
				i, expected, s.Amplitude(i))
		}
	}

	// Apply X to first qubit
	x := gates.NewPauliX()
	err = s.ApplyGate(x, 0)
	if err != nil {
		t.Errorf("Error applying PauliX to qubit 0: %v", err)
	}

	// Should flip the qubits where first qubit is 0 with those where it's 1
	// This means |00⟩ <-> |10⟩ and |01⟩ <-> |11⟩
	expectedAmplitudes = []complex128{
		0.5, 0.5, 0.5, 0.5,
	}

	for i, expected := range expectedAmplitudes {
		if cmplx.Abs(s.Amplitude(i)-complex(real(expected), 0)) > 1e-10 {
			t.Errorf("Incorrect amplitude after X on qubit 0, index %d: expected %v, got %v",
				i, expected, s.Amplitude(i))
		}
	}
}

// TestBellState tests creating and measuring a Bell state
func TestBellState(t *testing.T) {
	// Create a 2-qubit system
	s := state.New(2)

	// Apply Hadamard to first qubit
	h := gates.NewHadamard()
	err := s.ApplyGate(h, 0)
	if err != nil {
		t.Errorf("Error applying Hadamard to qubit 0: %v", err)
	}

	// Apply CNOT with control=0, target=1
	cnot := gates.NewCNOT()
	err = s.ApplyGate(cnot, 0, 1)
	if err != nil {
		t.Errorf("Error applying CNOT gate: %v", err)
	}

	// Should get Bell state (|00⟩ + |11⟩)/√2
	expectedAmplitudes := []complex128{
		complex(1/math.Sqrt2, 0),
		0,
		0,
		complex(1/math.Sqrt2, 0),
	}

	for i, expected := range expectedAmplitudes {
		if cmplx.Abs(s.Amplitude(i)-expected) > 1e-10 {
			t.Errorf("Incorrect Bell state amplitude at index %d: expected %v, got %v",
				i, expected, s.Amplitude(i))
		}
	}

	// Measure first qubit
	result0, err := s.Measure(0)
	if err != nil {
		t.Errorf("Error measuring qubit 0: %v", err)
	}

	// Measure second qubit
	result1, err := s.Measure(1)
	if err != nil {
		t.Errorf("Error measuring qubit 1: %v", err)
	}

	// In a Bell state, both qubits should have the same measurement outcome
	if result0 != result1 {
		t.Errorf("Bell state measurement failed: qubits measured to different values %d and %d",
			result0, result1)
	}
}

// TestInvalidOperations tests error handling for invalid operations
func TestInvalidOperations(t *testing.T) {
	// Test out-of-bounds measurement
	s := state.New(2)
	_, err := s.Measure(2)
	if err == nil {
		t.Error("Expected error when measuring out-of-bounds qubit, got nil")
	}
	if _, ok := err.(*quantum.QubitsOutOfRangeError); !ok {
		t.Errorf("Expected QubitsOutOfRangeError, got %T", err)
	}

	// Test out-of-bounds gate application
	h := gates.NewHadamard()
	err = s.ApplyGate(h, 3)
	if err == nil {
		t.Error("Expected error when applying gate to out-of-bounds qubit, got nil")
	}
	if _, ok := err.(*quantum.QubitsOutOfRangeError); !ok {
		t.Errorf("Expected QubitsOutOfRangeError, got %T", err)
	}

	// Test applying a multi-qubit gate to a single qubit
	cnot := gates.NewCNOT()
	err = s.ApplyGate(cnot, 0)
	if err == nil {
		t.Error("Expected error when applying CNOT to a single qubit, got nil")
	}
	if _, ok := err.(*quantum.InvalidGateApplicationError); !ok {
		t.Errorf("Expected InvalidGateApplicationError, got %T", err)
	}
}

// ExampleQubitHadamard demonstrates using the Hadamard gate on a qubit
func Example_hadamardApply() {
	// Create a new qubit in |0⟩ state
	q := qubit.New()

	// Apply Hadamard gate
	h := gates.NewHadamard()
	_ = h.Apply(q)

	// Show the resulting state
	fmt.Printf("H|0⟩ = %.4f|0⟩ + %.4f|1⟩\n", real(q.Alpha()), real(q.Beta()))
	// Output: H|0⟩ = 0.7071|0⟩ + 0.7071|1⟩
}

// ExampleBellState demonstrates creating a Bell state
func Example_bellState() {
	// Create a 2-qubit state
	s := state.New(2)

	// Apply Hadamard to first qubit
	h := gates.NewHadamard()
	_ = s.ApplyGate(h, 0)

	// Apply CNOT with first qubit as control
	cnot := gates.NewCNOT()
	_ = s.ApplyGate(cnot, 0, 1)

	// Print the state amplitudes
	fmt.Printf("Bell state: %.4f|00⟩ + %.4f|11⟩\n",
		real(s.Amplitude(0)), real(s.Amplitude(3)))
	// Output: Bell state: 0.7071|00⟩ + 0.7071|11⟩
}
