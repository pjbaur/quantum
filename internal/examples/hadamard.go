package examples

import (
	"fmt"
	"math/rand"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// HadamardSingleQubitDemo demonstrates applying a Hadamard gate to a single qubit
// and shows the resulting superposition through multiple trials
func HadamardSingleQubitDemo() {
	fmt.Println("\n=== Hadamard Gate Demonstration ===")

	// Create a new 1-qubit state in |0⟩ state
	s, err := state.New(1)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	fmt.Printf("Initial state: |0⟩ (α=%v, β=%v)\n", s.Amplitude(0), s.Amplitude(1))

	// Create and apply Hadamard gate using state-vector-first API
	h := gates.NewHadamard()
	err = s.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error applying Hadamard gate: %v\n", err)
		return
	}

	fmt.Printf("After H: |+⟩ (α=%v, β=%v)\n", s.Amplitude(0), s.Amplitude(1))
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		s.Probability(0), s.Probability(1))

	// Perform measurement
	result, _ := s.Measure(0)
	fmt.Printf("Measurement result: |%d⟩\n", result)
	fmt.Printf("Final state: (α=%v, β=%v)\n", s.Amplitude(0), s.Amplitude(1))
}

// HadamardMultipleQubitsDemo demonstrates Hadamard gates applied to multiple qubits
func HadamardMultipleQubitsDemo() {
	trials := 5
	nQubits := 2 // Number of qubits to simulate

	fmt.Println("\nMultiple Qubits Hadamard Gate Demonstration")
	fmt.Println("------------------------------------------")

	for i := 0; i < trials; i++ {
		// Create a new quantum state with n qubits
		s, err := state.New(nQubits)
		if err != nil {
			fmt.Printf("Error creating state: %v\n", err)
			return
		}
		h := gates.NewHadamard()

		fmt.Printf("\nTrial %d:\n", i+1)
		fmt.Println("Initial state:")
		printState(s)

		// Apply Hadamard to each qubit
		for q := 0; q < nQubits; q++ {
			err := s.ApplyGate(h, q)
			if err != nil {
				fmt.Printf("Error applying Hadamard: %v\n", err)
				return
			}
		}

		fmt.Println("\nAfter applying Hadamard gates to all qubits:")
		printState(s)

		// Measure all qubits and show the result
		result := 0
		for q := 0; q < nQubits; q++ {
			bit, err := s.Measure(q)
			if err != nil {
				fmt.Printf("Error measuring qubit %d: %v\n", q, err)
				return
			}
			if bit == 1 {
				result |= (1 << q)
			}
		}

		fmt.Printf("\nMeasured state: |%0*b>\n", nQubits, result)
		fmt.Println("---")
	}
}

// HadamardProbabilityDistribution demonstrates the probability distribution
// created by applying Hadamard gates to qubits
func HadamardProbabilityDistribution() {
	nQubits := 3
	trials := 1000
	outcomes := make(map[int]int)

	fmt.Println("\nHadamard Probability Distribution Demonstration")
	fmt.Println("--------------------------------------------")
	fmt.Printf("Running %d trials with %d qubits\n", trials, nQubits)

	for i := 0; i < trials; i++ {
		s, err := state.New(nQubits)
		if err != nil {
			fmt.Printf("Error creating state: %v\n", err)
			return
		}
		h := gates.NewHadamard()

		// Apply Hadamard to all qubits
		for q := 0; q < nQubits; q++ {
			err := s.ApplyGate(h, q)
			if err != nil {
				fmt.Printf("Error applying Hadamard to qubit %d: %v\n", q, err)
				return
			}
		}

		// Measure all qubits and record the outcome
		result := 0
		for q := 0; q < nQubits; q++ {
			bit, err := s.Measure(q)
			if err != nil {
				fmt.Printf("Error measuring qubit %d: %v\n", q, err)
				return
			}
			if bit == 1 {
				result |= (1 << q)
			}
		}

		outcomes[result]++
	}

	// Display the distribution
	fmt.Println("\nProbability distribution:")
	fmt.Println("State\tCount\tProbability")
	fmt.Println("-----\t-----\t-----------")

	// Calculate the expected probability (should be uniform)
	possibleOutcomes := 1 << uint(nQubits) // Explicit integer shift
	expectedProb := 1.0 / float64(possibleOutcomes)

	for i := 0; i < (1 << nQubits); i++ {
		count := outcomes[i]
		probability := float64(count) / float64(trials)
		fmt.Printf("|%0*b>\t%d\t%.1f%% (Expected: %.1f%%)\n",
			nQubits, i, count, probability*100, expectedProb*100)
	}
}

// GenerateRandomNumber uses a Hadamard-based quantum circuit to generate random numbers
func GenerateRandomNumber(maxNumber int) int {
	// Determine how many qubits we need
	nQubits := 1
	for (1 << nQubits) < maxNumber {
		nQubits++
	}

	// Create quantum state
	s, err := state.New(nQubits)
	if err != nil {
		// In case of error, fall back to classic random
		fmt.Printf("Error creating state: %v\n", err)
		return rand.Intn(maxNumber)
	}
	h := gates.NewHadamard()

	// Apply Hadamard to all qubits to create superposition
	for i := 0; i < nQubits; i++ {
		err := s.ApplyGate(h, i)
		if err != nil {
			// In case of error, fall back to classic random
			fmt.Printf("Error in quantum random generation: %v\n", err)
			return rand.Intn(maxNumber)
		}
	}

	// Measure all qubits to get a random number
	result := 0
	for q := 0; q < nQubits; q++ {
		bit, err := s.Measure(q)
		if err != nil {
			// In case of error, fall back to classic random
			fmt.Printf("Error measuring qubit %d: %v\n", q, err)
			return rand.Intn(maxNumber)
		}
		if bit == 1 {
			result |= (1 << q)
		}
	}

	// Make sure the result is within our range
	return result % maxNumber
}

// RunAllHadamardDemos executes all Hadamard gate demonstrations
func RunAllHadamardDemos() {
	fmt.Println("=============================================")
	fmt.Println("    HADAMARD GATE DEMONSTRATIONS")
	fmt.Println("=============================================")

	// Run each demonstration
	HadamardSingleQubitDemo()
	HadamardMultipleQubitsDemo()
	HadamardProbabilityDistribution()

	// Demonstrate random number generation
	fmt.Println("\nQuantum Random Number Generator")
	fmt.Println("-----------------------------")
	fmt.Println("Generating 10 random numbers between 0 and 99:")
	for i := 0; i < 10; i++ {
		num := GenerateRandomNumber(100)
		fmt.Printf("%d ", num)
	}
	fmt.Println()
	fmt.Println("=============================================")
}
