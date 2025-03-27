package examples

import (
	"fmt"
	"math/rand"

	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/state"
)

// HadamardSingleQubitDemo demonstrates applying a Hadamard gate to a single qubit
// and shows the resulting superposition through multiple trials
func HadamardSingleQubitDemo() {
	trials := 1000
	zeros := 0
	ones := 0

	fmt.Println("Single Qubit Hadamard Gate Demonstration")
	fmt.Println("---------------------------------------")
	fmt.Println("Running", trials, "trials")

	for i := 0; i < trials; i++ {
		q := qubit.NewQubit()
		// Apply Hadamard gate to create superposition
		q.ApplyHadamard()

		// Measure the qubit
		result := q.Measure()
		if result == 0 {
			zeros++
		} else {
			ones++
		}
	}

	fmt.Printf("Results: %d zeros (%.1f%%), %d ones (%.1f%%)\n",
		zeros, float64(zeros)/float64(trials)*100,
		ones, float64(ones)/float64(trials)*100)
}

// HadamardMultipleQubitsDemo demonstrates Hadamard gates applied to multiple qubits
func HadamardMultipleQubitsDemo() {
	trials := 5
	nQubits := 2 // Number of qubits to simulate

	fmt.Println("\nMultiple Qubits Hadamard Gate Demonstration")
	fmt.Println("------------------------------------------")

	for i := 0; i < trials; i++ {
		// Create a new quantum state with n qubits
		qs := state.NewQuantumState(nQubits)

		fmt.Printf("\nTrial %d:\n", i+1)
		fmt.Println("Initial state:")
		qs.PrintState()

		// Apply Hadamard to each qubit
		for q := 0; q < nQubits; q++ {
			err := qs.ApplyHadamard(q)
			if err != nil {
				fmt.Printf("Error applying Hadamard: %v\n", err)
				return
			}
		}

		fmt.Println("\nAfter applying Hadamard gates to all qubits:")
		qs.PrintState()

		// Measure the state
		result := qs.Measure()
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
		qs := state.NewQuantumState(nQubits)

		// Apply Hadamard to all qubits
		for q := 0; q < nQubits; q++ {
			qs.ApplyHadamard(q)
		}

		// Measure and record the outcome
		result := qs.Measure()
		outcomes[result]++
	}

	// Display the distribution
	fmt.Println("\nProbability distribution:")
	fmt.Println("State\tCount\tProbability")
	fmt.Println("-----\t-----\t-----------")

	// Calculate the expected probability
	base := 1 << uint(nQubits)          // Perform shift with integers
	expectedProb := 1.0 / float64(base) // Then convert to float64

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
	qs := state.NewQuantumState(nQubits)

	// Apply Hadamard to all qubits to create superposition
	for i := 0; i < nQubits; i++ {
		qs.ApplyHadamard(i)
	}

	// Measure the state to get a random number
	result := qs.Measure()

	// Make sure the result is within our range
	return result % maxNumber
}

// RunAllHadamardDemos executes all Hadamard gate demonstrations
func RunAllHadamardDemos() {
	// Seed the random number generator
	rand.Seed(42) // Use a fixed seed for reproducible demos

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
