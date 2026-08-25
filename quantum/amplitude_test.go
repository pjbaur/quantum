package quantum

import (
	"math"
	"testing"
)

func TestIsFiniteAmplitude(t *testing.T) {
	tests := []struct {
		name string
		amp  complex128
		want bool
	}{
		{name: "zero", amp: 0, want: true},
		{name: "unit real", amp: 1, want: true},
		{name: "unit imaginary", amp: 1i, want: true},
		{name: "mixed", amp: complex(0.6, -0.8), want: true},
		{name: "largest finite float", amp: complex(math.MaxFloat64, -math.MaxFloat64), want: true},
		{name: "smallest subnormal", amp: complex(math.SmallestNonzeroFloat64, 0), want: true},
		{name: "NaN real part", amp: complex(math.NaN(), 0), want: false},
		{name: "NaN imaginary part", amp: complex(0, math.NaN()), want: false},
		{name: "positive infinite real part", amp: complex(math.Inf(1), 0), want: false},
		{name: "negative infinite imaginary part", amp: complex(0, math.Inf(-1)), want: false},
		// cmplx.IsNaN reports false when a part is infinite, so this pair
		// is only rejected because the infinity check runs as well.
		{name: "infinity and NaN together", amp: complex(math.Inf(1), math.NaN()), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFiniteAmplitude(tt.amp); got != tt.want {
				t.Errorf("IsFiniteAmplitude(%v) = %v, want %v", tt.amp, got, tt.want)
			}
		})
	}
}

func TestNonFiniteAmplitudeError(t *testing.T) {
	err := &NonFiniteAmplitudeError{BasisState: 3, Value: complex(math.NaN(), 0)}
	want := "amplitude for basis state 3 is not finite: (NaN+0i)"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
