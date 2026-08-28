package quantum

import (
	"math"
	"testing"
)

func TestNormalizationToleranceValue(t *testing.T) {
	if NormalizationTolerance != 1e-10 {
		t.Fatalf("NormalizationTolerance = %g, want 1e-10", NormalizationTolerance)
	}
}

func TestIsNormalizedSum(t *testing.T) {
	// Boundary cases avoid exactly-representable edges on purpose: 1±tolerance
	// does not round-trip through float64 addition, so the tests pin the
	// contract (a window around 1, NaN outside it) rather than one bit pattern.
	tests := []struct {
		name string
		sum  float64
		want bool
	}{
		{name: "exactly one", sum: 1.0, want: true},
		{name: "comfortably above", sum: 1 + 1e-11, want: true},
		{name: "comfortably below", sum: 1 - 1e-11, want: true},
		{name: "too high", sum: 1 + 1e-9, want: false},
		{name: "too low", sum: 1 - 1e-9, want: false},
		{name: "zero", sum: 0, want: false},
		{name: "NaN fails every comparison", sum: math.NaN(), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNormalizedSum(tt.sum); got != tt.want {
				t.Errorf("IsNormalizedSum(%g) = %v, want %v", tt.sum, got, tt.want)
			}
		})
	}
}

func TestCheckNormalization(t *testing.T) {
	if err := CheckNormalization(1.0, 1.0); err != nil {
		t.Fatalf("normalized sum returned error: %v", err)
	}

	err := CheckNormalization(1.5, 1.0)
	nerr, ok := err.(*NormalizationError)
	if !ok {
		t.Fatalf("error type %T, want *NormalizationError", err)
	}
	if nerr.AttemptedSum != 1.5 || nerr.CurrentSum != 1.0 {
		t.Errorf("fields = (%g, %g), want (1.5, 1.0)", nerr.AttemptedSum, nerr.CurrentSum)
	}
}

func TestValidateAmplitudeVector(t *testing.T) {
	t.Run("length mismatch message is byte-identical to the backends", func(t *testing.T) {
		_, err := ValidateAmplitudeVector([]complex128{1}, 4)
		want := "values slice length 1 does not match state size 4"
		if err == nil || err.Error() != want {
			t.Fatalf("error = %v, want %q", err, want)
		}
	})

	t.Run("first non-finite amplitude wins", func(t *testing.T) {
		values := []complex128{0, 1, complex(0, math.NaN()), complex(math.Inf(1), 0)}
		_, err := ValidateAmplitudeVector(values, 4)
		nf, ok := err.(*NonFiniteAmplitudeError)
		if !ok {
			t.Fatalf("error type %T, want *NonFiniteAmplitudeError", err)
		}
		if nf.BasisState != 2 {
			t.Errorf("BasisState = %d, want 2 (first offender)", nf.BasisState)
		}
	})

	t.Run("length check beats finite check", func(t *testing.T) {
		_, err := ValidateAmplitudeVector([]complex128{complex(math.NaN(), 0)}, 4)
		if err == nil || err.Error() != "values slice length 1 does not match state size 4" {
			t.Fatalf("error = %v, want the length message", err)
		}
	})

	t.Run("returns the probability sum", func(t *testing.T) {
		// 0.5² is exact in binary, so the sum is exactly 1.
		values := []complex128{0.5, 0.5, 0.5, 0.5}
		sum, err := ValidateAmplitudeVector(values, 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sum != 1.0 {
			t.Errorf("sum = %g, want 1", sum)
		}
	})
}
