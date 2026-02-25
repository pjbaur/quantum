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

func TestFormatBlochVector(t *testing.T) {
	cases := []struct {
		name      string
		vector    visualization.BlochVector
		precision int
		expected  string
	}{
		{
			name: "default_precision",
			vector: visualization.BlochVector{
				X: 0.123456789,
				Y: 0.987654321,
				Z: 0.555555555,
			},
			precision: 4,
			expected:  "x=0.1235 y=0.9877 z=0.5556",
		},
		{
			name: "custom_precision_2",
			vector: visualization.BlochVector{
				X: 1.234,
				Y: 2.345,
				Z: 3.456,
			},
			precision: 2,
			expected:  "x=1.23 y=2.35 z=3.46",
		},
		{
			name: "custom_precision_6",
			vector: visualization.BlochVector{
				X: 0.5,
				Y: 0.5,
				Z: 0.7071,
			},
			precision: 6,
			expected:  "x=0.500000 y=0.500000 z=0.707100",
		},
		{
			name: "zero_precision_defaults_to_4",
			vector: visualization.BlochVector{
				X: 0.111111,
				Y: 0.222222,
				Z: 0.333333,
			},
			precision: 0,
			expected:  "x=0.1111 y=0.2222 z=0.3333",
		},
		{
			name: "negative_precision_defaults_to_4",
			vector: visualization.BlochVector{
				X: 0.111111,
				Y: 0.222222,
				Z: 0.333333,
			},
			precision: -1,
			expected:  "x=0.1111 y=0.2222 z=0.3333",
		},
		{
			name: "negative_values",
			vector: visualization.BlochVector{
				X: -0.5,
				Y: -0.7071,
				Z: -1.0,
			},
			precision: 2,
			expected:  "x=-0.50 y=-0.71 z=-1.00",
		},
		{
			name: "zero_vector",
			vector: visualization.BlochVector{
				X: 0,
				Y: 0,
				Z: 0,
			},
			precision: 2,
			expected:  "x=0.00 y=0.00 z=0.00",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := visualization.FormatBlochVector(tt.vector, tt.precision)
			if got != tt.expected {
				t.Errorf("FormatBlochVector() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBlochCSV(t *testing.T) {
	cases := []struct {
		name      string
		vector    visualization.BlochVector
		precision int
		expected  string
	}{
		{
			name: "default_precision",
			vector: visualization.BlochVector{
				X: 0.123456789,
				Y: 0.987654321,
				Z: 0.555555555,
			},
			precision: 4,
			expected:  "0.1235,0.9877,0.5556",
		},
		{
			name: "custom_precision_2",
			vector: visualization.BlochVector{
				X: 1.234,
				Y: 2.345,
				Z: 3.456,
			},
			precision: 2,
			expected:  "1.23,2.35,3.46",
		},
		{
			name: "custom_precision_6",
			vector: visualization.BlochVector{
				X: 0.5,
				Y: 0.5,
				Z: 0.7071,
			},
			precision: 6,
			expected:  "0.500000,0.500000,0.707100",
		},
		{
			name: "zero_precision_defaults_to_4",
			vector: visualization.BlochVector{
				X: 0.111111,
				Y: 0.222222,
				Z: 0.333333,
			},
			precision: 0,
			expected:  "0.1111,0.2222,0.3333",
		},
		{
			name: "negative_precision_defaults_to_4",
			vector: visualization.BlochVector{
				X: 0.111111,
				Y: 0.222222,
				Z: 0.333333,
			},
			precision: -1,
			expected:  "0.1111,0.2222,0.3333",
		},
		{
			name: "negative_values",
			vector: visualization.BlochVector{
				X: -0.5,
				Y: -0.7071,
				Z: -1.0,
			},
			precision: 2,
			expected:  "-0.50,-0.71,-1.00",
		},
		{
			name: "zero_vector",
			vector: visualization.BlochVector{
				X: 0,
				Y: 0,
				Z: 0,
			},
			precision: 2,
			expected:  "0.00,0.00,0.00",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := visualization.BlochCSV(tt.vector, tt.precision)
			if got != tt.expected {
				t.Errorf("BlochCSV() = %q, want %q", got, tt.expected)
			}
		})
	}
}
