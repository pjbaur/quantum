package examples

import (
	"fmt"

	"github.com/pjbaur/quantum/state"
)

// Helper function to print quantum state in a readable format
func printState(s *state.State) {
	// fmt.Println("State vector:")
	// s.Print()

	numQubits := s.NumQubits()
	numStates := 1 << numQubits

	for i := 0; i < numStates; i++ {
		amplitude := s.Amplitude(i)
		probability := s.Probability(i)
		if probability > 0.001 { // Only show non-zero probabilities
			fmt.Printf("|%0*b>: %.4f (%.1f%%)\n",
				numQubits, i, amplitude, probability*100)
		}
	}
}
