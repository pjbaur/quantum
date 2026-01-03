package gates

import (
	"math/cmplx"
	"testing"
)

func TestComposeMatricesHadamardSquared(t *testing.T) {
	h := NewHadamard().Matrix()
	composed, err := ComposeMatrices(h, h)
	if err != nil {
		t.Fatalf("compose matrices: %v", err)
	}

	identity := [][]complex128{
		{1, 0},
		{0, 1},
	}

	if !matrixClose(composed, identity, 1e-12) {
		t.Fatalf("expected hadamard squared to be identity: %#v", composed)
	}
}

func TestTensorProduct(t *testing.T) {
	x := NewPauliX().Matrix()
	z := NewPauliZ().Matrix()

	product, err := TensorProduct(x, z)
	if err != nil {
		t.Fatalf("tensor product: %v", err)
	}

	expected := [][]complex128{
		{0, 0, 1, 0},
		{0, 0, 0, -1},
		{1, 0, 0, 0},
		{0, -1, 0, 0},
	}

	if !matrixClose(product, expected, 1e-12) {
		t.Fatalf("unexpected tensor product: %#v", product)
	}
}

func matrixClose(left, right [][]complex128, tol float64) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if len(left[i]) != len(right[i]) {
			return false
		}
		for j := range left[i] {
			if cmplx.Abs(left[i][j]-right[i][j]) > tol {
				return false
			}
		}
	}
	return true
}
