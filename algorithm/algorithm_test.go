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

// Phase 3.1: Negative path tests for DeutschJozsa

func TestDeutschJozsaNegativePaths(t *testing.T) {
	tests := []struct {
		name           string
		numInputQubits int
		oracle         Oracle
		wantErr        string
	}{
		{
			name:           "zero qubits",
			numInputQubits: 0,
			oracle: func(int) int {
				return 0
			},
			wantErr: "numInputQubits must be positive",
		},
		{
			name:           "negative qubits",
			numInputQubits: -1,
			oracle: func(int) int {
				return 0
			},
			wantErr: "numInputQubits must be positive",
		},
		{
			name:           "nil oracle",
			numInputQubits: 2,
			oracle:         nil,
			wantErr:        "oracle must not be nil",
		},
		{
			name:           "oracle returns invalid value 2",
			numInputQubits: 2,
			oracle: func(int) int {
				return 2
			},
			wantErr: "oracle returned 2 for input 0",
		},
		{
			name:           "oracle returns invalid negative value",
			numInputQubits: 2,
			oracle: func(int) int {
				return -1
			},
			wantErr: "oracle returned -1 for input 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeutschJozsa(tt.numInputQubits, tt.oracle)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

// Phase 3.1: Negative path tests for Grover

func TestGroverNegativePaths(t *testing.T) {
	tests := []struct {
		name      string
		numQubits int
		marked    []int
		wantErr   string
	}{
		{
			name:      "zero qubits",
			numQubits: 0,
			marked:    []int{0},
			wantErr:   "numQubits must be positive",
		},
		{
			name:      "negative qubits",
			numQubits: -1,
			marked:    []int{0},
			wantErr:   "numQubits must be positive",
		},
		{
			name:      "empty marked set",
			numQubits: 2,
			marked:    []int{},
			wantErr:   "marked set must not be empty",
		},
		{
			name:      "nil marked set",
			numQubits: 2,
			marked:    nil,
			wantErr:   "marked set must not be empty",
		},
		{
			name:      "marked state negative",
			numQubits: 2,
			marked:    []int{-1},
			wantErr:   "marked state -1 out of range",
		},
		{
			name:      "marked state exceeds total states",
			numQubits: 2,
			marked:    []int{4},
			wantErr:   "marked state 4 out of range",
		},
		{
			name:      "marked state way out of range",
			numQubits: 3,
			marked:    []int{100},
			wantErr:   "marked state 100 out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Grover(tt.numQubits, tt.marked)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
