package gates

import (
	"errors"
	"fmt"
	"math/bits"
)

// MatrixGate is a quantum gate defined entirely by its unitary matrix.
type MatrixGate struct {
	name   string
	matrix [][]complex128
	qubits int
}

// NewMatrixGate validates the matrix and returns a gate.
// The matrix must be non-empty, square, and its dimension a power of two >= 2.
func NewMatrixGate(name string, matrix [][]complex128) (*MatrixGate, error) {
	if name == "" {
		return nil, errors.New("gate name must not be empty")
	}
	size, err := squareSize(matrix)
	if err != nil {
		return nil, err
	}
	if size < 2 {
		return nil, errors.New("matrix must be at least 2x2")
	}
	if size&(size-1) != 0 {
		return nil, fmt.Errorf("matrix size %d is not a power of two", size)
	}

	return &MatrixGate{
		name:   name,
		matrix: copyMatrix(matrix),
		qubits: bits.Len(uint(size)) - 1,
	}, nil
}

// Name returns the name of the gate.
func (g *MatrixGate) Name() string {
	return g.name
}

// Matrix returns a copy of the matrix representation of the gate.
func (g *MatrixGate) Matrix() [][]complex128 {
	return copyMatrix(g.matrix)
}

// NumQubits returns the number of qubits the gate operates on.
func (g *MatrixGate) NumQubits() int {
	return g.qubits
}

func copyMatrix(matrix [][]complex128) [][]complex128 {
	out := make([][]complex128, len(matrix))
	for i, row := range matrix {
		out[i] = append([]complex128(nil), row...)
	}
	return out
}
