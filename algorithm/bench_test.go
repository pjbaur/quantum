package algorithm

import (
	"fmt"
	"testing"
)

func BenchmarkGrover(b *testing.B) {
	for _, numQubits := range []int{4, 6, 8, 10, 12} {
		b.Run(fmt.Sprintf("Qubits=%d", numQubits), func(b *testing.B) {
			marked := []int{0} // Mark the first basis state
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Grover(numQubits, marked)
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
				_, err := Grover(numQubits, marked)
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
		b.Run(fmt.Sprintf("Qubits=%d/Constant", numQubits), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := DeutschJozsa(numQubits, constantZero)
				if err != nil {
					b.Fatalf("DeutschJozsa failed: %v", err)
				}
			}
		})

		b.Run(fmt.Sprintf("Qubits=%d/Balanced", numQubits), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := DeutschJozsa(numQubits, balancedParity)
				if err != nil {
					b.Fatalf("DeutschJozsa failed: %v", err)
				}
			}
		})
	}
}
