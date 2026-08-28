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

// Reset restores the state to |0…0⟩ in place, satisfying
// quantum.Resetter. The amplitude map is cleared rather than reallocated,
// and an injected randomness source survives.
func (s *State) Reset() {
	clear(s.amplitudes)
	s.amplitudes[0] = 1
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

// SetAmplitudes sets all amplitudes at once with a single normalization check,
// satisfying quantum.BulkAmplitudeSetter. The values slice must have exactly
// 2^numQubits elements, every value must be finite, and the probabilities must
// sum to 1 within the same 1e-10 tolerance the dense backend applies; the three
// checks run in that order, so a vector that is wrong in more than one way
// reports the same failure the dense backend would.
//
// The one behavioral difference from the dense backend is this one's storage
// rule, not its validation: an amplitude at or below pruneEpsilon is dropped
// rather than stored, so it reads back as exactly zero. The probability such a
// value carries is at most pruneEpsilon², fourteen orders of magnitude below
// the normalization tolerance.
func (s *State) SetAmplitudes(values []complex128) error {
	size := 1 << s.numQubits
	if len(values) != size {
		return fmt.Errorf("values slice length %d does not match state size %d", len(values), size)
	}

	// Non-finite amplitudes have to be caught here rather than by the sum: a
	// NaN amplitude makes the sum NaN, and NaN fails every comparison, so the
	// tolerance test below would let it through as normalized.
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

	// Rebuild rather than overwrite: entries the old vector held and the new
	// one leaves at zero must not survive as stale map keys.
	amplitudes := make(map[int]complex128, len(s.amplitudes))
	for i, v := range values {
		if isNearZero(v) {
			continue
		}
		amplitudes[i] = v
	}
	s.amplitudes = amplitudes
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

	return s.applyMultiQubitGate(gate, targets)
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

// applyMultiQubitGate applies a k-qubit gate (k >= 2) by grouping the
// non-zero amplitudes into bases — registers where every target qubit is
// zero — and multiplying each group of 2^k amplitudes by the gate matrix.
// Cost is O(nonzero * 4^k), so wide gates on sparse states trade the
// matrix's density for the state's sparsity.
func (s *State) applyMultiQubitGate(gate quantum.Gate, targets []int) error {
	k := len(targets)
	size := 1 << k

	matrix := gate.Matrix()
	if len(matrix) != size || len(matrix[0]) != size {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: size,
			ActualLen:   len(matrix),
		}
	}

	// CNOT keeps its permutation fast path: no matrix multiply at all.
	if k == 2 && isCanonicalCNOT(matrix) {
		return s.applyCNOT(targets[0], targets[1])
	}

	comboMasks := make([]int, size)
	targetMask := backendmath.ComboMasks(comboMasks, targets)
	bases := make(map[int]struct{}, len(s.amplitudes))
	for index := range s.amplitudes {
		bases[index&^targetMask] = struct{}{}
	}

	inputs := make([]complex128, size)
	outputs := make([]complex128, size)
	newAmplitudes := make(map[int]complex128, len(s.amplitudes))

	for base := range bases {
		for combo := 0; combo < size; combo++ {
			inputs[combo] = s.amplitudes[base|comboMasks[combo]]
		}

		for row := 0; row < size; row++ {
			sum := complex(0, 0)
			for col := 0; col < size; col++ {
				sum += matrix[row][col] * inputs[col]
			}
			outputs[row] = sum
		}

		for combo := 0; combo < size; combo++ {
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
// operating on the specified number of qubits. The sparse backend applies
// any k-qubit gate generically (k >= 1); see applyMultiQubitGate for the
// cost tradeoff on wide gates.
func (s *State) SupportsGateQubits(qubitCount int) bool {
	return qubitCount >= 1
}

// MaxGateQubits returns the maximum number of qubits a gate can operate on.
// Zero means no limit; the sparse backend applies gates of any width.
func (s *State) MaxGateQubits() int {
	return 0
}
