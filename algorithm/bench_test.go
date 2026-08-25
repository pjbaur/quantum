package algorithm

import (
	"fmt"
	"testing"

	"github.com/pjbaur/quantum/state"
)

// newDenseState is the benchmark's backend. Allocating it inside the timed
// loop keeps these numbers comparable with the runs from before the algorithms
// took the state as an argument, when they allocated it themselves.
func newDenseState(b *testing.B, numQubits int) *state.State {
	b.Helper()

	s, err := state.New(numQubits)
	if err != nil {
		b.Fatalf("state.New(%d) failed: %v", numQubits, err)
	}
	return s
}

func BenchmarkGrover(b *testing.B) {
	for _, numQubits := range []int{4, 6, 8, 10, 12} {
		b.Run(fmt.Sprintf("Qubits=%d", numQubits), func(b *testing.B) {
			marked := []int{0} // Mark the first basis state
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Grover(newDenseState(b, numQubits), marked)
				if err != nil {
					b.Fatalf("Grover failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkGroverMultipleMarked(b *testing.B) {
	numQubits := 10
	for _, numMarked := range []int{1, 4, 16, 64} {
		b.Run(fmt.Sprintf("Marked=%d", numMarked), func(b *testing.B) {
			marked := make([]int, numMarked)
			for i := 0; i < numMarked; i++ {
				marked[i] = i
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Grover(newDenseState(b, numQubits), marked)
				if err != nil {
					b.Fatalf("Grover failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkDeutschJozsa(b *testing.B) {
	// Constant zero oracle
	constantZero := func(int) int { return 0 }
	// Balanced parity oracle
	balancedParity := func(input int) int {
		parity := 0
		for bit := input; bit > 0; bit >>= 1 {
			parity ^= bit & 1
		}
		return parity
	}

	for _, numQubits := range []int{4, 6, 8, 10, 12} {
		// numQubits counts the input register; the state adds the ancilla.
		totalQubits := numQubits + 1

		b.Run(fmt.Sprintf("Qubits=%d/Constant", numQubits), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := DeutschJozsa(newDenseState(b, totalQubits), constantZero)
				if err != nil {
					b.Fatalf("DeutschJozsa failed: %v", err)
				}
			}
		})

		b.Run(fmt.Sprintf("Qubits=%d/Balanced", numQubits), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := DeutschJozsa(newDenseState(b, totalQubits), balancedParity)
				if err != nil {
					b.Fatalf("DeutschJozsa failed: %v", err)
				}
			}
		})
	}
}
