package algorithm

import (
	"errors"
	"fmt"
	"math"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// Grover executes Grover's search algorithm and returns the final state.
// The marked slice contains basis states to amplify.
func Grover(numQubits int, marked []int) (*state.State, error) {
	if numQubits <= 0 {
		return nil, errors.New("numQubits must be positive")
	}
	if len(marked) == 0 {
		return nil, errors.New("marked set must not be empty")
	}

	totalStates := 1 << numQubits
	markedSet, err := uniqueMarked(marked, totalStates)
	if err != nil {
		return nil, err
	}

	hGate := gates.NewHadamard()
	search, err := state.New(numQubits)
	if err != nil {
		return nil, err
	}
	for i := 0; i < numQubits; i++ {
		if err := search.ApplyGate(hGate, i); err != nil {
			return nil, err
		}
	}

	oracleGate, err := groverOracleGate(numQubits, markedSet)
	if err != nil {
		return nil, err
	}
	diffusionGate, err := groverDiffusionGate(numQubits)
	if err != nil {
		return nil, err
	}

	iterations := groverIterations(totalStates, len(markedSet))
	targets := descendingTargets(numQubits)
	for i := 0; i < iterations; i++ {
		if err := search.ApplyGate(oracleGate, targets...); err != nil {
			return nil, err
		}
		if err := search.ApplyGate(diffusionGate, targets...); err != nil {
			return nil, err
		}
	}

	return search, nil
}

func groverIterations(totalStates, marked int) int {
	estimate := (math.Pi / 4) * math.Sqrt(float64(totalStates)/float64(marked))
	iterations := int(math.Round(estimate))
	if iterations < 1 {
		return 1
	}
	return iterations
}

func uniqueMarked(marked []int, totalStates int) ([]int, error) {
	seen := make(map[int]struct{}, len(marked))
	unique := make([]int, 0, len(marked))
	for _, state := range marked {
		if state < 0 || state >= totalStates {
			return nil, fmt.Errorf("marked state %d out of range", state)
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		unique = append(unique, state)
	}
	return unique, nil
}

func groverOracleGate(numQubits int, marked []int) (*matrixGate, error) {
	size := 1 << numQubits
	matrix := make([][]complex128, size)
	for i := range matrix {
		matrix[i] = make([]complex128, size)
	}

	markedSet := make(map[int]struct{}, len(marked))
	for _, state := range marked {
		markedSet[state] = struct{}{}
	}

	for i := 0; i < size; i++ {
		value := complex(1, 0)
		if _, ok := markedSet[i]; ok {
			value = -1
		}
		matrix[i][i] = value
	}

	return newMatrixGate("GroverOracle", matrix)
}

func groverDiffusionGate(numQubits int) (*matrixGate, error) {
	size := 1 << numQubits
	matrix := make([][]complex128, size)
	for i := range matrix {
		matrix[i] = make([]complex128, size)
	}

	twoOverN := 2.0 / float64(size)
	for row := 0; row < size; row++ {
		for col := 0; col < size; col++ {
			value := twoOverN
			if row == col {
				value -= 1
			}
			matrix[row][col] = complex(value, 0)
		}
	}

	return newMatrixGate("GroverDiffusion", matrix)
}
