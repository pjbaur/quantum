// Package algorithm provides small, composable quantum algorithms.
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

	// Execute circuit to get |+⟩ on all qubits with ancilla in |-⟩ state
	finalState, err := state.New(totalQubits)
	if err != nil {
		return nil, err
	}
	if err := c.Execute(finalState); err != nil {
		return nil, err
	}

	// Apply oracle directly to state vector (no matrix construction)
	if err := applyDeutschJozsaOracle(finalState, numInputQubits, oracle); err != nil {
		return nil, err
	}

	// Apply Hadamard to input qubits
	for i := 0; i < numInputQubits; i++ {
		if err := finalState.ApplyGate(hGate, i); err != nil {
			return nil, err
		}
	}

	return finalState, nil
}

// applyDeutschJozsaOracle applies the oracle directly to the state vector.
// The oracle maps |x⟩|y⟩ → |x⟩|y ⊕ f(x)⟩.
// For each input x:
//   - If f(x) = 0: leave the pair (|x,0⟩, |x,1⟩) unchanged
//   - If f(x) = 1: swap the pair (|x,0⟩ ↔ |x,1⟩)
//
// This is O(2^n) instead of O(4^n) for constructing the full oracle matrix.
func applyDeutschJozsaOracle(s *state.State, numInputQubits int, oracle Oracle) error {
	numInputStates := 1 << numInputQubits
	ancillaBit := 1 << numInputQubits
	totalStates := 1 << (numInputQubits + 1)

	// Get all amplitudes
	amps := make([]complex128, totalStates)
	for i := 0; i < totalStates; i++ {
		amps[i] = s.Amplitude(i)
	}

	// Apply oracle: for each input x, conditionally swap |x,0⟩ and |x,1⟩
	for x := 0; x < numInputStates; x++ {
		fx := oracle(x)
		if fx != 0 && fx != 1 {
			return fmt.Errorf("oracle returned %d for input %d", fx, x)
		}

		if fx == 1 {
			// Swap amplitudes at |x,0⟩ and |x,1⟩
			idx0 := x              // |x⟩|0⟩
			idx1 := x | ancillaBit // |x⟩|1⟩
			amps[idx0], amps[idx1] = amps[idx1], amps[idx0]
		}
	}

	// Set all amplitudes at once (single normalization check)
	return s.SetAmplitudes(amps)
}
