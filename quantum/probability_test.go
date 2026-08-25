package quantum

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestProbability(t *testing.T) {
	tests := []struct {
		name string
		amp  complex128
		want float64
	}{
		{name: "zero amplitude", amp: 0, want: 0},
		{name: "unit real amplitude", amp: 1, want: 1},
		{name: "unit imaginary amplitude", amp: 1i, want: 1},
		{name: "negative real amplitude", amp: -1, want: 1},
		{name: "3+4i", amp: 3 + 4i, want: 25},
		{name: "equal superposition", amp: complex(1/math.Sqrt2, 0), want: 0.5},
		{name: "mixed real and imaginary", amp: complex(0.6, 0.8), want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Probability(tt.amp); math.Abs(got-tt.want) > 1e-15 {
				t.Errorf("Probability(%v) = %v, want %v", tt.amp, got, tt.want)
			}
		})
	}
}

// TestProbabilityMatchesAbsSquared pins the helper to the expression it
// replaced across the backends: |c|² computed via cmplx.Abs. The two agree
// to within floating-point rounding for normalized amplitudes.
func TestProbabilityMatchesAbsSquared(t *testing.T) {
	amps := []complex128{
		0,
		1,
		1i,
		complex(0.6, -0.8),
		complex(1/math.Sqrt2, 1/math.Sqrt2),
		complex(-0.3, 0.4),
		complex(1e-8, 1e-8),
	}

	for _, amp := range amps {
		want := math.Pow(cmplx.Abs(amp), 2)
		got := Probability(amp)
		if math.Abs(got-want) > 1e-15 {
			t.Errorf("Probability(%v) = %v, want %v (from math.Pow(cmplx.Abs, 2))", amp, got, want)
		}
	}
}
