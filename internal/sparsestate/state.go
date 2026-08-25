package sparsestate

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"

	"github.com/pjbaur/quantum/internal/backendmath"
	"github.com/pjbaur/quantum/quantum"
)

// pruneEpsilon is the amplitude magnitude at or below which this backend
// drops a basis state from the map. Its square is the probability of the
// faintest amplitude the backend keeps, which is where backendmath sets the
// floor for a measurement branch that holds nothing.
const pruneEpsilon = 1e-12

// State is a sparse quantum state representation that stores only non-zero amplitudes.
type State struct {
	numQubits  int
	amplitudes map[int]complex128
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

// New creates a new sparse quantum state with the specified number of qubits.
// All qubits are initialized to |0⟩.
// Returns InvalidQubitCountError if numQubits <= 0.
func New(numQubits int) (*State, error) {
	if numQubits <= 0 {
		return nil, &quantum.InvalidQubitCountError{
			Requested: numQubits,
			Reason:    "must be positive",
		}
	}

	return &State{
		numQubits:  numQubits,
		amplitudes: map[int]complex128{0: 1},
	}, nil
}

// NumQubits returns the number of qubits in the state.
func (s *State) NumQubits() int {
	return s.numQubits
}

// Amplitude returns the amplitude of a specific basis state.
func (s *State) Amplitude(basisState int) complex128 {
	if basisState < 0 || basisState >= (1<<s.numQubits) {
		return 0
	}
	return s.amplitudes[basisState]
}

// SetAmplitude sets the amplitude for a specific basis state.
// A NaN or infinite value is rejected outright; otherwise the write is
// rolled back unless the state stays normalized.
func (s *State) SetAmplitude(basisState int, value complex128) error {
	if basisState < 0 || basisState >= (1<<s.numQubits) {
		return &quantum.QubitsOutOfRangeError{
			Index:    basisState,
			MaxIndex: (1 << s.numQubits) - 1,
		}
	}

	// A non-finite amplitude has to be caught before the normalization
	// check, which cannot see it: NaN makes the probability sum NaN, and
	// NaN fails every comparison against the tolerance.
	if !quantum.IsFiniteAmplitude(value) {
		return &quantum.NonFiniteAmplitudeError{BasisState: basisState, Value: value}
	}

	oldValue, had := s.amplitudes[basisState]
	s.setAmplitudeUnsafe(basisState, value)

	if !s.isNormalized() {
		// Capture the attempted sum before rollback
		attemptedSum := s.probabilitySum()
		if had {
			s.setAmplitudeUnsafe(basisState, oldValue)
		} else {
			delete(s.amplitudes, basisState)
		}
		return &quantum.NormalizationError{
			AttemptedSum: attemptedSum,
			CurrentSum:   s.probabilitySum(),
		}
	}

	return nil
}

// ApplyGate applies a gate to the specified qubit(s).
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

	if err := backendmath.ValidateTargets(targets, s.numQubits); err != nil {
		return err
	}

	if requiredQubits == 1 {
		return s.applySingleQubitGate(gate, targets[0])
	}

	if requiredQubits == 2 {
		return s.applyTwoQubitGate(gate, targets)
	}

	// 3+ qubit gates are not supported by the sparse backend
	return &quantum.UnsupportedOperationError{
		Operation:   fmt.Sprintf("%d-qubit gate application", requiredQubits),
		Backend:     "sparse",
		Alternative: "dense state backend (state.State)",
	}
}

func (s *State) applySingleQubitGate(gate quantum.Gate, target int) error {
	matrix := gate.Matrix()
	if len(matrix) != 2 || len(matrix[0]) != 2 {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: 2,
			ActualLen:   len(matrix),
		}
	}

	targetMask := 1 << target
	bases := make(map[int]struct{}, len(s.amplitudes))
	for index := range s.amplitudes {
		bases[index&^targetMask] = struct{}{}
	}

	newAmplitudes := make(map[int]complex128, len(s.amplitudes))
	for base := range bases {
		a0 := s.amplitudes[base]
		a1 := s.amplitudes[base|targetMask]

		new0 := matrix[0][0]*a0 + matrix[0][1]*a1
		new1 := matrix[1][0]*a0 + matrix[1][1]*a1

		if !isNearZero(new0) {
			newAmplitudes[base] = new0
		}
		if !isNearZero(new1) {
			newAmplitudes[base|targetMask] = new1
		}
	}

	s.amplitudes = newAmplitudes
	return nil
}

func (s *State) applyTwoQubitGate(gate quantum.Gate, targets []int) error {
	matrix := gate.Matrix()
	if len(matrix) != 4 || len(matrix[0]) != 4 {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: 4,
			ActualLen:   len(matrix),
		}
	}

	if isCanonicalCNOT(matrix) {
		return s.applyCNOT(targets[0], targets[1])
	}

	comboMasks := make([]int, 1<<len(targets))
	targetMask := backendmath.ComboMasks(comboMasks, targets)
	bases := make(map[int]struct{}, len(s.amplitudes))
	for index := range s.amplitudes {
		bases[index&^targetMask] = struct{}{}
	}

	inputs := make([]complex128, 4)
	outputs := make([]complex128, 4)
	newAmplitudes := make(map[int]complex128, len(s.amplitudes))

	for base := range bases {
		for combo := 0; combo < 4; combo++ {
			inputs[combo] = s.amplitudes[base|comboMasks[combo]]
		}

		for row := 0; row < 4; row++ {
			sum := complex(0, 0)
			for col := 0; col < 4; col++ {
				sum += matrix[row][col] * inputs[col]
			}
			outputs[row] = sum
		}

		for combo := 0; combo < 4; combo++ {
			value := outputs[combo]
			if !isNearZero(value) {
				newAmplitudes[base|comboMasks[combo]] = value
			}
		}
	}

	s.amplitudes = newAmplitudes
	return nil
}

func (s *State) applyCNOT(control, target int) error {
	controlMask := 1 << control
	targetMask := 1 << target

	newAmplitudes := make(map[int]complex128, len(s.amplitudes))
	for index, amplitude := range s.amplitudes {
		if index&controlMask != 0 {
			index ^= targetMask
		}
		if !isNearZero(amplitude) {
			newAmplitudes[index] = amplitude
		}
	}

	s.amplitudes = newAmplitudes
	return nil
}

var canonicalCNOT = [4][4]complex128{
	{1, 0, 0, 0},
	{0, 1, 0, 0},
	{0, 0, 0, 1},
	{0, 0, 1, 0},
}

// isCanonicalCNOT reports whether matrix is exactly the standard CNOT matrix.
// The CNOT fast path is selected by matrix content rather than gate name, so
// a user-defined gate named "CNOT" with different semantics is not hijacked.
func isCanonicalCNOT(matrix [][]complex128) bool {
	if len(matrix) != 4 {
		return false
	}
	for row := range matrix {
		if len(matrix[row]) != 4 {
			return false
		}
		for col, value := range matrix[row] {
			if value != canonicalCNOT[row][col] {
				return false
			}
		}
	}
	return true
}

// Measure measures the specified qubit and collapses the state.
func (s *State) Measure(qubitIndex int) (int, error) {
	if qubitIndex < 0 || qubitIndex >= s.numQubits {
		return 0, &quantum.QubitsOutOfRangeError{
			Index:    qubitIndex,
			MaxIndex: s.numQubits - 1,
		}
	}

	prob0, prob1 := 0.0, 0.0
	mask := 1 << qubitIndex
	for index, amplitude := range s.amplitudes {
		if index&mask == 0 {
			prob0 += quantum.Probability(amplitude)
		} else {
			prob1 += quantum.Probability(amplitude)
		}
	}

	// The draw can land on a branch holding no probability at all, which
	// this backend reaches by pruning the faint amplitude away; the guard
	// inside PlanCollapse is what keeps the collapse below from dividing by
	// a zero normalization factor and emptying the map.
	collapse, err := backendmath.PlanCollapse(qubitIndex, s.randFloat64(), prob0, prob1)
	if err != nil {
		return 0, err
	}

	newAmplitudes := make(map[int]complex128, len(s.amplitudes))
	for index, amplitude := range s.amplitudes {
		if collapse.Keeps(index) {
			newAmplitudes[index] = collapse.Renormalize(amplitude)
		}
	}

	s.amplitudes = newAmplitudes
	return collapse.Outcome, nil
}

// Probability returns the probability of measuring a specific basis state.
func (s *State) Probability(basisState int) float64 {
	amp := s.Amplitude(basisState)
	return quantum.Probability(amp)
}

// Clone creates a copy of this quantum state.
func (s *State) Clone() quantum.QuantumState {
	amplitudes := make(map[int]complex128, len(s.amplitudes))
	for index, amplitude := range s.amplitudes {
		amplitudes[index] = amplitude
	}

	// The clone shares the randomness source (if any), so seeded
	// pipelines stay deterministic across clones.
	return &State{
		numQubits:  s.numQubits,
		amplitudes: amplitudes,
		randSource: s.randSource,
	}
}

func (s *State) isNormalized() bool {
	sum := s.probabilitySum()
	return math.Abs(sum-1.0) <= 1e-10
}

func (s *State) probabilitySum() float64 {
	sum := 0.0
	for _, amp := range s.amplitudes {
		sum += quantum.Probability(amp)
	}
	return sum
}

func (s *State) setAmplitudeUnsafe(basisState int, value complex128) {
	if isNearZero(value) {
		delete(s.amplitudes, basisState)
		return
	}
	s.amplitudes[basisState] = value
}

func isNearZero(value complex128) bool {
	return cmplx.Abs(value) <= pruneEpsilon
}

// SupportsGateQubits returns whether this backend can apply gates
// operating on the specified number of qubits.
// The sparse backend supports 1- and 2-qubit gates only.
func (s *State) SupportsGateQubits(qubitCount int) bool {
	return qubitCount >= 1 && qubitCount <= 2
}

// MaxGateQubits returns the maximum number of qubits a gate can operate on.
// The sparse backend supports at most 2-qubit gates.
func (s *State) MaxGateQubits() int {
	return 2
}
