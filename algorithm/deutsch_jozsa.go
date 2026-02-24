package algorithm

import (
	"errors"
	"fmt"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// Oracle defines the boolean function used by Deutsch-Jozsa.
// It must return 0 or 1 for any input in [0, 2^n).
type Oracle func(input int) int

// DeutschJozsa executes the Deutsch-Jozsa algorithm and returns the final state.
// The state includes numInputQubits input qubits and one ancilla qubit.
func DeutschJozsa(numInputQubits int, oracle Oracle) (*state.State, error) {
	if numInputQubits <= 0 {
		return nil, errors.New("numInputQubits must be positive")
	}
	if oracle == nil {
		return nil, errors.New("oracle must not be nil")
	}

	totalQubits := numInputQubits + 1
	c, err := circuit.New(totalQubits)
	if err != nil {
		return nil, err
	}

	ancilla := numInputQubits
	xGate := gates.NewPauliX()
	if err := c.AddGate(xGate, ancilla); err != nil {
		return nil, err
	}

	hGate := gates.NewHadamard()
	for i := 0; i < totalQubits; i++ {
		if err := c.AddGate(hGate, i); err != nil {
			return nil, err
		}
	}

	oracleGate, err := deutschJozsaOracleGate(numInputQubits, oracle)
	if err != nil {
		return nil, err
	}
	if err := c.AddGate(oracleGate, descendingTargets(totalQubits)...); err != nil {
		return nil, err
	}

	for i := 0; i < numInputQubits; i++ {
		if err := c.AddGate(hGate, i); err != nil {
			return nil, err
		}
	}

	finalState, err := state.New(totalQubits)
	if err != nil {
		return nil, err
	}
	if err := c.Execute(finalState); err != nil {
		return nil, err
	}

	return finalState, nil
}

func deutschJozsaOracleGate(numInputQubits int, oracle Oracle) (*matrixGate, error) {
	size := 1 << (numInputQubits + 1)
	matrix := make([][]complex128, size)
	for i := range matrix {
		matrix[i] = make([]complex128, size)
	}

	ancillaBit := 1 << numInputQubits
	mask := ancillaBit - 1
	for col := 0; col < size; col++ {
		input := col & mask
		value := oracle(input)
		if value != 0 && value != 1 {
			return nil, fmt.Errorf("oracle returned %d for input %d", value, input)
		}

		out := col
		if value == 1 {
			out = col ^ ancillaBit
		}
		matrix[out][col] = 1
	}

	return newMatrixGate("DeutschJozsaOracle", matrix)
}
