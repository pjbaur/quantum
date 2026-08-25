package qubit

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/quantum"
)

const invSqrt2 = 0.7071067811865476

func TestNewStartsInZeroState(t *testing.T) {
	q := New()

	if q.Alpha() != 1 || q.Beta() != 0 {
		t.Fatalf("expected |0⟩ state, got alpha=%v beta=%v", q.Alpha(), q.Beta())
	}
	if p := q.Probability0(); p != 1 {
		t.Fatalf("expected Probability0 == 1, got %v", p)
	}
	if p := q.Probability1(); p != 0 {
		t.Fatalf("expected Probability1 == 0, got %v", p)
	}
	if !q.IsNormalized() {
		t.Fatal("new qubit must be normalized")
	}
}

func TestNewWithValues(t *testing.T) {
	tests := []struct {
		name    string
		alpha   complex128
		beta    complex128
		wantErr bool
	}{
		{name: "basis one", alpha: 0, beta: 1},
		{name: "equal superposition", alpha: complex(invSqrt2, 0), beta: complex(invSqrt2, 0)},
		{name: "complex phase", alpha: complex(0, invSqrt2), beta: complex(invSqrt2, 0)},
		{name: "unnormalized", alpha: 1, beta: 1, wantErr: true},
		{name: "zero vector", alpha: 0, beta: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewWithValues(tt.alpha, tt.beta)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var normErr *quantum.NormalizationError
				if !errors.As(err, &normErr) {
					t.Fatalf("expected NormalizationError, got %T: %v", err, err)
				}
				if q != nil {
					t.Fatalf("expected nil qubit on error, got %v", q)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if q.Alpha() != tt.alpha || q.Beta() != tt.beta {
				t.Fatalf("amplitudes not stored: got alpha=%v beta=%v", q.Alpha(), q.Beta())
			}
		})
	}
}

func TestSetToleranceBoundary(t *testing.T) {
	// Set accepts drift up to 1e-6 (looser than the backends' 1e-10).
	tests := []struct {
		name    string
		drift   float64
		wantErr bool
	}{
		{name: "within tolerance", drift: 9e-7},
		{name: "beyond tolerance", drift: 2e-6, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := New()
			alpha := complex(math.Sqrt(1+tt.drift), 0)
			err := q.Set(alpha, 0)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var normErr *quantum.NormalizationError
				if !errors.As(err, &normErr) {
					t.Fatalf("expected NormalizationError, got %T: %v", err, err)
				}
				// Rejected Set must leave the qubit untouched.
				if q.Alpha() != 1 || q.Beta() != 0 {
					t.Fatalf("failed Set mutated qubit: alpha=%v beta=%v", q.Alpha(), q.Beta())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if q.Alpha() != alpha {
				t.Fatalf("alpha not stored: got %v", q.Alpha())
			}
		})
	}
}

func TestIsNormalizedStricterThanSet(t *testing.T) {
	// Drift between IsNormalized's 1e-10 and Set's 1e-6 is accepted by Set
	// but reported as not normalized. Documents the current inconsistency
	// (see specs/architecture/architectural-issues.md).
	q := New()
	if err := q.Set(complex(math.Sqrt(1+1e-7), 0), 0); err != nil {
		t.Fatalf("Set within its own tolerance failed: %v", err)
	}
	if q.IsNormalized() {
		t.Fatal("expected IsNormalized to reject 1e-7 drift")
	}
}

func TestProbabilitiesOfSuperposition(t *testing.T) {
	q, err := NewWithValues(complex(invSqrt2, 0), complex(0, -invSqrt2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p := q.Probability0(); math.Abs(p-0.5) > 1e-12 {
		t.Fatalf("expected Probability0 ≈ 0.5, got %v", p)
	}
	if p := q.Probability1(); math.Abs(p-0.5) > 1e-12 {
		t.Fatalf("expected Probability1 ≈ 0.5, got %v", p)
	}
}

func TestMeasureDeterministicStates(t *testing.T) {
	zero := New()
	for i := 0; i < 100; i++ {
		if got := zero.Measure(); got != 0 {
			t.Fatalf("measuring |0⟩ returned %d", got)
		}
	}

	one, err := NewWithValues(0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 100; i++ {
		if got := one.Measure(); got != 1 {
			t.Fatalf("measuring |1⟩ returned %d", got)
		}
	}
}

func TestMeasureCollapsesState(t *testing.T) {
	q, err := NewWithValues(complex(invSqrt2, 0), complex(invSqrt2, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := q.Measure()
	if result != 0 && result != 1 {
		t.Fatalf("Measure returned %d, want 0 or 1", result)
	}

	// Collapse must land exactly on the measured basis state.
	wantAlpha, wantBeta := complex128(1), complex128(0)
	if result == 1 {
		wantAlpha, wantBeta = 0, 1
	}
	if q.Alpha() != wantAlpha || q.Beta() != wantBeta {
		t.Fatalf("collapse to |%d⟩ left alpha=%v beta=%v", result, q.Alpha(), q.Beta())
	}
	if !q.IsNormalized() {
		t.Fatal("collapsed state must be normalized")
	}

	// Repeated measurement of a collapsed state is stable.
	for i := 0; i < 100; i++ {
		if got := q.Measure(); got != result {
			t.Fatalf("re-measurement returned %d, want %d", got, result)
		}
	}
}

func TestMeasureRoughDistribution(t *testing.T) {
	// Coarse statistical check that Measure uses the amplitudes at all.
	// Bounds are ±6σ wide; deterministic assertions need injectable
	// randomness first (tracked separately).
	const trials = 1000
	ones := 0
	for i := 0; i < trials; i++ {
		q, err := NewWithValues(complex(invSqrt2, 0), complex(invSqrt2, 0))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ones += q.Measure()
	}
	if ones < 400 || ones > 600 {
		t.Fatalf("measured 1 in %d/%d trials, expected roughly half", ones, trials)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	original, err := NewWithValues(complex(invSqrt2, 0), complex(invSqrt2, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cloned, ok := original.Clone().(*Qubit)
	if !ok {
		t.Fatalf("Clone returned %T, want *Qubit", original.Clone())
	}
	if cloned == original {
		t.Fatal("Clone returned the same instance")
	}
	if cloned.Alpha() != original.Alpha() || cloned.Beta() != original.Beta() {
		t.Fatalf("clone amplitudes differ: alpha=%v beta=%v", cloned.Alpha(), cloned.Beta())
	}

	// Mutating the original must not affect the clone, and vice versa.
	if err := original.Set(0, 1); err != nil {
		t.Fatalf("Set on original failed: %v", err)
	}
	if cloned.Alpha() != complex(invSqrt2, 0) || cloned.Beta() != complex(invSqrt2, 0) {
		t.Fatal("mutating original changed the clone")
	}
	if err := cloned.Set(1, 0); err != nil {
		t.Fatalf("Set on clone failed: %v", err)
	}
	if original.Alpha() != 0 || original.Beta() != 1 {
		t.Fatal("mutating clone changed the original")
	}
}

func TestProbabilitiesSumToOne(t *testing.T) {
	q, err := NewWithValues(complex(0.6, 0), complex(0, 0.8))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sum := q.Probability0() + q.Probability1()
	if math.Abs(sum-1.0) > 1e-12 {
		t.Fatalf("probabilities sum to %v, want 1", sum)
	}
	if math.Abs(cmplx.Abs(q.Beta())-0.8) > 1e-12 {
		t.Fatalf("|beta| = %v, want 0.8", cmplx.Abs(q.Beta()))
	}
}
