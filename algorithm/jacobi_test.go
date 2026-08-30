package algorithm_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/quantum"
)

// pauliMatrix returns the 2x2 matrix for a Pauli axis.
func pauliMatrix(axis quantum.PauliAxis) [][]float64 {
	switch axis {
	case quantum.PauliI:
		return [][]float64{{1, 0}, {0, 1}}
	case quantum.PauliX:
		return [][]float64{{0, 1}, {1, 0}}
	case quantum.PauliY:
		return [][]float64{{0, -1}, {1, 0}} // real representation of Y (up to global phase per factor); Y x Y products stay real
	case quantum.PauliZ:
		return [][]float64{{1, 0}, {0, -1}}
	}
	return nil
}

// pauliStringMatrix builds the 2^n x 2^n real matrix for axes, where axes[i]
// acts on qubit i. Qubit 0 is the least significant bit (index bit 0),
// matching quantum.Expectation's convention.
func pauliStringMatrix(axes []quantum.PauliAxis) [][]float64 {
	size := 1
	for range axes {
		size *= 2
	}
	m := make([][]float64, size)
	for i := range m {
		m[i] = make([]float64, size)
	}
	for row := 0; row < size; row++ {
		for col := 0; col < size; col++ {
			prod := 1.0
			for q, axis := range axes {
				pm := pauliMatrix(axis)
				// Matrix element of a single-qubit Pauli between basis states.
				rowBit := (row >> q) & 1
				colBit := (col >> q) & 1
				prod *= pm[rowBit][colBit]
			}
			m[row][col] = prod
		}
	}
	return m
}

// jacobiMinEigenvalue computes the smallest eigenvalue of a small real
// symmetric matrix via cyclic Jacobi rotations. Test-only verifier: the
// H2 ground energy target must come from diagonalizing the same Pauli sum
// the simulator evaluates, never from a hand-copied constant.
func jacobiMinEigenvalue(t testing.TB, a [][]float64) float64 {
	t.Helper()
	n := len(a)
	// Work on a copy.
	m := make([][]float64, n)
	for i := range m {
		m[i] = append([]float64(nil), a[i]...)
	}
	for sweep := 0; sweep < 100; sweep++ {
		off := 0.0
		for p := 0; p < n; p++ {
			for q := p + 1; q < n; q++ {
				off += m[p][q] * m[p][q]
			}
		}
		if off < 1e-24 {
			break
		}
		for p := 0; p < n; p++ {
			for q := p + 1; q < n; q++ {
				if math.Abs(m[p][q]) < 1e-15 {
					continue
				}
				theta := (m[q][q] - m[p][p]) / (2 * m[p][q])
				tSign := 1.0
				if theta < 0 {
					tSign = -1.0
				}
				tval := tSign / (theta*tSign + math.Sqrt(theta*theta+1))
				c := 1 / math.Sqrt(tval*tval+1)
				s := tval * c
				for k := 0; k < n; k++ {
					mkp, mkq := m[k][p], m[k][q]
					m[k][p] = c*mkp - s*mkq
					m[k][q] = s*mkp + c*mkq
				}
				for k := 0; k < n; k++ {
					mpk, mqk := m[p][k], m[q][k]
					m[p][k] = c*mpk - s*mqk
					m[q][k] = s*mpk + c*mqk
				}
			}
		}
	}
	min := m[0][0]
	for i := 1; i < n; i++ {
		if m[i][i] < min {
			min = m[i][i]
		}
	}
	return min
}

// h2Matrix builds the 4x4 real-symmetric H2 matrix from the same Pauli
// terms the simulator evaluates (algorithm.H2Terms).
func h2Matrix(t testing.TB) [][]float64 {
	t.Helper()
	n := 4
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
	}
	for _, term := range algorithm.H2Terms() {
		if len(term.Axes) == 0 {
			for i := 0; i < n; i++ {
				m[i][i] += term.Coeff
			}
			continue
		}
		pm := pauliStringMatrix(term.Axes)
		for r := 0; r < n; r++ {
			for c := 0; c < n; c++ {
				m[r][c] += term.Coeff * pm[r][c]
			}
		}
	}
	return m
}

// h2GroundEnergy diagonalizes the H2 Hamiltonian's Pauli sum independently
// of the simulator path.
func h2GroundEnergy(t testing.TB) float64 {
	t.Helper()
	return jacobiMinEigenvalue(t, h2Matrix(t))
}
