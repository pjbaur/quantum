package gates

import (
	"math/cmplx"
	"testing"
)

func TestNewMatrixGateValidation(t *testing.T) {
	valid2x2 := [][]complex128{{0, 1}, {1, 0}}
	cases := []struct {
		name     string
		gateName string
		matrix   [][]complex128
		wantErr  bool
	}{
		{"valid 2x2", "X", valid2x2, false},
		{"valid 4x4", "CZ", [][]complex128{
			{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1},
		}, false},
		{"empty name", "", valid2x2, true},
		{"nil matrix", "G", nil, true},
		{"1x1 matrix", "G", [][]complex128{{1}}, true},
		{"non-square", "G", [][]complex128{{1, 0}, {0}}, true},
		{"non power of two", "G", [][]complex128{
			{1, 0, 0}, {0, 1, 0}, {0, 0, 1},
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gate, err := NewMatrixGate(tc.gateName, tc.matrix)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NewMatrixGate(%q) succeeded, want error", tc.gateName)
				}
				if gate != nil {
					t.Errorf("NewMatrixGate(%q) returned non-nil gate with error", tc.gateName)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewMatrixGate(%q) failed: %v", tc.gateName, err)
			}
			if gate.Name() != tc.gateName {
				t.Errorf("Name() = %q, want %q", gate.Name(), tc.gateName)
			}
		})
	}
}

func TestMatrixGateNumQubits(t *testing.T) {
	twoByTwo, err := NewMatrixGate("X", [][]complex128{{0, 1}, {1, 0}})
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}
	if got := twoByTwo.NumQubits(); got != 1 {
		t.Errorf("2x2 NumQubits() = %d, want 1", got)
	}

	fourByFour, err := NewMatrixGate("CZ", [][]complex128{
		{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1},
	})
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}
	if got := fourByFour.NumQubits(); got != 2 {
		t.Errorf("4x4 NumQubits() = %d, want 2", got)
	}
}

func TestMatrixGateMatrixIsDeepCopy(t *testing.T) {
	input := [][]complex128{{0, 1}, {1, 0}}
	gate, err := NewMatrixGate("X", input)
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}

	// Mutating the caller's input after construction must not affect the gate.
	input[0][0] = 99
	if got := gate.Matrix()[0][0]; got != 0 {
		t.Errorf("gate matrix affected by input mutation: got %v, want 0", got)
	}

	// Mutating a returned matrix must not affect later calls.
	first := gate.Matrix()
	first[0][1] = 42
	if got := gate.Matrix()[0][1]; cmplx.Abs(got-1) > 0 {
		t.Errorf("gate matrix affected by returned-slice mutation: got %v, want 1", got)
	}
}
