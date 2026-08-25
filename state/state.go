package state

import (
	"errors"
	"fmt"
	"math"
	"math/rand"

	"github.com/pjbaur/quantum/quantum"
)

// minBranchProbability is the probability below which a measurement branch
// is treated as empty rather than collapsed onto. It is far below any
// probability an honest draw can single out: selecting a branch this faint
// needs prob0 to sit within 1e-24 of 1, but float64 resolves values near 1
// only to ~1e-16, so no legitimate outcome is suppressed by the floor. It
// matches the square of the sparse backend's prune threshold, so both
// backends treat the same faint branches as carrying no amplitude.
const minBranchProbability = 1e-24

// State implements the quantum.QuantumState interface
type State struct {
	numQubits  int
	amplitudes []complex128
	scratch    []complex128
	comboMasks []int
	inputs     []complex128
	outputs    []complex128
	randSource quantum.RandomSource
}

// SetRandSource sets the randomness source used by Measure, enabling
// reproducible measurements from a seeded generator. A nil source
// restores the default (the global math/rand source).
func (s *State) SetRandSource(src quantum.RandomSource) {
	s.randSource = src
}

func (s *State) randFloat64() float64 {
	if s.randSource != nil {
		return s.randSource.Float64()
	}
	return rand.Float64()
}

// New creates a new quantum state with the specified number of qubits.
// All qubits are initialized to |0⟩.
// Returns InvalidQubitCountError if numQubits <= 0.
func New(numQubits int) (*State, error) {
	if numQubits <= 0 {
		return nil, &quantum.InvalidQubitCountError{
			Requested: numQubits,
			Reason:    "must be positive",
		}
	}

	// Allocate 2^n amplitudes
	size := 1 << numQubits
	amplitudes := make([]complex128, size)

	// Initialize to |00...0⟩
	amplitudes[0] = 1.0

	return &State{
		numQubits:  numQubits,
		amplitudes: amplitudes,
	}, nil
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

// SetAmplitude sets the amplitude for a specific basis state.
// A NaN or infinite value is rejected outright; otherwise the write is
// rolled back unless the state stays normalized.
func (s *State) SetAmplitude(basisState int, value complex128) error {
	if basisState < 0 || basisState >= len(s.amplitudes) {
		return &quantum.QubitsOutOfRangeError{
			Index:    basisState,
			MaxIndex: len(s.amplitudes) - 1,
		}
	}

	if !quantum.IsFiniteAmplitude(value) {
		return &quantum.NonFiniteAmplitudeError{BasisState: basisState, Value: value}
	}

	// Make the change and check normalization
	oldValue := s.amplitudes[basisState]
	s.amplitudes[basisState] = value

	if !s.isNormalized() {
		// Capture the attempted sum before rollback
		attemptedSum := s.probabilitySum()
		// Restore the previous value
		s.amplitudes[basisState] = oldValue
		return &quantum.NormalizationError{
			AttemptedSum: attemptedSum,
			CurrentSum:   s.probabilitySum(),
		}
	}

	return nil
}

// SetAmplitudes sets all amplitudes at once with a single normalization check.
// This is more efficient than calling SetAmplitude repeatedly when updating
// multiple amplitudes, as it only validates normalization once at the end.
// The values slice must have exactly 2^numQubits elements, and every value
// must be finite.
func (s *State) SetAmplitudes(values []complex128) error {
	if len(values) != len(s.amplitudes) {
		return fmt.Errorf("values slice length %d does not match state size %d", len(values), len(s.amplitudes))
	}

	// Check normalization of new values. Non-finite amplitudes have to be
	// caught here rather than by the sum: a NaN amplitude makes the sum NaN,
	// and NaN fails every comparison, so the tolerance test below would let
	// it through as normalized.
	sum := 0.0
	for i, v := range values {
		if !quantum.IsFiniteAmplitude(v) {
			return &quantum.NonFiniteAmplitudeError{BasisState: i, Value: v}
		}
		sum += quantum.Probability(v)
	}
	if math.Abs(sum-1.0) > 1e-10 {
		return &quantum.NormalizationError{
			AttemptedSum: sum,
			CurrentSum:   s.probabilitySum(),
		}
	}

	// Copy values
	copy(s.amplitudes, values)
	return nil
}

// ApplyGate applies a gate to the specified qubit(s).
// Qubit indices are little-endian (qubit 0 is the least-significant bit).
// Gate matrix ordering follows the targets slice, with targets[0] as the
// most-significant bit in the gate's basis ordering.
func (s *State) ApplyGate(gate quantum.Gate, targets ...int) error {
	requiredQubits, err := quantum.GateQubitCount(gate)
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
	if s.numQubits <= 64 {
		var seen uint64
		for _, target := range targets {
			if target < 0 || target >= s.numQubits {
				return &quantum.QubitsOutOfRangeError{
					Index:    target,
					MaxIndex: s.numQubits - 1,
				}
			}
			bit := uint64(1) << target
			if seen&bit != 0 {
				return errors.New("targets must be unique")
			}
			seen |= bit
		}
	} else {
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

	newAmplitudes := s.ensureScratch()

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
	s.amplitudes, s.scratch = newAmplitudes, s.amplitudes

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

	comboMasks := s.comboMasks
	if cap(comboMasks) < comboCount {
		comboMasks = make([]int, comboCount)
	}
	comboMasks = comboMasks[:comboCount]
	s.comboMasks = comboMasks
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

	inputs := s.inputs
	if cap(inputs) < comboCount {
		inputs = make([]complex128, comboCount)
	}
	inputs = inputs[:comboCount]
	s.inputs = inputs

	outputs := s.outputs
	if cap(outputs) < comboCount {
		outputs = make([]complex128, comboCount)
	}
	outputs = outputs[:comboCount]
	s.outputs = outputs

	newAmplitudes := s.ensureScratch()

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

	s.amplitudes, s.scratch = newAmplitudes, s.amplitudes
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

	// Calculate the probability of each measurement outcome
	prob0, prob1 := 0.0, 0.0
	for i, amplitude := range s.amplitudes {
		if (i & (1 << qubitIndex)) == 0 {
			prob0 += quantum.Probability(amplitude)
		} else {
			prob1 += quantum.Probability(amplitude)
		}
	}

	// Randomly determine the measurement outcome
	result := 0
	if s.randFloat64() >= prob0 {
		result = 1
	}

	// The draw can land on a branch that holds no probability at all.
	// Round-off in prob0 leaves a sliver of the [0,1) draw range pointing
	// at an outcome the state has nothing in, and amplitudes small enough
	// to square to zero still pass the normalization check. Collapsing
	// there would divide by a zero normalization factor and fill the state
	// with NaN, so measure the other outcome instead: it holds essentially
	// all of the probability, which is what the draw would have selected
	// had prob0 been exact. Both branches empty means the state is not
	// normalized and there is nothing to collapse onto.
	branchProb, otherProb := prob0, prob1
	if result == 1 {
		branchProb, otherProb = prob1, prob0
	}
	if branchProb < minBranchProbability {
		if otherProb < minBranchProbability {
			return 0, fmt.Errorf("measuring qubit %d: neither outcome has any probability (sum %g); state is not normalized",
				qubitIndex, prob0+prob1)
		}
		result, branchProb = 1-result, otherProb
	}

	// Collapse the state onto the measured outcome, normalizing as we go
	newAmplitudes := s.ensureScratch()
	normalizationFactor := complex(math.Sqrt(branchProb), 0)

	for i, amplitude := range s.amplitudes {
		isBitSet := (i & (1 << qubitIndex)) != 0
		if (result == 1 && isBitSet) || (result == 0 && !isBitSet) {
			newAmplitudes[i] = amplitude / normalizationFactor
		} else {
			newAmplitudes[i] = 0
		}
	}

	s.amplitudes, s.scratch = newAmplitudes, s.amplitudes
	return result, nil
}

// Probability returns the probability of measuring a specific basis state
func (s *State) Probability(basisState int) float64 {
	if basisState < 0 || basisState >= len(s.amplitudes) {
		return 0
	}
	return quantum.Probability(s.amplitudes[basisState])
}

// Clone creates a copy of this quantum state
func (s *State) Clone() quantum.QuantumState {
	newAmplitudes := make([]complex128, len(s.amplitudes))
	copy(newAmplitudes, s.amplitudes)

	// The clone shares the randomness source (if any), so seeded
	// pipelines stay deterministic across clones.
	return &State{
		numQubits:  s.numQubits,
		amplitudes: newAmplitudes,
		randSource: s.randSource,
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
		sum += quantum.Probability(amp)
	}
	return sum
}

func (s *State) ensureScratch() []complex128 {
	if len(s.scratch) != len(s.amplitudes) {
		s.scratch = make([]complex128, len(s.amplitudes))
	}
	return s.scratch
}

// SupportsGateQubits returns whether this backend can apply gates
// operating on the specified number of qubits.
// The dense backend supports all gate sizes (memory permitting).
func (s *State) SupportsGateQubits(qubitCount int) bool {
	return qubitCount >= 1
}

// MaxGateQubits returns the maximum number of qubits a gate can operate on.
// Returns 0 to indicate no limit (dense backend supports all gate sizes).
func (s *State) MaxGateQubits() int {
	return 0
}
