package visualization_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/visualization"
)

func TestBlochVectorFromQubit(t *testing.T) {
	invSqrt2 := 1 / math.Sqrt2

	cases := []struct {
		name     string
		alpha    complex128
		beta     complex128
		expected visualization.BlochVector
	}{
		{
			name:  "zero",
			alpha: 1,
			beta:  0,
			expected: visualization.BlochVector{
				X: 0,
				Y: 0,
				Z: 1,
			},
		},
		{
			name:  "one",
			alpha: 0,
			beta:  1,
			expected: visualization.BlochVector{
				X: 0,
				Y: 0,
				Z: -1,
			},
		},
		{
			name:  "plus",
			alpha: complex(invSqrt2, 0),
			beta:  complex(invSqrt2, 0),
			expected: visualization.BlochVector{
				X: 1,
				Y: 0,
				Z: 0,
			},
		},
		{
			name:  "plus-i",
			alpha: complex(invSqrt2, 0),
			beta:  complex(0, invSqrt2),
			expected: visualization.BlochVector{
				X: 0,
				Y: 1,
				Z: 0,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			q, err := qubit.NewWithValues(tt.alpha, tt.beta)
			if err != nil {
				t.Fatalf("NewWithValues error: %v", err)
			}

			got := visualization.BlochVectorFromQubit(q)
			if !nearEqual(got.X, tt.expected.X) || !nearEqual(got.Y, tt.expected.Y) || !nearEqual(got.Z, tt.expected.Z) {
				t.Fatalf("Bloch vector mismatch: got=%+v want=%+v", got, tt.expected)
			}
		})
	}
}

func nearEqual(a, b float64) bool {
	const tol = 1e-10
	return math.Abs(a-b) < tol
}
