package gates

import (
	"fmt"
	"math/cmplx"

	"github.com/yourusername/quantum/qubit"
	"github.com/yourusername/quantum/state"
)

// ApplyHadamardToQubit applies the Hadamard gate to a single qubit
func ApplyHadamardToQubit(q *qubit.Qubit) {
	newAlpha := H[0][0]*q.Alpha + H[0][1]*q.Beta
	newBeta := H[1][0]*q.Alpha + H[1][1]*q.Beta

	q.Alpha = newAlpha
	q.Beta = newBeta
}

// ApplyHadamard applies Hadamard gate to the specified qubit (0-based index) in a quantum state
func ApplyHadamard(qs *state.QuantumState, qubit int) error {
	// Input validation
	if qubit < 0 || qubit >= qs.NQubits {
		return fmt.Errorf("qubit index %d out of range [0,%d)", qubit, qs.NQubits)
	}

	newAmplitudes := make([]complex128, len(qs.Amplitudes))

	// For each basis state
	for state := 0; state < len(qs.Amplitudes); state++ {
		// Check the bit at target qubit position
		bit := (state >> qubit) & 1
		// Calculate the state with flipped bit at the target position
		flipped := state ^ (1 << qubit)

		// Apply Hadamard transformation using the global Hadamard matrix
		if bit == 0 {
			// If qubit is |0⟩: newAmplitudes += H[0][0] * currentAmplitude (no flip) + H[0][1] * currentAmplitude (with flip)
			newAmplitudes[state] += H[0][0] * qs.Amplitudes[state]
			newAmplitudes[flipped] += H[0][1] * qs.Amplitudes[state]
		} else {
			// If qubit is |1⟩: newAmplitudes += H[1][0] * currentAmplitude (with flip) + H[1][1] * currentAmplitude (no flip)
			newAmplitudes[flipped] += H[1][0] * qs.Amplitudes[state]
			newAmplitudes[state] += H[1][1] * qs.Amplitudes[state]
		}
	}

	qs.Amplitudes = newAmplitudes
	return nil
}

// ApplyCNOT applies the CNOT gate to the quantum state with specified control and target qubits
func ApplyCNOT(qs *state.QuantumState, control, target int) error {
	// Input validation
	if control < 0 || control >= qs.NQubits {
		return fmt.Errorf("control qubit index %d out of range [0,%d)", control, qs.NQubits)
	}
	if target < 0 || target >= qs.NQubits {
		return fmt.Errorf("target qubit index %d out of range [0,%d)", target, qs.NQubits)
	}
	if control == target {
		return fmt.Errorf("control and target qubits must be different, got %d for both", control)
	}

	newAmplitudes := make([]complex128, len(qs.Amplitudes))
	copy(newAmplitudes, qs.Amplitudes)

	// For each basis state
	for state := 0; state < len(qs.Amplitudes); state++ {
		// Check if control qubit is 1
		controlBit := (state >> control) & 1
		if controlBit == 1 {
			// Flip the target bit
			newState := state ^ (1 << target)
			newAmplitudes[newState] = qs.Amplitudes[state]
			newAmplitudes[state] = 0
		}
	}
	qs.Amplitudes = newAmplitudes
	return nil
}

// ApplyPauliX applies the Pauli-X (NOT) gate to the specified qubit
func ApplyPauliX(qs *state.QuantumState, qubit int) error {
	if qubit < 0 || qubit >= qs.NQubits {
		return fmt.Errorf("qubit index %d out of range [0,%d)", qubit, qs.NQubits)
	}

	newAmplitudes := make([]complex128, len(qs.Amplitudes))

	for state := 0; state < len(qs.Amplitudes); state++ {
		// Flip the bit at qubit position
		flippedState := state ^ (1 << qubit)
		newAmplitudes[flippedState] = qs.Amplitudes[state]
	}

	qs.Amplitudes = newAmplitudes
	return nil
}

// ApplyPauliZ applies the Pauli-Z gate to the specified qubit
func ApplyPauliZ(qs *state.QuantumState, qubit int) error {
	if qubit < 0 || qubit >= qs.NQubits {
		return fmt.Errorf("qubit index %d out of range [0,%d)", qubit, qs.NQubits)
	}

	for state := 0; state < len(qs.Amplitudes); state++ {
		// If the qubit is in state |1⟩, apply phase flip
		if ((state >> qubit) & 1) == 1 {
			qs.Amplitudes[state] = -qs.Amplitudes[state]
		}
	}

	return nil
}

// ApplyPhase applies a phase rotation gate to the specified qubit
func ApplyPhase(qs *state.QuantumState, qubit int, theta float64) error {
	if qubit < 0 || qubit >= qs.NQubits {
		return fmt.Errorf("qubit index %d out of range [0,%d)", qubit, qs.NQubits)
	}

	phase := cmplx.Exp(complex(0, theta))

	for state := 0; state < len(qs.Amplitudes); state++ {
		// If the qubit is in state |1⟩, apply phase rotation
		if ((state >> qubit) & 1) == 1 {
			qs.Amplitudes[state] *= phase
		}
	}

	return nil
}

// ApplyControlledPhase applies a controlled phase rotation between two qubits
func ApplyControlledPhase(qs *state.QuantumState, control, target int, theta float64) error {
	if control < 0 || control >= qs.NQubits {
		return fmt.Errorf("control qubit index %d out of range [0,%d)", control, qs.NQubits)
	}
	if target < 0 || target >= qs.NQubits {
		return fmt.Errorf("target qubit index %d out of range [0,%d)", target, qs.NQubits)
	}
	if control == target {
		return fmt.Errorf("control and target qubits must be different, got %d for both", control)
	}

	phase := cmplx.Exp(complex(0, theta))

	for state := 0; state < len(qs.Amplitudes); state++ {
		// Check if both control and target qubits are in state |1⟩
		controlBit := (state >> control) & 1
		targetBit := (state >> target) & 1

		if controlBit == 1 && targetBit == 1 {
			qs.Amplitudes[state] *= phase
		}
	}

	return nil
}

// ApplySwap swaps the states of two qubits
func ApplySwap(qs *state.QuantumState, qubit1, qubit2 int) error {
	if qubit1 < 0 || qubit1 >= qs.NQubits {
		return fmt.Errorf("qubit1 index %d out of range [0,%d)", qubit1, qs.NQubits)
	}
	if qubit2 < 0 || qubit2 >= qs.NQubits {
		return fmt.Errorf("qubit2 index %d out of range [0,%d)", qubit2, qs.NQubits)
	}
	if qubit1 == qubit2 {
		return nil // No operation needed if qubits are the same
	}

	newAmplitudes := make([]complex128, len(qs.Amplitudes))

	for state := 0; state < len(qs.Amplitudes); state++ {
		bit1 := (state >> qubit1) & 1
		bit2 := (state >> qubit2) & 1

		if bit1 != bit2 {
			// Swap the bits
			newState := state ^ (1 << qubit1) ^ (1 << qubit2)
			newAmplitudes[newState] = qs.Amplitudes[state]
		} else {
			// No swap needed if bits are the same
			newAmplitudes[state] = qs.Amplitudes[state]
		}
	}

	qs.Amplitudes = newAmplitudes
	return nil
}
