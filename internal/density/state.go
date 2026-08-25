package density

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"

	"github.com/pjbaur/quantum/quantum"
)

// Matrix represents a density matrix for an n-qubit system.
// This type is internal to the module to keep the API surface focused.
type Matrix struct {
	numQubits int
	dim       int
	data      []complex128
	temp      []complex128
	scratch   []complex128
	work      []complex128
}

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

// ApplySingleQubitGate applies a single-qubit unitary to the specified target.
func (m *Matrix) ApplySingleQubitGate(gate quantum.Gate, target int) error {
	if target < 0 || target >= m.numQubits {
		return &quantum.QubitsOutOfRangeError{
			Index:    target,
			MaxIndex: m.numQubits - 1,
		}
	}

	matrix := gate.Matrix()
	if len(matrix) != 2 || len(matrix[0]) != 2 || len(matrix[1]) != 2 {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: 2,
			ActualLen:   len(matrix),
		}
	}

	m.applySingleQubitOperator(matrix, target, m.data, m.data)
	return nil
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
