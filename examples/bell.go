/*
This `examples/bell.go` file contains several demonstration functions focused on Bell states and their applications in quantum information:

1. `CreateBellState()` - A utility function that creates the standard Bell state (Φ+): (|00⟩ + |11⟩)/√2.

2. `BellStateDemo()` - Demonstrates the step-by-step creation of a Bell state using Hadamard and CNOT gates.

3. `BellStateMeasurementCorrelation()` - Shows the perfect correlation between measurements of qubits in a Bell state through multiple trials.

4. `CreateFourBellStates()` - Creates and displays all four Bell states (Φ+, Φ-, Ψ+, Ψ-) using combinations of quantum gates.

5. `BellStateEntanglementDemo()` - Demonstrates quantum entanglement by measuring one qubit of a Bell pair and observing the effect on the other qubit.

6. `QuantumTeleportationDemo()` - Simulates quantum teleportation using Bell states, one of the most important quantum information protocols.

7. `AreQubitStatesEqual()` - A helper function to compare qubit states within a numerical tolerance.

8. `RunAllBellDemos()` - A convenience function that runs all the demonstrations in sequence.

The code assumes your new package structure with imports from packages like `quantum/state`. It includes detailed explanations of Bell states and their properties, focusing on entanglement and non-classical correlations that make Bell states fundamental to quantum information science.

Note that the implementation includes some placeholder functions (like `MeasureQubit()` and `GetQubitState()`) that would need to be implemented in your `state` package. The exact implementation would depend on how you design your state representation and measurement operations.
*/

package examples

import (
	"fmt"
	"math/cmplx"
	"math/rand"

	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/state"
)

// CreateBellState creates the standard Bell state (Φ+): |00⟩ + |11⟩ / √2
func CreateBellState() *state.QuantumState {
	// Start with a 2-qubit state in |00⟩
	qs := state.NewQuantumState(2)

	// Apply Hadamard to the first qubit
	qs.ApplyHadamard(0)

	// Apply CNOT with first qubit as control and second as target
	qs.ApplyCNOT(0, 1)

	return qs
}

// BellStateDemo demonstrates the creation of the Bell state
func BellStateDemo() {
	fmt.Println("Bell State Creation Demonstration")
	fmt.Println("-------------------------------")

	// Create a new 2-qubit state initialized to |00⟩
	qs := state.NewQuantumState(2)
	fmt.Println("Initial state |00⟩:")
	qs.PrintState()

	// Apply Hadamard to first qubit
	qs.ApplyHadamard(0)
	fmt.Println("\nAfter applying Hadamard to qubit 0:")
	qs.PrintState()

	// Apply CNOT with first qubit as control and second as target
	qs.ApplyCNOT(0, 1)
	fmt.Println("\nAfter applying CNOT (control=0, target=1):")
	fmt.Println("This is the Bell state |Φ+⟩ = (|00⟩ + |11⟩)/√2:")
	qs.PrintState()
}

// BellStateMeasurementCorrelation demonstrates the measurement correlation in Bell states
func BellStateMeasurementCorrelation() {
	trials := 1000
	results := make(map[int]int)

	fmt.Println("\nBell State Measurement Correlation")
	fmt.Println("--------------------------------")
	fmt.Printf("Running %d trials\n", trials)

	for i := 0; i < trials; i++ {
		// Create Bell state
		qs := CreateBellState()

		// Measure the state
		outcome := qs.Measure()
		results[outcome]++
	}

	// Display results
	total := float64(trials)
	fmt.Println("\nMeasurement results:")
	fmt.Printf("|00⟩: %d occurrences (%.1f%%)\n", results[0], float64(results[0])/total*100)
	fmt.Printf("|01⟩: %d occurrences (%.1f%%)\n", results[1], float64(results[1])/total*100)
	fmt.Printf("|10⟩: %d occurrences (%.1f%%)\n", results[2], float64(results[2])/total*100)
	fmt.Printf("|11⟩: %d occurrences (%.1f%%)\n", results[3], float64(results[3])/total*100)

	fmt.Println("\nNote: Observe that only |00⟩ and |11⟩ states occur, demonstrating")
	fmt.Println("the perfect correlation between qubits in the Bell state.")
}

// CreateFourBellStates demonstrates creating all four Bell states
func CreateFourBellStates() {
	fmt.Println("\nCreating the Four Bell States")
	fmt.Println("---------------------------")

	// Φ+ = (|00⟩ + |11⟩)/√2 - Standard Bell state
	fmt.Println("Bell state |Φ+⟩ = (|00⟩ + |11⟩)/√2:")
	bellPhiPlus := CreateBellState()
	bellPhiPlus.PrintState()

	// Φ- = (|00⟩ - |11⟩)/√2
	fmt.Println("\nBell state |Φ-⟩ = (|00⟩ - |11⟩)/√2:")
	bellPhiMinus := CreateBellState()
	bellPhiMinus.ApplyZ(1) // Apply Z gate to second qubit
	bellPhiMinus.PrintState()

	// Ψ+ = (|01⟩ + |10⟩)/√2
	fmt.Println("\nBell state |Ψ+⟩ = (|01⟩ + |10⟩)/√2:")
	bellPsiPlus := CreateBellState()
	bellPsiPlus.ApplyX(1) // Apply X gate to second qubit
	bellPsiPlus.PrintState()

	// Ψ- = (|01⟩ - |10⟩)/√2
	fmt.Println("\nBell state |Ψ-⟩ = (|01⟩ - |10⟩)/√2:")
	bellPsiMinus := CreateBellState()
	bellPsiMinus.ApplyX(1) // Apply X gate to second qubit
	bellPsiMinus.ApplyZ(1) // Apply Z gate to second qubit
	bellPsiMinus.PrintState()
}

// BellStateEntanglementDemo demonstrates entanglement by measuring one qubit
// of a Bell state and observing the effect on the other qubit
func BellStateEntanglementDemo() {
	trials := 1000
	correlatedResults := 0

	fmt.Println("\nBell State Entanglement Demonstration")
	fmt.Println("---------------------------------")
	fmt.Printf("Running %d trials\n", trials)

	for i := 0; i < trials; i++ {
		// Create a new 2-qubit quantum state
		qs := state.New(2)

		// Create Bell state
		qs.ApplyHadamard(0)
		qs.ApplyCNOT(0, 1)

		// Measure only the first qubit using custom measurement
		// Implementation depends on your state package's capabilities
		firstQubitResult := qs.MeasureQubit(0)

		// Now measure the second qubit
		secondQubitResult := qs.MeasureQubit(1)

		// In a Bell state, the qubits should have the same value
		if firstQubitResult == secondQubitResult {
			correlatedResults++
		}
	}

	// Display correlation statistics
	correlationPercentage := float64(correlatedResults) / float64(trials) * 100
	fmt.Printf("\nQubit measurements matched in %d out of %d trials (%.1f%%)\n",
		correlatedResults, trials, correlationPercentage)

	fmt.Println("\nNote: The strong correlation between measurements demonstrates")
	fmt.Println("quantum entanglement, a non-classical property where measuring")
	fmt.Println("one qubit instantaneously determines the state of the other.")
}

// QuantumTeleportationDemo simulates quantum teleportation using Bell states
func QuantumTeleportationDemo() {
	trials := 100
	successfulTeleportations := 0

	fmt.Println("\nQuantum Teleportation Using Bell States")
	fmt.Println("-----------------------------------")
	fmt.Printf("Running %d trials\n", trials)

	for i := 0; i < trials; i++ {
		// Create a 3-qubit system: message qubit + Bell pair
		qs := state.New(3)

		// Prepare a random state for the message qubit (qubit 0)
		if rand.Float64() < 0.5 {
			// Create a superposition state by applying Hadamard
			qs.ApplyHadamard(0)
			// Apply a random phase with T gate for more variety
			if rand.Float64() < 0.5 {
				qs.ApplyT(0)
			}
		} else {
			// Or just flip to |1⟩ state for simple demonstration
			qs.ApplyX(0)
		}

		// Record the initial state of the message qubit
		initialState := qs.GetQubitState(0)

		// Create Bell state between qubits 1 and 2
		qs.ApplyHadamard(1)
		qs.ApplyCNOT(1, 2)

		// Begin teleportation protocol

		// 1. Entangle the message qubit with the Bell pair
		qs.ApplyCNOT(0, 1)
		qs.ApplyHadamard(0)

		// 2. Measure the first two qubits (message and first of Bell pair)
		// This would collapse the state
		firstQubitResult := qs.MeasureQubit(0)
		secondQubitResult := qs.MeasureQubit(1)

		// 3. Apply corrections to the third qubit based on measurement results
		if secondQubitResult == 1 {
			qs.ApplyX(2)
		}
		if firstQubitResult == 1 {
			qs.ApplyZ(2)
		}

		// 4. Verify teleportation success by checking final state of qubit 2
		finalState := qs.GetQubitState(2)

		// Compare the initial and final states (within tolerance)
		if AreQubitStatesEqual(initialState, finalState) {
			successfulTeleportations++
		}
	}

	// Display teleportation success rate
	successRate := float64(successfulTeleportations) / float64(trials) * 100
	fmt.Printf("\nSuccessful teleportations: %d out of %d trials (%.1f%%)\n",
		successfulTeleportations, trials, successRate)

	fmt.Println("\nNote: In a perfect quantum system, teleportation success rate would be 100%.")
	fmt.Println("Any deviations are due to simulation approximations or implementation details.")
}

// AreQubitStatesEqual compares two qubit states within numerical tolerance
func AreQubitStatesEqual(state1, state2 *qubit.Qubit) bool {
	// Implementation depends on your qubit package, this is a placeholder
	// Real implementation would compare alpha and beta values within tolerance

	// This is a simplified comparison assuming a qubit.State struct with Alpha and Beta fields
	alphaDiff := cmplx.Abs(state1.Alpha - state2.Alpha)
	betaDiff := cmplx.Abs(state1.Beta - state2.Beta)

	// Allow small numerical differences
	tolerance := 1e-10
	return alphaDiff < tolerance && betaDiff < tolerance
}

// RunAllBellDemos executes all Bell state demonstrations
func RunAllBellDemos() {
	// Seed the random number generator
	rand.Seed(42) // Use a fixed seed for reproducible demos

	fmt.Println("=============================================")
	fmt.Println("         BELL STATE DEMONSTRATIONS")
	fmt.Println("=============================================")

	// Run each demonstration
	BellStateDemo()
	BellStateMeasurementCorrelation()
	CreateFourBellStates()
	BellStateEntanglementDemo()
	QuantumTeleportationDemo()

	fmt.Println("=============================================")
}
