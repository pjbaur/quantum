package examples

import (
	"fmt"
	"strings"

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

// PrintRule prints a single line of width `=` characters, the horizontal
// rule used to open and close demo section banners.
func PrintRule(width int) {
	fmt.Println(strings.Repeat("=", width))
}

// PrintBanner prints a demo section banner: a rule of the given width, then
// each of titles on its own line, then a matching closing rule. Each title
// is printed exactly as passed — including any leading spaces callers use
// to visually center it — so output stays byte-for-byte identical to the
// hand-written banners this replaces.
func PrintBanner(width int, titles ...string) {
	PrintRule(width)
	for _, title := range titles {
		fmt.Println(title)
	}
	PrintRule(width)
}
