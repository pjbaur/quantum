package algorithm

import (
	"math"
	"testing"
)

func TestDeutschJozsaOutputDistribution(t *testing.T) {
	tests := []struct {
		name            string
		numInputQubits  int
		oracle          Oracle
		wantZeroProbMin float64
		wantZeroProbMax float64
	}{
		{
			name:           "constant zero oracle",
			numInputQubits: 3,
			oracle: func(int) int {
				return 0
			},
			wantZeroProbMin: 1.0,
			wantZeroProbMax: 1.0,
		},
		{
			name:           "balanced parity oracle",
			numInputQubits: 3,
			oracle: func(input int) int {
				parity := 0
				for bit := input; bit > 0; bit >>= 1 {
					parity ^= bit & 1
				}
				return parity
			},
			wantZeroProbMin: 0.0,
			wantZeroProbMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := DeutschJozsa(tt.numInputQubits, tt.oracle)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			zeroProb := inputRegisterZeroProbability(s, tt.numInputQubits)
			if zeroProb < tt.wantZeroProbMin-1e-9 || zeroProb > tt.wantZeroProbMax+1e-9 {
				t.Fatalf("zero probability %.6f out of expected range", zeroProb)
			}
		})
	}
}

func TestGroverOutputDistribution(t *testing.T) {
	s, err := Grover(3, []int{5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prob := s.Probability(5); prob < 0.9 {
		t.Fatalf("expected marked state probability > 0.9, got %.4f", prob)
	}

	total := 0.0
	for i := 0; i < 8; i++ {
		total += s.Probability(i)
	}
	if math.Abs(total-1.0) > 1e-9 {
		t.Fatalf("state not normalized, sum %.6f", total)
	}
}

func inputRegisterZeroProbability(s interface {
	Probability(basisState int) float64
}, numInputQubits int) float64 {
	ancillaBit := 1 << numInputQubits
	return s.Probability(0) + s.Probability(ancillaBit)
}
