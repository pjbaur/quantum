// Package algorithm provides small, composable quantum algorithms.
package algorithm

import (
	"errors"
	"fmt"
	"math/bits"

	"github.com/pjbaur/quantum/quantum"
)

type matrixGate struct {
	name   string
	matrix [][]complex128
	qubits int
}

func newMatrixGate(name string, matrix [][]complex128) (*matrixGate, error) {
	if len(matrix) == 0 || len(matrix) != len(matrix[0]) {
		return nil, errors.New("matrix must be non-empty and square")
	}
	size := len(matrix)
	if size&(size-1) != 0 {
		return nil, fmt.Errorf("matrix size %d is not a power of two", size)
	}
	for _, row := range matrix {
		if len(row) != size {
			return nil, errors.New("matrix must be square")
		}
	}

	return &matrixGate{
		name:   name,
		matrix: matrix,
		qubits: bits.Len(uint(size)) - 1,
	}, nil
}

func (g *matrixGate) Name() string {
	return g.name
}

func (g *matrixGate) Matrix() [][]complex128 {
	return g.matrix
}

func (g *matrixGate) Apply(q quantum.Qubit) error {
	return &quantum.InvalidGateApplicationError{
		Gate:        g.name,
		RequiredLen: g.qubits,
		ActualLen:   1,
	}
}

func descendingTargets(count int) []int {
	targets := make([]int, count)
	for i := 0; i < count; i++ {
		targets[i] = count - 1 - i
	}
	return targets
}
