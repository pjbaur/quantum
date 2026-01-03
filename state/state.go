package state

import (
	"math"
	"math/cmplx"
	"math/rand"

	"github.com/pjbaur/quantum/quantum"
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

	if len(targets) == 2 && len(gate.Matrix()) == 4 {
		if targets[0] == targets[1] {
			return &quantum.InvalidGateApplicationError{
				Gate:        gate.Name(),
				RequiredLen: 2,
				ActualLen:   1,
			}
		}
		return s.applyTwoQubitGate(gate, targets[0], targets[1])
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
		i0 := i &^ (1 << target)                // Clear the target bit
		i1 := i | (1 << target)                 // Set the target bit
		isTargetSet := (i & (1 << target)) != 0 // Check if target bit is set

		// Get the affected amplitudes
		a0 := s.amplitudes[i0] // Amplitude where target qubit is 0
		a1 := s.amplitudes[i1] // Amplitude where target qubit is 1

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

// applyTwoQubitGate applies a two-qubit gate to the specified qubits
func (s *State) applyTwoQubitGate(gate quantum.Gate, target0, target1 int) error {
	matrix := gate.Matrix()
	if len(matrix) != 4 {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: 4,
			ActualLen:   len(matrix),
		}
	}
	for i := range matrix {
		if len(matrix[i]) != 4 {
			return &quantum.InvalidGateApplicationError{
				Gate:        gate.Name(),
				RequiredLen: 4,
				ActualLen:   len(matrix[i]),
			}
		}
	}

	newAmplitudes := make([]complex128, len(s.amplitudes))
	mask0 := 1 << target0
	mask1 := 1 << target1

	for base := 0; base < len(s.amplitudes); base++ {
		if (base&mask0) != 0 || (base&mask1) != 0 {
			continue
		}

		i00 := base
		i01 := base | mask1
		i10 := base | mask0
		i11 := base | mask0 | mask1

		a00 := s.amplitudes[i00]
		a01 := s.amplitudes[i01]
		a10 := s.amplitudes[i10]
		a11 := s.amplitudes[i11]

		newAmplitudes[i00] = matrix[0][0]*a00 + matrix[0][1]*a01 + matrix[0][2]*a10 + matrix[0][3]*a11
		newAmplitudes[i01] = matrix[1][0]*a00 + matrix[1][1]*a01 + matrix[1][2]*a10 + matrix[1][3]*a11
		newAmplitudes[i10] = matrix[2][0]*a00 + matrix[2][1]*a01 + matrix[2][2]*a10 + matrix[2][3]*a11
		newAmplitudes[i11] = matrix[3][0]*a00 + matrix[3][1]*a01 + matrix[3][2]*a10 + matrix[3][3]*a11
	}

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
