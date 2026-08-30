// Package density implements a density-matrix backend. Matrix satisfies
// quantum.QuantumState (see docs/adr/0009-density-backend-quantumstate.md),
// so circuits execute on it interchangeably with the state-vector
// backends, and adds what only a density matrix can represent: Kraus
// noise channels (depolarizing, dephasing, amplitude damping),
// trace/purity, reduced single-qubit Bloch vectors, and FromState for
// bridging in a pure state. Mixed states have no amplitude vector, so
// Amplitude returns NaN once noise mixes the state and SetAmplitude is
// refused; cost is O(4ⁿ) in memory and O(4ⁿ·4ᵏ) per k-qubit gate.
package density

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
	"reflect"

	"github.com/pjbaur/quantum/internal/backendmath"
	"github.com/pjbaur/quantum/quantum"
)

// Matrix represents a density matrix for an n-qubit system.
// This type is internal to the module to keep the API surface focused.
type Matrix struct {
	numQubits  int
	dim        int
	data       []complex128
	temp       []complex128
	scratch    []complex128
	work       []complex128
	randSource quantum.RandomSource
}

// Matrix implements quantum.QuantumState (see ADR-0009) and
// quantum.BackendCapabilities. It deliberately does not implement
// quantum.BulkAmplitudeSetter (a mixed state has no amplitude vector to
// replace, so QFT refuses it) or quantum.Resetter (no consumer).
var (
	_ quantum.QuantumState        = (*Matrix)(nil)
	_ quantum.BackendCapabilities = (*Matrix)(nil)
)

// New creates a density matrix initialized to |00...0⟩⟨00...0|.
// Returns InvalidQubitCountError if numQubits <= 0.
func New(numQubits int) (*Matrix, error) {
	if numQubits <= 0 {
		return nil, &quantum.InvalidQubitCountError{
			Requested: numQubits,
			Reason:    "must be positive",
		}
	}

	dim := 1 << numQubits
	data := make([]complex128, dim*dim)
	data[0] = 1

	return &Matrix{
		numQubits: numQubits,
		dim:       dim,
		data:      data,
	}, nil
}

// FromState builds the density matrix ρ = |ψ⟩⟨ψ| of the pure state s.
// Amplitudes are read through the quantum.QuantumState accessor, so every
// backend works without this package knowing the concrete type.
//
// The result holds 4ⁿ elements for an n-qubit state, quadratically more than
// the state vector it was built from, so this is meant for the small states
// diagnostics and visualization inspect.
//
// Returns an error if s is nil or a typed nil (e.g. a nil *state.State
// boxed in the interface); either way the error is untyped, a deliberate
// choice consistent with nil-argument handling elsewhere in the module
// (e.g. circuit.Execute).
// Returns InvalidQubitCountError if s reports a non-positive qubit count.
// Returns NonFiniteAmplitudeError if any amplitude is NaN or infinite; in
// particular a mixed density matrix, whose Amplitude method reports NaN,
// is refused rather than copied into a NaN-filled outer product.
func FromState(s quantum.QuantumState) (*Matrix, error) {
	if isNilState(s) {
		return nil, errors.New("state must not be nil")
	}

	m, err := New(s.NumQubits())
	if err != nil {
		return nil, err
	}

	// Read each amplitude once: the outer product touches every pair.
	// A non-finite amplitude (NaN or Inf) is rejected here rather than
	// spread across the whole matrix — notably, a mixed density source
	// reports NaN amplitudes because it has no amplitude vector.
	amplitudes := make([]complex128, m.dim)
	for i := range amplitudes {
		amplitudes[i] = s.Amplitude(i)
		if !quantum.IsFiniteAmplitude(amplitudes[i]) {
			return nil, &quantum.NonFiniteAmplitudeError{
				BasisState: i,
				Value:      amplitudes[i],
			}
		}
	}

	for i := 0; i < m.dim; i++ {
		row := i * m.dim
		for j := 0; j < m.dim; j++ {
			m.data[row+j] = amplitudes[i] * cmplx.Conj(amplitudes[j])
		}
	}

	return m, nil
}

// isNilState reports whether s is nil or holds a typed nil (for example, a
// nil *state.State boxed in the quantum.QuantumState interface). A plain
// `s == nil` check misses the typed-nil case: the interface value itself is
// non-nil (it carries a concrete type), but the underlying pointer is nil,
// and calling a method on it, such as NumQubits(), would panic.
func isNilState(s quantum.QuantumState) bool {
	if s == nil {
		return true
	}
	v := reflect.ValueOf(s)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

// NumQubits returns the number of qubits represented by the matrix.
func (m *Matrix) NumQubits() int {
	return m.numQubits
}

// Element returns the element at row, col in the density matrix.
func (m *Matrix) Element(row, col int) complex128 {
	if row < 0 || col < 0 || row >= m.dim || col >= m.dim {
		return 0
	}
	return m.data[row*m.dim+col]
}

// Trace returns the trace of the density matrix.
func (m *Matrix) Trace() float64 {
	sum := complex(0, 0)
	for i := 0; i < m.dim; i++ {
		sum += m.data[i*m.dim+i]
	}
	return real(sum)
}

// Purity returns Tr(ρ²), which is 1 for pure states and 1/2ⁿ for the
// maximally mixed state.
func (m *Matrix) Purity() float64 {
	sum := 0.0
	for _, v := range m.data {
		sum += real(v)*real(v) + imag(v)*imag(v)
	}
	return sum
}

// SetRandSource sets the randomness source used by Measure, enabling
// reproducible measurements from a seeded generator. A nil source
// restores the default (the global math/rand source).
func (m *Matrix) SetRandSource(src quantum.RandomSource) {
	m.randSource = src
}

// Probability returns the probability of measuring a specific basis
// state: the diagonal element ρᵢᵢ, exact for pure and mixed states alike.
func (m *Matrix) Probability(basisState int) float64 {
	if basisState < 0 || basisState >= m.dim {
		return 0
	}
	return real(m.data[basisState*m.dim+basisState])
}

// Amplitude returns the amplitude of a specific basis state when ρ is
// pure, and NaN when it is mixed — a mixed state has no amplitude
// vector, and this signature has no error to return. The NaN flows into
// the NaN-safe UnnormalizedStateError guards of Sample, Expectation, and
// Fidelity, which is how those helpers refuse mixed states.
//
// For pure ρ = |ψ⟩⟨ψ|, the vector is recovered from the column of the
// first basis state k with nonzero probability: ψᵢ = ρᵢₖ/√ρₖₖ, which
// fixes the global phase by making ψₖ real positive. Each call scans the
// matrix (the purity check is O(4ⁿ)); callers reading the whole vector
// on the small states this backend serves are still cheap.
func (m *Matrix) Amplitude(basisState int) complex128 {
	if basisState < 0 || basisState >= m.dim {
		return 0
	}

	if math.Abs(m.Purity()-1) > quantum.NormalizationTolerance {
		return cmplx.NaN()
	}

	for k := 0; k < m.dim; k++ {
		diag := real(m.data[k*m.dim+k])
		if diag > quantum.NormalizationTolerance {
			return m.data[basisState*m.dim+k] / complex(math.Sqrt(diag), 0)
		}
	}
	return cmplx.NaN()
}

// SetAmplitude is not supported: a density matrix has no amplitude
// vector to write one entry of. It always returns
// UnsupportedOperationError, the capability-refusal convention
// BulkAmplitudeSetter established. Prepare states with gates, FromState,
// or a state-vector backend instead.
func (m *Matrix) SetAmplitude(basisState int, value complex128) error {
	return &quantum.UnsupportedOperationError{
		Operation:   "SetAmplitude",
		Backend:     "density matrix backend",
		Alternative: "gates, FromState, or a state-vector backend",
	}
}

// Clone creates an independent copy of the density matrix. The clone
// shares the randomness source (if any), so seeded pipelines stay
// deterministic across clones; scratch buffers are not copied.
func (m *Matrix) Clone() quantum.QuantumState {
	data := make([]complex128, len(m.data))
	copy(data, m.data)
	return &Matrix{
		numQubits:  m.numQubits,
		dim:        m.dim,
		data:       data,
		randSource: m.randSource,
	}
}

// SupportsGateQubits returns whether this backend can apply gates
// operating on the specified number of qubits. The density backend
// supports all gate sizes (memory permitting).
func (m *Matrix) SupportsGateQubits(qubitCount int) bool {
	return qubitCount >= 1
}

// MaxGateQubits returns the maximum number of qubits a gate can operate
// on. Returns 0 to indicate no limit.
func (m *Matrix) MaxGateQubits() int {
	return 0
}

// ReducedBlochVector traces out all qubits except target and returns the
// Bloch vector (x, y, z) of the reduced single-qubit state. For mixed
// states the vector lies inside the unit sphere.
func (m *Matrix) ReducedBlochVector(target int) (x, y, z float64, err error) {
	if target < 0 || target >= m.numQubits {
		return 0, 0, 0, &quantum.QubitsOutOfRangeError{
			Index:    target,
			MaxIndex: m.numQubits - 1,
		}
	}

	low := (1 << target) - 1
	var r00, r01, r11 complex128
	for k := 0; k < m.dim/2; k++ {
		base := ((k &^ low) << 1) | (k & low)
		i0 := base
		i1 := base | (1 << target)
		r00 += m.data[i0*m.dim+i0]
		r01 += m.data[i0*m.dim+i1]
		r11 += m.data[i1*m.dim+i1]
	}

	x = 2 * real(r01)
	y = -2 * imag(r01)
	z = real(r00 - r11)
	return x, y, z, nil
}

// ApplyGate applies a k-qubit unitary as ρ → U ρ U†, satisfying
// quantum.QuantumState. Qubit indices are little-endian and the gate
// matrix ordering follows the targets slice, with targets[0] as the most
// significant bit — the same convention as the state-vector backends.
// Cost is O(4ⁿ·4ᵏ) for an n-qubit register.
func (m *Matrix) ApplyGate(gate quantum.Gate, targets ...int) error {
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

	if err := backendmath.ValidateTargets(targets, m.numQubits); err != nil {
		return err
	}

	if requiredQubits == 1 {
		matrix := gate.Matrix()
		if len(matrix) != 2 || len(matrix[0]) != 2 {
			return &quantum.InvalidGateApplicationError{
				Gate:        gate.Name(),
				RequiredLen: 2,
				ActualLen:   len(matrix),
			}
		}
		m.applySingleQubitOperator(matrix, targets[0], m.data, m.data)
		return nil
	}

	m.applyMultiQubitGate(gate.Matrix(), targets)
	return nil
}

// Measure performs a projective measurement of one qubit and collapses
// ρ in place: outcome b with probability Σ ρᵢᵢ over basis states whose
// qubit bit is b, then ρ → ΠρΠ / p. Randomness comes from the source
// set via SetRandSource (global math/rand by default), so outcomes are
// forceable in tests. The shared PlanCollapse supplies the outcome
// choice and its zero-branch and unnormalized-state safety.
func (m *Matrix) Measure(qubitIndex int) (int, error) {
	if qubitIndex < 0 || qubitIndex >= m.numQubits {
		return 0, &quantum.QubitsOutOfRangeError{
			Index:    qubitIndex,
			MaxIndex: m.numQubits - 1,
		}
	}

	prob0, prob1 := 0.0, 0.0
	for i := 0; i < m.dim; i++ {
		p := real(m.data[i*m.dim+i])
		if (i>>qubitIndex)&1 == 0 {
			prob0 += p
		} else {
			prob1 += p
		}
	}

	collapse, err := backendmath.PlanCollapse(qubitIndex, backendmath.RandFloat64(m.randSource), prob0, prob1)
	if err != nil {
		return 0, err
	}

	// ρᵢⱼ survives only when both indices agree with the outcome, scaled
	// by 1/p. Renormalize divides by √p, so applying it to both the row
	// and column factor of each element divides by p, exactly ΠρΠ/p.
	for i := 0; i < m.dim; i++ {
		row := i * m.dim
		for j := 0; j < m.dim; j++ {
			if collapse.Keeps(i) && collapse.Keeps(j) {
				m.data[row+j] = collapse.Renormalize(collapse.Renormalize(m.data[row+j]))
			} else {
				m.data[row+j] = 0
			}
		}
	}

	return collapse.Outcome, nil
}

// applyMultiQubitGate computes ρ → U ρ U† for a k-qubit gate, k ≥ 2.
//
// Left pass (Uρ): every column of ρ transforms exactly like a state
// vector, so the shared ComboMasks/MixCombos kernel applies per column.
// Right pass ((Uρ)U†): ((Uρ)U†)ᵢⱼ = Σₖ (Uρ)ᵢₖ·conj(Uⱼₖ), which is the
// same mixing along each row with the conjugated matrix.
//
// Buffers are allocated per call: unlike the dense backend's
// bounds-check-sensitive loops, the two O(4ⁿ) passes dominate the cost
// here, so caller-shaped buffer plumbing buys nothing.
func (m *Matrix) applyMultiQubitGate(matrix [][]complex128, targets []int) {
	comboCount := 1 << len(targets)

	comboMasks := make([]int, comboCount)
	targetMask := backendmath.ComboMasks(comboMasks, targets)

	conj := make([][]complex128, comboCount)
	for r := range conj {
		conj[r] = make([]complex128, comboCount)
		for c := range conj[r] {
			conj[r][c] = cmplx.Conj(matrix[r][c])
		}
	}

	inputs := make([]complex128, comboCount)
	outputs := make([]complex128, comboCount)
	temp := m.ensureTemp()

	// Left pass: temp = U·ρ, mixing down each column j.
	for base := 0; base < m.dim; base++ {
		if base&targetMask != 0 {
			continue
		}
		for j := 0; j < m.dim; j++ {
			for combo := 0; combo < comboCount; combo++ {
				inputs[combo] = m.data[(base|comboMasks[combo])*m.dim+j]
			}
			backendmath.MixCombos(matrix, inputs, outputs)
			for combo := 0; combo < comboCount; combo++ {
				temp[(base|comboMasks[combo])*m.dim+j] = outputs[combo]
			}
		}
	}

	// Right pass: data = temp·U†, mixing along each row i with conj(U).
	for i := 0; i < m.dim; i++ {
		row := i * m.dim
		for base := 0; base < m.dim; base++ {
			if base&targetMask != 0 {
				continue
			}
			for combo := 0; combo < comboCount; combo++ {
				inputs[combo] = temp[row+(base|comboMasks[combo])]
			}
			backendmath.MixCombos(conj, inputs, outputs)
			for combo := 0; combo < comboCount; combo++ {
				m.data[row+(base|comboMasks[combo])] = outputs[combo]
			}
		}
	}
}

// ApplyDepolarizing applies a single-qubit depolarizing channel to the target.
func (m *Matrix) ApplyDepolarizing(target int, p float64) error {
	if err := validateNoise(target, m.numQubits, p); err != nil {
		return err
	}

	scale := math.Sqrt(p / 3)
	ops := [][][]complex128{
		{
			{complex(math.Sqrt(1-p), 0), 0},
			{0, complex(math.Sqrt(1-p), 0)},
		},
		{
			{0, complex(scale, 0)},
			{complex(scale, 0), 0},
		},
		{
			{0, complex(0, -scale)},
			{complex(0, scale), 0},
		},
		{
			{complex(scale, 0), 0},
			{0, complex(-scale, 0)},
		},
	}

	return m.applySingleQubitChannel(target, ops)
}

// ApplyDephasing applies a phase-flip (dephasing) channel to the target.
func (m *Matrix) ApplyDephasing(target int, p float64) error {
	if err := validateNoise(target, m.numQubits, p); err != nil {
		return err
	}

	root := math.Sqrt(1 - p)
	scale := math.Sqrt(p)
	ops := [][][]complex128{
		{
			{complex(root, 0), 0},
			{0, complex(root, 0)},
		},
		{
			{complex(scale, 0), 0},
			{0, complex(-scale, 0)},
		},
	}

	return m.applySingleQubitChannel(target, ops)
}

// ApplyAmplitudeDamping applies an amplitude damping channel to the target.
func (m *Matrix) ApplyAmplitudeDamping(target int, gamma float64) error {
	if err := validateNoise(target, m.numQubits, gamma); err != nil {
		return err
	}

	root := math.Sqrt(1 - gamma)
	scale := math.Sqrt(gamma)
	ops := [][][]complex128{
		{
			{1, 0},
			{0, complex(root, 0)},
		},
		{
			{0, complex(scale, 0)},
			{0, 0},
		},
	}

	return m.applySingleQubitChannel(target, ops)
}

func validateNoise(target, numQubits int, p float64) error {
	if target < 0 || target >= numQubits {
		return &quantum.QubitsOutOfRangeError{
			Index:    target,
			MaxIndex: numQubits - 1,
		}
	}
	if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 1 {
		return fmt.Errorf("noise probability %f must be between 0 and 1", p)
	}
	return nil
}

func (m *Matrix) applySingleQubitChannel(target int, ops [][][]complex128) error {
	if len(ops) == 0 {
		return errors.New("no channel operators provided")
	}

	accum := m.ensureWork()
	for i := range accum {
		accum[i] = 0
	}

	out := m.ensureScratch()
	for _, op := range ops {
		if len(op) != 2 || len(op[0]) != 2 || len(op[1]) != 2 {
			return errors.New("channel operators must be 2x2")
		}
		m.applySingleQubitOperator(op, target, m.data, out)
		for i := range accum {
			accum[i] += out[i]
		}
	}

	m.data, m.work = accum, m.data
	return nil
}

func (m *Matrix) applySingleQubitOperator(op [][]complex128, target int, src, dst []complex128) {
	temp := m.ensureTemp()

	u00 := op[0][0]
	u01 := op[0][1]
	u10 := op[1][0]
	u11 := op[1][1]

	for i := 0; i < m.dim; i++ {
		i0 := i &^ (1 << target)
		i1 := i | (1 << target)
		bit := (i >> target) & 1
		row0 := i0 * m.dim
		row1 := i1 * m.dim
		row := i * m.dim

		if bit == 0 {
			for j := 0; j < m.dim; j++ {
				a0 := src[row0+j]
				a1 := src[row1+j]
				temp[row+j] = u00*a0 + u01*a1
			}
		} else {
			for j := 0; j < m.dim; j++ {
				a0 := src[row0+j]
				a1 := src[row1+j]
				temp[row+j] = u10*a0 + u11*a1
			}
		}
	}

	ud00 := cmplx.Conj(u00)
	ud01 := cmplx.Conj(u10)
	ud10 := cmplx.Conj(u01)
	ud11 := cmplx.Conj(u11)

	for i := 0; i < m.dim; i++ {
		row := i * m.dim
		for j := 0; j < m.dim; j++ {
			j0 := j &^ (1 << target)
			j1 := j | (1 << target)
			if ((j >> target) & 1) == 0 {
				a0 := temp[row+j0]
				a1 := temp[row+j1]
				dst[row+j] = a0*ud00 + a1*ud10
			} else {
				a0 := temp[row+j0]
				a1 := temp[row+j1]
				dst[row+j] = a0*ud01 + a1*ud11
			}
		}
	}
}

func (m *Matrix) ensureScratch() []complex128 {
	size := m.dim * m.dim
	if len(m.scratch) != size {
		m.scratch = make([]complex128, size)
	}
	return m.scratch
}

func (m *Matrix) ensureTemp() []complex128 {
	size := m.dim * m.dim
	if len(m.temp) != size {
		m.temp = make([]complex128, size)
	}
	return m.temp
}

func (m *Matrix) ensureWork() []complex128 {
	size := m.dim * m.dim
	if len(m.work) != size {
		m.work = make([]complex128, size)
	}
	return m.work
}
