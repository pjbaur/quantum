package main

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"
)

type Qubit struct {
	Alpha complex128
	Beta  complex128
}

type QuantumState struct {
	Amplitudes []complex128 // State vector for all possible combinations
	NQubits    int          // Number of qubits
}

func NewQubit() *Qubit {
	return &Qubit{
		Alpha: 1.0 + 0i,
		Beta:  0.0 + 0i,
	}
}

func (q *Qubit) ApplyHadamard() {
	h00 := complex(1.0/math.Sqrt(2), 0)
	h01 := complex(1.0/math.Sqrt(2), 0)
	h10 := complex(1.0/math.Sqrt(2), 0)
	h11 := complex(-1.0/math.Sqrt(2), 0)

	newAlpha := h00*q.Alpha + h01*q.Beta
	newBeta := h10*q.Alpha + h11*q.Beta

	q.Alpha = newAlpha
	q.Beta = newBeta
}

// Apply Hadamard gate to the specified qubit (0-based index)
func (qs *QuantumState) ApplyHadamard(qubit int) error {
	// Input validation
	if qubit < 0 || qubit >= qs.NQubits {
		return fmt.Errorf("qubit index %d out of range [0,%d)", qubit, qs.NQubits)
	}

	h := [2][2]complex128{
		{complex(1.0/math.Sqrt(2), 0), complex(1.0/math.Sqrt(2), 0)},
		{complex(1.0/math.Sqrt(2), 0), complex(-1.0/math.Sqrt(2), 0)},
	}

	newAmplitudes := make([]complex128, len(qs.Amplitudes))

	// For each basis state
	for state := 0; state < len(qs.Amplitudes); state++ {
		// Check the bit at target qubit position
		bit := (state >> qubit) & 1
		// Calculate the state with flipped bit
		flipped := state ^ (1 << qubit)

		// Apply Hadamard transformation
		for i := 0; i < 2; i++ {
			src := state
			if i == 0 {
				src = flipped
			}
			newAmplitudes[state] += h[bit][i] * qs.Amplitudes[src]
		}
	}
	qs.Amplitudes = newAmplitudes
	return nil
}

func (q *Qubit) Measure() int {
	// Fix: Remove unnecessary real() since Abs squared is already float64
	prob0 := cmplx.Abs(q.Alpha) * cmplx.Abs(q.Alpha)
	if rand.Float64() < prob0 {
		q.Alpha = 1.0 + 0i
		q.Beta = 0.0 + 0i
		return 0
	}
	q.Alpha = 0.0 + 0i
	q.Beta = 1.0 + 0i
	return 1
}

// Measure all qubits and collapse the state
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
	return len(probs) - 1 // Fallback
}

// Creates a new quantum state with n qubits initialized to |00...0>
func NewQuantumState(n int) *QuantumState {
	size := 1 << n // 2^n states
	amplitudes := make([]complex128, size)
	amplitudes[0] = 1.0 + 0i // Initial state |00...0>
	return &QuantumState{
		Amplitudes: amplitudes,
		NQubits:    n,
	}
}

// Helper function to print state in binary format
func (qs *QuantumState) PrintState() {
	for i, amp := range qs.Amplitudes {
		if cmplx.Abs(amp) > 0.001 { // Only print non-negligible amplitudes
			fmt.Printf("|%0*b> : %.3f + %.3fi\n",
				qs.NQubits, i,
				real(amp), imag(amp))
		}
	}
}

// Apply CNOT gate to QuantumState, returns error instead of panicking
func (qs *QuantumState) ApplyCNOT(control, target int) error {
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

// Add a test function for CNOT with error handling
func CNOTTrials() {
	fmt.Println("\nCNOT gate trials:")

	// Trial 1: |00> -> |00> (control=0, no flip)
	qs1 := NewQuantumState(2)
	fmt.Println("Initial state |00>:")
	qs1.PrintState()
	err := qs1.ApplyCNOT(0, 1)
	if err != nil {
		fmt.Printf("Error applying CNOT: %v\n", err)
		return
	}
	fmt.Println("After CNOT (control=0, target=1):")
	qs1.PrintState()

	// Trial 2: |10> -> |11> (control=1, flip target)
	qs2 := NewQuantumState(2)
	qs2.Amplitudes[0] = 0
	qs2.Amplitudes[2] = 1.0 + 0i // |10>
	fmt.Println("\nInitial state |10>:")
	qs2.PrintState()
	err = qs2.ApplyCNOT(0, 1)
	if err != nil {
		fmt.Printf("Error applying CNOT: %v\n", err)
		return
	}
	fmt.Println("After CNOT (control=0, target=1):")
	qs2.PrintState()

	// Trial 3: Bell state creation
	qs3 := NewQuantumState(2)
	err = qs3.ApplyHadamard(0)
	if err != nil {
		fmt.Printf("Error applying Hadamard: %v\n", err)
		return
	}
	err = qs3.ApplyCNOT(0, 1)
	if err != nil {
		fmt.Printf("Error applying CNOT: %v\n", err)
		return
	}
	fmt.Println("\nAfter H(0) then CNOT(0,1) - Bell state:")
	qs3.PrintState()

	// Demo of error handling
	fmt.Println("\nTrying invalid CNOT operation (control=target):")
	err = qs3.ApplyCNOT(1, 1)
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}
}

func HadamardTrials() {
	trials := 1000
	zeros := 0
	ones := 0

	for i := 0; i < trials; i++ {
		qubit := NewQubit()
		// fmt.Printf("Initial state: Alpha=%.3f, Beta=%.3f\n", real(qubit.Alpha), real(qubit.Beta))

		qubit.ApplyHadamard()
		// fmt.Printf("After Hadamard: Alpha=%.3f, Beta=%.3f\n", real(qubit.Alpha), real(qubit.Beta))

		result := qubit.Measure()
		// fmt.Printf("Measured: %d\n", result)
		if result == 0 {
			zeros++
		} else {
			ones++
		}
	}

	fmt.Printf("\nResults: %d zeros, %d ones\n", zeros, ones)
}

func MultipleQubitsTrials() {
	trials := 5
	nQubits := 2 // Number of qubits to simulate

	for i := 0; i < trials; i++ {
		qs := NewQuantumState(nQubits)
		fmt.Printf("Trial %d:\nInitial state:\n", i+1)
		qs.PrintState()

		// Apply Hadamard to each qubit
		for q := 0; q < nQubits; q++ {
			qs.ApplyHadamard(q)
		}
		fmt.Printf("\nAfter Hadamard gates:\n")
		qs.PrintState()

		result := qs.Measure()
		fmt.Printf("\nMeasured state: |%0*b>\n", nQubits, result)
		fmt.Println("---")
	}
}

func main() {
	fmt.Println("Quantum computing in Go")
	fmt.Println("------------------------")
	fmt.Println("Hadamard gate trials:")
	HadamardTrials()
	fmt.Println("------------------------")
	fmt.Println("\nMultiple qubits trials:")
	MultipleQubitsTrials()
	fmt.Println("------------------------")
	fmt.Println("\nCNOT gate trials:")
	CNOTTrials()
}
