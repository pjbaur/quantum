package gates

import (
	"math"
	"testing"
)

// TestBuiltinGateGoldenValues locks the name and exact matrix of every
// built-in gate to the values the concrete types produced before the
// data-driven rewrite.
func TestBuiltinGateGoldenValues(t *testing.T) {
	cases := []struct {
		gate   *MatrixGate
		name   string
		matrix [][]complex128
	}{
		{NewHadamard(), "Hadamard", [][]complex128{
			{1.0 / complex(math.Sqrt(2), 0), 1.0 / complex(math.Sqrt(2), 0)},
			{1.0 / complex(math.Sqrt(2), 0), -1.0 / complex(math.Sqrt(2), 0)},
		}},
		{NewPauliX(), "PauliX", [][]complex128{
			{0, 1},
			{1, 0},
		}},
		{NewPauliY(), "PauliY", [][]complex128{
			{0, complex(0, -1)},
			{complex(0, 1), 0},
		}},
		{NewPauliZ(), "PauliZ", [][]complex128{
			{1, 0},
			{0, -1},
		}},
		{NewS(), "S", [][]complex128{
			{1, 0},
			{0, complex(0, 1)},
		}},
		{NewT(), "T", [][]complex128{
			{1, 0},
			{0, complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))},
		}},
		{NewCNOT(), "CNOT", [][]complex128{
			{1, 0, 0, 0},
			{0, 1, 0, 0},
			{0, 0, 0, 1},
			{0, 0, 1, 0},
		}},
		{NewSwap(), "SWAP", [][]complex128{
			{1, 0, 0, 0},
			{0, 0, 1, 0},
			{0, 1, 0, 0},
			{0, 0, 0, 1},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.gate.Name() != tc.name {
				t.Errorf("Name() = %q, want %q", tc.gate.Name(), tc.name)
			}
			got := tc.gate.Matrix()
			if len(got) != len(tc.matrix) {
				t.Fatalf("matrix size %d, want %d", len(got), len(tc.matrix))
			}
			for i := range tc.matrix {
				for j := range tc.matrix[i] {
					// Exact comparison on purpose: values must stay
					// bit-identical to the pre-rewrite matrices.
					if got[i][j] != tc.matrix[i][j] {
						t.Errorf("matrix[%d][%d] = %v, want %v", i, j, got[i][j], tc.matrix[i][j])
					}
				}
			}
		})
	}
}

// TestMustGatePanicsOnInvalidTable documents the built-in-table safety net:
// an invalid matrix in a built-in definition is a programmer error and must
// panic at construction rather than produce a half-valid gate.
func TestMustGatePanicsOnInvalidTable(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustGate with a 1x1 matrix did not panic")
		}
	}()
	mustGate("Broken", [][]complex128{{1}})
}
