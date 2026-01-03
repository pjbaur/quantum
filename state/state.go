package state

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
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

// ApplyGate applies a gate to the specified qubit(s).
// Qubit indices are little-endian (qubit 0 is the least-significant bit).
// Gate matrix ordering follows the targets slice, with targets[0] as the
// most-significant bit in the gate's basis ordering.
func (s *State) ApplyGate(gate quantum.Gate, targets ...int) error {
	requiredQubits, err := gateQubitCount(gate)
	if err != nil {
		return err
	}

	if len(targets) != requiredQubits {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: requiredQubits,
			ActualLen:   len(targets),
		}
	}

	// Validate target qubits.
	seen := make(map[int]struct{}, len(targets))
	for _, target := range targets {
		if target < 0 || target >= s.numQubits {
			return &quantum.QubitsOutOfRangeError{
				Index:    target,
				MaxIndex: s.numQubits - 1,
			}
		}
		if _, exists := seen[target]; exists {
			return errors.New("targets must be unique")
		}
		seen[target] = struct{}{}
	}

	if len(targets) == 1 && requiredQubits == 1 {
		// Single-qubit gate
		return s.applySingleQubitGate(gate, targets[0])
	}

	if requiredQubits > 1 {
		return s.applyMultiQubitGate(gate, targets)
	}

	return &quantum.InvalidGateApplicationError{
		Gate:        gate.Name(),
		RequiredLen: requiredQubits,
		ActualLen:   len(targets),
	}
}

func gateQubitCount(gate quantum.Gate) (int, error) {
	matrix := gate.Matrix()
	if len(matrix) == 0 {
		return 0, fmt.Errorf("gate %s has empty matrix", gate.Name())
	}

	size := len(matrix)
	for _, row := range matrix {
		if len(row) != size {
			return 0, fmt.Errorf("gate %s matrix must be square", gate.Name())
		}
	}

	if size&(size-1) != 0 {
		return 0, fmt.Errorf("gate %s matrix size %d is not a power of two", gate.Name(), size)
	}

	return bits.Len(uint(size)) - 1, nil
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

// applyMultiQubitGate applies a multi-qubit gate to the specified qubits.
func (s *State) applyMultiQubitGate(gate quantum.Gate, targets []int) error {
	matrix := gate.Matrix()
	targetCount := len(targets)
	comboCount := 1 << targetCount

	targetMask := 0
	for _, target := range targets {
		targetMask |= 1 << target
	}

	comboMasks := make([]int, comboCount)
	for combo := 0; combo < comboCount; combo++ {
		mask := 0
		for i, target := range targets {
			shift := targetCount - 1 - i
			if (combo>>shift)&1 == 1 {
				mask |= 1 << target
			}
		}
		comboMasks[combo] = mask
	}

	inputs := make([]complex128, comboCount)
	outputs := make([]complex128, comboCount)
	newAmplitudes := make([]complex128, len(s.amplitudes))

	for base := 0; base < len(s.amplitudes); base++ {
		if base&targetMask != 0 {
			continue
		}

		for combo := 0; combo < comboCount; combo++ {
			inputs[combo] = s.amplitudes[base|comboMasks[combo]]
		}

		for row := 0; row < comboCount; row++ {
			sum := complex(0, 0)
			for col := 0; col < comboCount; col++ {
				sum += matrix[row][col] * inputs[col]
			}
			outputs[row] = sum
		}

		for combo := 0; combo < comboCount; combo++ {
			newAmplitudes[base|comboMasks[combo]] = outputs[combo]
		}
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
