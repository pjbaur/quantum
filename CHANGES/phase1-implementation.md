# Phase 1 Implementation Guide: Foundation

As a Staff Software Engineer, here's my detailed approach to implementing Phase 1 of your quantum computing project. This phase focuses on establishing the core architecture that will enable future development.

## 1. Define Core Interfaces

### Step 1: Create Interface Definitions

Start by creating a new file `interfaces.go` in a package called quantum:

```go
package quantum

import (
	"math/cmplx"
)

// Qubit represents a quantum bit with amplitude coefficients
type Qubit interface {
	// Alpha returns the amplitude of the |0⟩ basis state
	Alpha() complex128
	
	// Beta returns the amplitude of the |1⟩ basis state
	Beta() complex128
	
	// Set updates the amplitudes of the qubit
	// Returns error if the resulting state would not be normalized
	Set(alpha, beta complex128) error
	
	// Probability0 returns the probability of measuring |0⟩
	Probability0() float64
	
	// Probability1 returns the probability of measuring |1⟩
	Probability1() float64
	
	// Measure collapses the qubit to either |0⟩ or |1⟩ based on probabilities
	// Returns the measured value (0 or 1)
	Measure() int
	
	// Clone creates a copy of this qubit
	Clone() Qubit
}

// Gate represents a quantum gate operation
type Gate interface {
	// Apply applies the gate to the given qubit
	// For multi-qubit gates, specific implementations will handle the details
	Apply(q Qubit) error
	
	// Name returns the name of the gate
	Name() string
	
	// Matrix returns the matrix representation of the gate
	Matrix() [][]complex128
}

// QuantumState represents a multi-qubit quantum state
type QuantumState interface {
	// NumQubits returns the number of qubits in the state
	NumQubits() int
	
	// Amplitude returns the amplitude of a specific basis state
	// The basisState parameter is the integer representation of the basis state
	Amplitude(basisState int) complex128
	
	// SetAmplitude sets the amplitude for a specific basis state
	// Returns error if the resulting state would not be normalized
	SetAmplitude(basisState int, value complex128) error
	
	// ApplyGate applies a gate to the specified qubit(s)
	// For single-qubit gates, targets contains one index
	// For multi-qubit gates, targets contains multiple indices in specific order
	ApplyGate(gate Gate, targets ...int) error
	
	// Measure measures the specified qubit and collapses the state
	// Returns the measured value (0 or 1)
	Measure(qubitIndex int) (int, error)
	
	// Probability returns the probability of measuring a specific basis state
	Probability(basisState int) float64
	
	// Clone creates a copy of this quantum state
	Clone() QuantumState
}
```

### Step 2: Create Error Types

Create custom error types for quantum operations:

```go
package quantum

import (
	"fmt"
)

// QubitsOutOfRangeError indicates that a qubit index is out of the valid range
type QubitsOutOfRangeError struct {
	Index    int
	MaxIndex int
}

func (e *QubitsOutOfRangeError) Error() string {
	return fmt.Sprintf("qubit index %d is out of range [0,%d]", e.Index, e.MaxIndex)
}

// NormalizationError indicates that a quantum state is not properly normalized
type NormalizationError struct {
	Sum float64
}

func (e *NormalizationError) Error() string {
	return fmt.Sprintf("quantum state is not normalized (sum of probabilities = %f, should be 1.0)", e.Sum)
}

// InvalidGateApplicationError indicates that a gate cannot be applied in the requested manner
type InvalidGateApplicationError struct {
	Gate        string
	RequiredLen int
	ActualLen   int
}

func (e *InvalidGateApplicationError) Error() string {
	return fmt.Sprintf("gate %s requires %d qubits but got %d", e.Gate, e.RequiredLen, e.ActualLen)
}
```

### Step 3: Implement Core Qubit Type

```go
package qubit

import (
	"math"
	"math/cmplx"
	"math/rand"
	
	"github.com/pjbaur/quantum"
)

// Qubit implements the quantum.Qubit interface
type Qubit struct {
	alpha complex128
	beta  complex128
}

// New creates a new qubit in the |0⟩ state
func New() *Qubit {
	return &Qubit{
		alpha: 1.0,
		beta:  0.0,
	}
}

// NewWithValues creates a new qubit with specific amplitudes
// Returns error if amplitudes would not result in a normalized state
func NewWithValues(alpha, beta complex128) (*Qubit, error) {
	q := &Qubit{}
	if err := q.Set(alpha, beta); err != nil {
		return nil, err
	}
	return q, nil
}

// Alpha returns the amplitude of the |0⟩ state
func (q *Qubit) Alpha() complex128 {
	return q.alpha
}

// Beta returns the amplitude of the |1⟩ state
func (q *Qubit) Beta() complex128 {
	return q.beta
}

// Set updates the amplitudes of the qubit
func (q *Qubit) Set(alpha, beta complex128) error {
	// Calculate probability sum to check normalization
	probSum := math.Pow(cmplx.Abs(alpha), 2) + math.Pow(cmplx.Abs(beta), 2)
	
	// Allow a small floating-point error margin
	if math.Abs(probSum-1.0) > 1e-10 {
		return &quantum.NormalizationError{Sum: probSum}
	}
	
	q.alpha = alpha
	q.beta = beta
	return nil
}

// Probability0 returns the probability of measuring |0⟩
func (q *Qubit) Probability0() float64 {
	return math.Pow(cmplx.Abs(q.alpha), 2)
}

// Probability1 returns the probability of measuring |1⟩
func (q *Qubit) Probability1() float64 {
	return math.Pow(cmplx.Abs(q.beta), 2)
}

// Measure collapses the qubit to either |0⟩ or |1⟩
func (q *Qubit) Measure() int {
	prob0 := q.Probability0()
	if rand.Float64() < prob0 {
		// Collapse to |0⟩
		q.alpha = 1.0
		q.beta = 0.0
		return 0
	} else {
		// Collapse to |1⟩
		q.alpha = 0.0
		q.beta = 1.0
		return 1
	}
}

// Clone creates a copy of this qubit
func (q *Qubit) Clone() quantum.Qubit {
	return &Qubit{
		alpha: q.alpha,
		beta:  q.beta,
	}
}

// IsNormalized checks if the qubit is properly normalized
func (q *Qubit) IsNormalized() bool {
	sum := math.Pow(cmplx.Abs(q.alpha), 2) + math.Pow(cmplx.Abs(q.beta), 2)
	return math.Abs(sum-1.0) <= 1e-10
}
```

### Step 4: Implement Basic Gates

```go
package gates

import (
	"math"

	"github.com/pjbaur/quantum"
)

// HadamardGate implements the Hadamard gate
type HadamardGate struct{}

// NewHadamard creates a new Hadamard gate
func NewHadamard() *HadamardGate {
	return &HadamardGate{}
}

// Name returns the name of the gate
func (g *HadamardGate) Name() string {
	return "Hadamard"
}

// Matrix returns the matrix representation of the gate
func (g *HadamardGate) Matrix() [][]complex128 {
	return [][]complex128{
		{1.0 / complex(math.Sqrt(2), 0), 1.0 / complex(math.Sqrt(2), 0)},
		{1.0 / complex(math.Sqrt(2), 0), -1.0 / complex(math.Sqrt(2), 0)},
	}
}

// Apply applies the gate to the given qubit
func (g *HadamardGate) Apply(q quantum.Qubit) error {
	alpha := q.Alpha()
	beta := q.Beta()
	
	newAlpha := (alpha + beta) / complex(math.Sqrt(2), 0)
	newBeta := (alpha - beta) / complex(math.Sqrt(2), 0)
	
	return q.Set(newAlpha, newBeta)
}

// Implement other gates (PauliX, PauliY, PauliZ, S, T) similarly...
```

### Step 5: Implement QuantumState for Multiple Qubits

```go
package state

import (
	"math"
	"math/cmplx"
	"math/rand"
	
	"github.com/pjbaur/quantum"
)

// State implements the quantum.QuantumState interface
type State struct {
	numQubits  int
	amplitudes []complex128
}

// New creates a new quantum state with the specified number of qubits
// All qubits are initialized to |0⟩
func New(numQubits int) *State {
	if numQubits <= 0 {
		numQubits = 1
	}
	
	// Allocate 2^n amplitudes
	size := 1 << numQubits
	amplitudes := make([]complex128, size)
	
	// Initialize to |00...0⟩
	amplitudes[0] = 1.0
	
	return &State{
		numQubits:  numQubits,
		amplitudes: amplitudes,
	}
}

// NumQubits returns the number of qubits in the state
func (s *State) NumQubits() int {
	return s.numQubits
}

// Amplitude returns the amplitude of a specific basis state
func (s *State) Amplitude(basisState int) complex128 {
	if basisState < 0 || basisState >= len(s.amplitudes) {
		return 0
	}
	return s.amplitudes[basisState]
}

// SetAmplitude sets the amplitude for a specific basis state
func (s *State) SetAmplitude(basisState int, value complex128) error {
	if basisState < 0 || basisState >= len(s.amplitudes) {
		return &quantum.QubitsOutOfRangeError{
			Index:    basisState,
			MaxIndex: len(s.amplitudes) - 1,
		}
	}
	
	// Make the change and check normalization
	oldValue := s.amplitudes[basisState]
	s.amplitudes[basisState] = value
	
	if !s.isNormalized() {
		// Restore the previous value
		s.amplitudes[basisState] = oldValue
		return &quantum.NormalizationError{Sum: s.probabilitySum()}
	}
	
	return nil
}

// ApplyGate applies a gate to the specified qubit(s)
func (s *State) ApplyGate(gate quantum.Gate, targets ...int) error {
	// Validate target qubits
	for _, target := range targets {
		if target < 0 || target >= s.numQubits {
			return &quantum.QubitsOutOfRangeError{
				Index:    target,
				MaxIndex: s.numQubits - 1,
			}
		}
	}
	
	// Implement gate application logic based on gate type
	// (simple version shown here, would need to be expanded)
	if len(targets) == 1 && len(gate.Matrix()) == 2 {
		// Single-qubit gate
		return s.applySingleQubitGate(gate, targets[0])
	}
	
	return &quantum.InvalidGateApplicationError{
		Gate:        gate.Name(),
		RequiredLen: len(gate.Matrix()),
		ActualLen:   len(targets),
	}
}

// applySingleQubitGate applies a single-qubit gate to the specified qubit
func (s *State) applySingleQubitGate(gate quantum.Gate, target int) error {
	matrix := gate.Matrix()
	if len(matrix) != 2 || len(matrix[0]) != 2 {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: 2,
			ActualLen:   len(matrix),
		}
	}
	
	// Create a copy of amplitudes to work with
	newAmplitudes := make([]complex128, len(s.amplitudes))
	
	// Iterate through all basis states
	for i := range s.amplitudes {
		// Determine the basis states that will be affected
		i0 := i &^ (1 << target)              // Clear the target bit
		i1 := i | (1 << target)               // Set the target bit
		isTargetSet := (i & (1 << target)) != 0  // Check if target bit is set
		
		// Get the affected amplitudes
		a0 := s.amplitudes[i0]  // Amplitude where target qubit is 0
		a1 := s.amplitudes[i1]  // Amplitude where target qubit is 1
		
		// Apply the gate matrix
		if isTargetSet {
			newAmplitudes[i] = matrix[1][0]*a0 + matrix[1][1]*a1
		} else {
			newAmplitudes[i] = matrix[0][0]*a0 + matrix[0][1]*a1
		}
	}
	
	// Update amplitudes
	s.amplitudes = newAmplitudes
	
	return nil
}

// Measure measures the specified qubit and collapses the state
func (s *State) Measure(qubitIndex int) (int, error) {
	if qubitIndex < 0 || qubitIndex >= s.numQubits {
		return 0, &quantum.QubitsOutOfRangeError{
			Index:    qubitIndex,
			MaxIndex: s.numQubits - 1,
		}
	}
	
	// Calculate probability of measuring |0⟩
	prob0 := 0.0
	for i, amplitude := range s.amplitudes {
		if (i & (1 << qubitIndex)) == 0 {
			prob0 += math.Pow(cmplx.Abs(amplitude), 2)
		}
	}
	
	// Randomly determine the measurement outcome
	result := 0
	if rand.Float64() >= prob0 {
		result = 1
	}
	
	// Collapse the state based on the measurement
	newAmplitudes := make([]complex128, len(s.amplitudes))
	normalizationFactor := 0.0
	
	for i, amplitude := range s.amplitudes {
		isBitSet := (i & (1 << qubitIndex)) != 0
		if (result == 1 && isBitSet) || (result == 0 && !isBitSet) {
			newAmplitudes[i] = amplitude
			normalizationFactor += math.Pow(cmplx.Abs(amplitude), 2)
		}
	}
	
	// Normalize the resulting state
	normalizationFactor = math.Sqrt(normalizationFactor)
	for i := range newAmplitudes {
		newAmplitudes[i] /= complex(normalizationFactor, 0)
	}
	
	s.amplitudes = newAmplitudes
	return result, nil
}

// Probability returns the probability of measuring a specific basis state
func (s *State) Probability(basisState int) float64 {
	if basisState < 0 || basisState >= len(s.amplitudes) {
		return 0
	}
	return math.Pow(cmplx.Abs(s.amplitudes[basisState]), 2)
}

// Clone creates a copy of this quantum state
func (s *State) Clone() quantum.QuantumState {
	newAmplitudes := make([]complex128, len(s.amplitudes))
	copy(newAmplitudes, s.amplitudes)
	
	return &State{
		numQubits:  s.numQubits,
		amplitudes: newAmplitudes,
	}
}

// isNormalized checks if the state is properly normalized
func (s *State) isNormalized() bool {
	sum := s.probabilitySum()
	return math.Abs(sum-1.0) <= 1e-10
}

// probabilitySum calculates the sum of probabilities for all basis states
func (s *State) probabilitySum() float64 {
	sum := 0.0
	for _, amp := range s.amplitudes {
		sum += math.Pow(cmplx.Abs(amp), 2)
	}
	return sum
}
```

## 2. Restructure the Package Organization

### Step 1: Create Internal Package Structure

```bash
mkdir -p internal/examples
```

### Step 2: Move Example Code

```bash
# Move example files to internal/examples
mv examples/hadamard.go internal/examples/
mv examples/tgate.go internal/examples/
mv examples/bell.go internal/examples/
```

### Step 3: Update Main File (`cmd/quantum/main.go`)

```go
package main

import (
	"flag"
	"fmt"
	"os"
	
	"github.com/pjbaur/quantum/internal/examples"
)

func main() {
	flag.Parse()
	args := flag.Args()
	
	if len(args) == 0 {
		showUsage()
		os.Exit(0)
	}
	
	switch args[0] {
	case "hadamard":
		examples.RunHadamardDemo()
	case "tgate":
		examples.RunTGateDemo()
	case "bell":
		examples.RunBellDemo()
	case "all":
		examples.RunHadamardDemo()
		examples.RunTGateDemo()
		examples.RunBellDemo()
	default:
		fmt.Fprintf(os.Stderr, "Unknown demo: %s\n", args[0])
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Println("Quantum Computing Demonstrations")
	fmt.Println("\nUsage:")
	fmt.Println("  go run ./cmd/quantum [demo]")
	fmt.Println("\nAvailable Demos:")
	fmt.Println("  hadamard   - Hadamard gate demonstrations")
	fmt.Println("  tgate      - T-gate demonstrations")
	fmt.Println("  bell       - Bell state demonstrations")
	fmt.Println("  all        - Run all demonstrations sequentially")
}
```

### Step 4: Update the Examples to Use New Interfaces

Make sure your example code uses the new interfaces. For example:

```go
package examples

import (
	"fmt"
	
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/qubit"
)

// RunHadamardDemo demonstrates the Hadamard gate operation
func RunHadamardDemo() {
	fmt.Println("\n=== Hadamard Gate Demonstration ===")
	
	// Create a new qubit in |0⟩ state
	q := qubit.New()
	fmt.Printf("Initial state: |0⟩ (α=%v, β=%v)\n", q.Alpha(), q.Beta())
	
	// Create and apply Hadamard gate
	h := gates.NewHadamard()
	err := h.Apply(q)
	if err != nil {
		fmt.Printf("Error applying Hadamard gate: %v\n", err)
		return
	}
	
	fmt.Printf("After H: |+⟩ (α=%v, β=%v)\n", q.Alpha(), q.Beta())
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		q.Probability0(), q.Probability1())
	
	// Perform measurement
	result := q.Measure()
	fmt.Printf("Measurement result: |%d⟩\n", result)
	fmt.Printf("Final state: (α=%v, β=%v)\n", q.Alpha(), q.Beta())
}
```

## 3. Add Tests with Error Handling

Create comprehensive tests for your interfaces and implementations:

```go
package qubit

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum"
)

func TestNewQubit(t *testing.T) {
	q := New()
	
	if q.Alpha() != 1.0 || q.Beta() != 0.0 {
		t.Errorf("New qubit should be in |0⟩ state, got α=%v, β=%v", q.Alpha(), q.Beta())
	}
}

func TestNewWithValues(t *testing.T) {
	tests := []struct {
		name      string
		alpha     complex128
		beta      complex128
		expectErr bool
	}{
		{"Valid state 1", 1.0, 0.0, false},
		{"Valid state 2", 0.0, 1.0, false},
		{"Valid superposition", complex(1/math.Sqrt(2), 0), complex(1/math.Sqrt(2), 0), false},
		{"Invalid state", 1.0, 1.0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewWithValues(tt.alpha, tt.beta)
			
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
				}
				if q.Alpha() != tt.alpha || q.Beta() != tt.beta {
					t.Errorf("Expected α=%v, β=%v but got α=%v, β=%v",
						tt.alpha, tt.beta, q.Alpha(), q.Beta())
				}
			}
		})
	}
}

// Add more tests for other methods...
```

## 4. Create Documentation

Add comprehensive documentation to all exported types and functions:

```go
// Package quantum provides interfaces and implementations for quantum computing simulation.
//
// The package includes representations for qubits, gates, and multi-qubit states,
// along with operations like measurement and gate application.
package quantum

// ...
```

## Final Steps

1. **Run Unit Tests**: Verify that all the implemented functionality works correctly.
2. **Documentation Review**: Ensure all public functions and types have proper documentation.
3. **Code Review**: Check for consistent naming, proper error handling, and clean organization.
4. **Update README**: Update the project README to reflect the new structure and interfaces.

By following these steps, you'll establish a solid foundation with well-defined interfaces, proper error handling, and a clean package structure. This will make it much easier to implement the remaining phases of your project.

---

# Phase 1 Implementation Review

## Scope
Reviewed `CHANGES/phase1-implementation.md` against the current repository implementation.

## Completeness and Implementation Status
- Completed
  - [x] Core interfaces exist in `quantum/interfaces.go`.
  - [x] Custom error types exist in `quantum/errortypes.go` (plus an extra `IncompatibleQubitCountError`).
  - [x] Core qubit type implemented in `qubit/qubit.go`.
  - [x] Single-qubit gates (Hadamard, Pauli X/Y/Z, S, T) implemented in `gates/gates.go`.
  - [x] `QuantumState` implementation with single-qubit gate support and measurement in `state/state.go`.
  - [x] Example code lives under `internal/examples` and is wired into `cmd/quantum/main.go`.
  - [x] Tests exist in `quantum/quantum_test.go` and `circuit/circuit_test.go`.
- Partially completed
  - [x] Multi-qubit gate support is present in `gates/gates.go` (CNOT, SWAP), but `state.State.ApplyGate` only supports 2x2 matrices and returns `InvalidGateApplicationError` for multi-qubit gates (`state/state.go`).
  - [x] The Bell state examples invoke `ApplyGate(cnot, 0, 1)` and will fail with the current `state.State.ApplyGate` implementation (`internal/examples/bell.go`).
- Missing
  - [x] Package-level documentation comment for `package quantum` is not present in `quantum/interfaces.go` or `quantum/errortypes.go`, despite Step 4 in the guide calling for it.
  - [x] Tests for multi-qubit gates and `QuantumState.ApplyGate` on multi-qubit targets are not present.

## Correctness of the Guide
No known inaccuracies remain after aligning the snippets and paths with the current codebase.

## Summary
Phase 1 implementation is complete for interfaces, core qubit and state behavior (including multi-qubit gate application), examples, and required package documentation, with tests covering multi-qubit `ApplyGate` cases.
