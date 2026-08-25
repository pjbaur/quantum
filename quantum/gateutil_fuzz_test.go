package quantum

import (
	"errors"
	"math"
	"math/bits"
	"testing"
)

// maxFuzzDim caps the generated matrix dimension so a fuzzed row length
// cannot ask for a multi-megabyte allocation.
const maxFuzzDim = 64

// fuzzGate is a gate whose matrix shape, matrix content, and declared qubit
// count all come from the fuzzer, so one target covers both of
// GateQubitCount's paths: the QubitCounter fast path and the matrix
// inspection it falls back to.
type fuzzGate struct {
	matrix   [][]complex128
	declared int
}

func (g *fuzzGate) Name() string           { return "Fuzz" }
func (g *fuzzGate) Matrix() [][]complex128 { return g.matrix }
func (g *fuzzGate) NumQubits() int         { return g.declared }

// FuzzGateQubitCount checks GateQubitCount against an independent statement
// of its contract: a matrix describes a gate only when it is square, at
// least 2x2, and a power of two on a side, and then the count is log2(size).
//
// rowLens gives the matrix its shape — one entry per row, holding that row's
// length — so jagged, rectangular, and empty matrices are all reachable.
// fill is written into every cell to confirm that content, non-finite values
// included, never influences the count.
func FuzzGateQubitCount(f *testing.F) {
	// Valid shapes: 1-qubit (identity, H, ...), CNOT, Toffoli.
	f.Add([]byte{2, 2}, int8(0), 1.0)
	f.Add([]byte{4, 4, 4, 4}, int8(0), 0.5)
	f.Add([]byte{8, 8, 8, 8, 8, 8, 8, 8}, int8(0), 0.0)
	// 1x1: a 2^0 "gate" acting on no qubits at all.
	f.Add([]byte{1}, int8(0), 1.0)
	// Empty.
	f.Add([]byte{}, int8(0), 0.0)
	// Dimensions that are not a power of two.
	f.Add([]byte{3, 3, 3}, int8(0), 1.0)
	f.Add([]byte{5, 5, 5, 5, 5}, int8(0), 1.0)
	f.Add([]byte{6, 6, 6, 6, 6, 6}, int8(0), 1.0)
	// Non-square: too many rows, and jagged.
	f.Add([]byte{2, 2, 2}, int8(0), 1.0)
	f.Add([]byte{4, 2, 4, 4}, int8(0), 1.0)
	// Non-finite cells.
	f.Add([]byte{2, 2}, int8(0), math.NaN())
	f.Add([]byte{2, 2}, int8(0), math.Inf(1))
	// QubitCounter fast path, including a declaration that contradicts the
	// matrix (a positive count is trusted by contract) and a non-positive
	// one that must fall back to the matrix.
	f.Add([]byte{2, 2}, int8(3), 1.0)
	f.Add([]byte{2, 2}, int8(-1), 1.0)

	f.Fuzz(func(t *testing.T, rowLens []byte, declared int8, fill float64) {
		if len(rowLens) > maxFuzzDim {
			rowLens = rowLens[:maxFuzzDim]
		}
		matrix := make([][]complex128, len(rowLens))
		for i, n := range rowLens {
			row := make([]complex128, int(n)%(maxFuzzDim+1))
			for j := range row {
				row[j] = complex(fill, fill)
			}
			matrix[i] = row
		}

		gate := &fuzzGate{matrix: matrix, declared: int(declared)}
		got, err := GateQubitCount(gate)

		if declared > 0 {
			if err != nil {
				t.Fatalf("declared %d qubits: unexpected error: %v", declared, err)
			}
			if got != int(declared) {
				t.Fatalf("declared %d qubits: GateQubitCount = %d, want the declared count", declared, got)
			}
			return
		}

		size := len(matrix)
		square := true
		for _, row := range matrix {
			if len(row) != size {
				square = false
				break
			}
		}
		// A gate acts on at least one qubit, so the smallest valid matrix
		// is 2x2; 1x1 is a power of two but describes no qubit at all.
		valid := size >= 2 && square && size&(size-1) == 0

		if !valid {
			if err == nil {
				t.Fatalf("shape %v: GateQubitCount = %d with no error, want a rejection", rowLens, got)
			}
			var matrixErr *InvalidGateMatrixError
			if !errors.As(err, &matrixErr) {
				t.Fatalf("shape %v: error is %T (%v), want *InvalidGateMatrixError", rowLens, err, err)
			}
			if got != 0 {
				t.Fatalf("shape %v: GateQubitCount = %d alongside error %v, want 0", rowLens, got, err)
			}
			return
		}

		if err != nil {
			t.Fatalf("size %d: unexpected error: %v", size, err)
		}
		want := bits.Len(uint(size)) - 1
		if got != want {
			t.Fatalf("size %d: GateQubitCount = %d, want %d", size, got, want)
		}
		if got < 1 || 1<<got != size {
			t.Fatalf("size %d: GateQubitCount = %d, want a positive count with 2^count == size", size, got)
		}
	})
}
