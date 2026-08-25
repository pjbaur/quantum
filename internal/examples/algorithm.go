package examples

import (
	"fmt"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/state"
)

// DeutschJozsaDemo shows how the algorithm distinguishes constant vs balanced oracles.
func DeutschJozsaDemo() {
	fmt.Println("\n=== Deutsch-Jozsa Demonstration ===")

	// Two input qubits plus the ancilla. The algorithm runs on whatever
	// backend it is handed; the dense one is the general-purpose choice.
	constantStart, err := state.New(3)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	constantOracle := func(int) int { return 0 }
	constantState, err := algorithm.DeutschJozsa(constantStart, constantOracle)
	if err != nil {
		fmt.Printf("Error running Deutsch-Jozsa: %v\n", err)
		return
	}

	balancedStart, err := state.New(3)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	balancedOracle := func(input int) int { return input & 1 }
	balancedState, err := algorithm.DeutschJozsa(balancedStart, balancedOracle)
	if err != nil {
		fmt.Printf("Error running Deutsch-Jozsa: %v\n", err)
		return
	}

	constantProb := constantState.Probability(0) + constantState.Probability(1<<2)
	balancedProb := balancedState.Probability(0) + balancedState.Probability(1<<2)

	fmt.Printf("Constant oracle zero-register probability: %.2f\n", constantProb)
	fmt.Printf("Balanced oracle zero-register probability: %.2f\n", balancedProb)
}

// GroverDemo shows probability amplification for a marked state.
func GroverDemo() {
	fmt.Println("\n=== Grover Demonstration ===")

	start, err := state.New(3)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	searchState, err := algorithm.Grover(start, []int{5})
	if err != nil {
		fmt.Printf("Error running Grover: %v\n", err)
		return
	}

	fmt.Printf("Marked state probability: %.2f\n", searchState.Probability(5))
}

// RunAllAlgorithmDemos executes all algorithm demonstrations.
func RunAllAlgorithmDemos() {
	DeutschJozsaDemo()
	GroverDemo()
}
