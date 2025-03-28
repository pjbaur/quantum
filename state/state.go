package state

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"

	"github.com/pjbaur/quantum/gates"
)

// QuantumState represents a multi-qubit quantum state.
// The state is represented as a vector of complex amplitudes for each basis state.
// The number of qubits is also stored to allow for operations on the state.
type QuantumState struct {
	Amplitudes []complex128 // State vector for all possible basis states
	NQubits    int          // Number of qubits
}

// NewQuantumState creates a new quantum state with n qubits initialized to |00...0>.
// The state is represented as a vector of complex amplitudes for each basis state.
// The initial state is |00...0> with amplitude 1.
func NewQuantumState(n int) *QuantumState {
	size := 1 << n // 2^n states
	amplitudes := make([]complex128, size)
	amplitudes[0] = 1.0 + 0i // Initial state |00...0>
	return &QuantumState{
		Amplitudes: amplitudes,
		NQubits:    n,
	}
}

// NewQuantumStateFromAmplitudes creates a quantum state with specified amplitudes.
// The number of qubits is inferred from the length of the amplitudes slice.
func NewQuantumStateFromAmplitudes(amplitudes []complex128, nQubits int) (*QuantumState, error) {
	expectedSize := 1 << nQubits
	if len(amplitudes) != expectedSize {
		return nil, fmt.Errorf("invalid amplitudes length: got %d, want %d (2^%d)",
			len(amplitudes), expectedSize, nQubits)
	}

	// Create a copy of the amplitudes
	ampCopy := make([]complex128, len(amplitudes))
	copy(ampCopy, amplitudes)

	return &QuantumState{
		Amplitudes: ampCopy,
		NQubits:    nQubits,
	}, nil
}

// ApplyHadamard applies a Hadamard gate to the specified qubit.
// The Hadamard gate is a 2x2 matrix that transforms the basis states |0⟩ and |1⟩ as follows:
// |0⟩ -> (|0⟩ + |1⟩) / √2
// |1⟩ -> (|0⟩ - |1⟩) / √2
func (qs *QuantumState) ApplyHadamard(qubit int) error {
	if qubit < 0 || qubit >= qs.NQubits {
		return fmt.Errorf("qubit index %d out of range [0,%d)", qubit, qs.NQubits)
	}

	return qs.ApplyMatrix(qubit, gates.H)
}

// Measure collapses the quantum state and returns the measured value.
// The state is collapsed to a single basis state based on the probabilities of each state.
// The probability of measuring a basis state is the squared magnitude of the amplitude.
// The state is then collapsed to the measured basis state.
func (qs *QuantumState) Measure() int {
	// Calculate probabilities
	var probs []float64
	var sum float64
	for _, amp := range qs.Amplitudes {
		prob := cmplx.Abs(amp) * cmplx.Abs(amp)
		probs = append(probs, prob)
		sum += prob
	}

	// Normalize probabilities
	for i := range probs {
		probs[i] /= sum
	}

	// Choose outcome based on probabilities
	r := rand.Float64()
	var cumulative float64
	for i := 0; i < len(probs); i++ {
		cumulative += probs[i]
		if r < cumulative {
			// Collapse to this state
			newAmplitudes := make([]complex128, len(qs.Amplitudes))
			newAmplitudes[i] = 1.0 + 0i
			qs.Amplitudes = newAmplitudes
			return i
		}
	}

	// Fallback (shouldn't normally happen with proper probabilities)
	return len(probs) - 1
}

// MeasureQubit measures a specific qubit and collapses the state accordingly.
func (qs *QuantumState) MeasureQubit(qubit int) (int, error) {
	if qubit < 0 || qubit >= qs.NQubits {
		return 0, fmt.Errorf("qubit index %d out of range [0,%d)", qubit, qs.NQubits)
	}

	// Calculate probability of measuring |0⟩
	zeroProb := 0.0
	for state := 0; state < len(qs.Amplitudes); state++ {
		// Check if the target qubit is 0
		if ((state >> qubit) & 1) == 0 {
			zeroProb += cmplx.Abs(qs.Amplitudes[state]) * cmplx.Abs(qs.Amplitudes[state])
		}
	}

	// Measure the qubit
	result := 0
	if rand.Float64() > zeroProb {
		result = 1
	}

	// Collapse the state based on measurement
	newAmplitudes := make([]complex128, len(qs.Amplitudes))
	newSum := 0.0

	for state := 0; state < len(qs.Amplitudes); state++ {
		// Check if this basis state matches our measurement result
		if ((state >> qubit) & 1) == result {
			newAmplitudes[state] = qs.Amplitudes[state]
			newSum += cmplx.Abs(newAmplitudes[state]) * cmplx.Abs(newAmplitudes[state])
		}
	}

	// Normalize the state
	normFactor := 1.0 / complex(math.Sqrt(newSum), 0)
	for i := range newAmplitudes {
		newAmplitudes[i] *= normFactor
	}

	qs.Amplitudes = newAmplitudes
	return result, nil
}

// GetProbability returns the probability of measuring a specific basis state.
// The probability is the squared magnitude of the amplitude.
// If the basis state is out of range, an error is returned.
func (qs *QuantumState) GetProbability(basisState int) (float64, error) {
	if basisState < 0 || basisState >= len(qs.Amplitudes) {
		return 0, fmt.Errorf("basis state %d out of range [0,%d)", basisState, len(qs.Amplitudes))
	}

	return cmplx.Abs(qs.Amplitudes[basisState]) * cmplx.Abs(qs.Amplitudes[basisState]), nil
}

// Clone creates a deep copy of the quantum state.
// This is useful for preserving the original state when applying operations.
func (qs *QuantumState) Clone() *QuantumState {
	newAmplitudes := make([]complex128, len(qs.Amplitudes))
	copy(newAmplitudes, qs.Amplitudes)

	return &QuantumState{
		Amplitudes: newAmplitudes,
		NQubits:    qs.NQubits,
	}
}

// PrintState prints the state in binary notation with amplitudes
func (qs *QuantumState) PrintState() {
	for i, amp := range qs.Amplitudes {
		if cmplx.Abs(amp) > 0.001 { // Only print non-negligible amplitudes
			fmt.Printf("|%0*b> : %.3f + %.3fi\n",
				qs.NQubits, i,
				real(amp), imag(amp))
		}
	}
}

// IsNormalized checks if the quantum state is properly normalized
func (qs *QuantumState) IsNormalized() bool {
	sum := 0.0
	for _, amp := range qs.Amplitudes {
		sum += cmplx.Abs(amp) * cmplx.Abs(amp)
	}
	return cmplx.Abs(complex(sum, 0)-complex(1.0, 0)) < 1e-10
}

// Normalize ensures the quantum state has unit norm.
// This is done by dividing each amplitude by the square root of the sum of squared amplitudes.
func (qs *QuantumState) Normalize() {
	sum := 0.0
	for _, amp := range qs.Amplitudes {
		sum += cmplx.Abs(amp) * cmplx.Abs(amp)
	}

	normFactor := 1.0 / complex(math.Sqrt(sum), 0)
	for i := range qs.Amplitudes {
		qs.Amplitudes[i] *= normFactor
	}
}

// ApplyCNOT applies a Controlled-NOT gate with the specified control and target qubits.
// The control qubit is the first argument and the target qubit is the second argument.
// The CNOT gate leaves the target qubit unchanged if the control qubit is |0⟩, and flips the
// target qubit if the control qubit is |1⟩. This is represented by the 4×4 matrix where the
// bottom-right 2×2 submatrix is swapped compared to the identity matrix.
func (qs *QuantumState) ApplyCNOT(controlQubit, targetQubit int) error {
	if controlQubit < 0 || controlQubit >= qs.NQubits {
		return fmt.Errorf("control qubit index %d out of range [0,%d)", controlQubit, qs.NQubits)
	}
	if targetQubit < 0 || targetQubit >= qs.NQubits {
		return fmt.Errorf("target qubit index %d out of range [0,%d)", targetQubit, qs.NQubits)
	}
	if controlQubit == targetQubit {
		return fmt.Errorf("control and target qubits must be different, got %d for both", controlQubit)
	}

	newAmplitudes := make([]complex128, len(qs.Amplitudes))

	for state := 0; state < len(qs.Amplitudes); state++ {
		if (state>>controlQubit)&1 == 1 {
			newState := state ^ (1 << targetQubit)
			newAmplitudes[newState] = qs.Amplitudes[state]
		} else {
			newAmplitudes[state] = qs.Amplitudes[state]
		}
	}

	qs.Amplitudes = newAmplitudes
	return nil
}

// ApplyMatrix applies an operation to the specified qubit using a 2x2 unitary matrix.
// The matrix is applied to the qubit in the computational basis |0⟩ and |1⟩.
func (qs *QuantumState) ApplyMatrix(qubit int, matrix [2][2]complex128) error {
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

		if bit == 0 {
			// If qubit is |0⟩: Apply first row of the matrix
			newAmplitudes[state] += matrix[0][0] * qs.Amplitudes[state]
			newAmplitudes[flipped] += matrix[0][1] * qs.Amplitudes[state]
		} else {
			// If qubit is |1⟩: Apply second row of the matrix
			newAmplitudes[flipped] += matrix[1][0] * qs.Amplitudes[state]
			newAmplitudes[state] += matrix[1][1] * qs.Amplitudes[state]
		}
	}

	qs.Amplitudes = newAmplitudes
	return nil
}

// Apply2QubitMatrix applies a 4x4 matrix operation to two qubits.
// The matrix is applied to the qubits in the computational basis |00⟩, |01⟩, |10⟩, |11⟩.
// The matrix is specified as a 4x4 complex matrix.
func (qs *QuantumState) Apply2QubitMatrix(qubit1, qubit2 int, matrix [4][4]complex128) error {
	if qubit1 < 0 || qubit1 >= qs.NQubits {
		return fmt.Errorf("first qubit index %d out of range [0,%d)", qubit1, qs.NQubits)
	}
	if qubit2 < 0 || qubit2 >= qs.NQubits {
		return fmt.Errorf("second qubit index %d out of range [0,%d)", qubit2, qs.NQubits)
	}
	if qubit1 == qubit2 {
		return fmt.Errorf("qubit indices must be different, got %d for both", qubit1)
	}

	newAmplitudes := make([]complex128, len(qs.Amplitudes))

	// For each basis state
	for state := 0; state < len(qs.Amplitudes); state++ {
		// Extract the bits at the target positions
		bit1 := (state >> qubit1) & 1
		bit2 := (state >> qubit2) & 1
		inputIdx := (bit1 << 1) | bit2

		// For each possible output
		for outputIdx := 0; outputIdx < 4; outputIdx++ {
			outBit1 := (outputIdx >> 1) & 1
			outBit2 := outputIdx & 1

			// Calculate new state by setting the target bits
			newState := state
			if bit1 != outBit1 {
				newState ^= (1 << qubit1)
			}
			if bit2 != outBit2 {
				newState ^= (1 << qubit2)
			}

			// Apply matrix element
			newAmplitudes[newState] += matrix[inputIdx][outputIdx] * qs.Amplitudes[state]
		}
	}

	qs.Amplitudes = newAmplitudes
	return nil
}
