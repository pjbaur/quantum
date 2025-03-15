package main

import (
	"math"
	"math/cmplx"
	"testing"
)

const tolerance = 1e-10 // Small tolerance for floating-point comparisons

// TestNewQubit checks if a new qubit starts in |0> state
func TestNewQubit(t *testing.T) {
	q := NewQubit()
	if real(q.Alpha) != 1.0 || imag(q.Alpha) != 0.0 {
		t.Errorf("NewQubit Alpha = %v, want 1 + 0i", q.Alpha)
	}
	if real(q.Beta) != 0.0 || imag(q.Beta) != 0.0 {
		t.Errorf("NewQubit Beta = %v, want 0 + 0i", q.Beta)
	}
}

// TestApplyHadamardFromZero tests Hadamard on |0>
func TestApplyHadamardFromZero(t *testing.T) {
	q := NewQubit()
	q.ApplyHadamard()

	expected := 1.0 / math.Sqrt(2)
	if math.Abs(real(q.Alpha)-expected) > tolerance || imag(q.Alpha) != 0.0 {
		t.Errorf("After Hadamard Alpha = %v, want %f + 0i", q.Alpha, expected)
	}
	if math.Abs(real(q.Beta)-expected) > tolerance || imag(q.Beta) != 0.0 {
		t.Errorf("After Hadamard Beta = %v, want %f + 0i", q.Beta, expected)
	}

	// Check normalization: |alpha|^2 + |beta|^2 = 1
	norm := cmplx.Abs(q.Alpha)*cmplx.Abs(q.Alpha) + cmplx.Abs(q.Beta)*cmplx.Abs(q.Beta)
	if math.Abs(norm-1.0) > tolerance {
		t.Errorf("Normalization failed: %f, want 1.0", norm)
	}
}

// TestApplyHadamardFromOne tests Hadamard on |1>
func TestApplyHadamardFromOne(t *testing.T) {
	q := &Qubit{Alpha: 0.0 + 0i, Beta: 1.0 + 0i} // Start in |1>
	q.ApplyHadamard()

	expectedAlpha := 1.0 / math.Sqrt(2)
	expectedBeta := -1.0 / math.Sqrt(2)
	if math.Abs(real(q.Alpha)-expectedAlpha) > tolerance || imag(q.Alpha) != 0.0 {
		t.Errorf("After Hadamard Alpha = %v, want %f + 0i", q.Alpha, expectedAlpha)
	}
	if math.Abs(real(q.Beta)-expectedBeta) > tolerance || imag(q.Beta) != 0.0 {
		t.Errorf("After Hadamard Beta = %v, want %f + 0i", q.Beta, expectedBeta)
	}

	// Check normalization
	norm := cmplx.Abs(q.Alpha)*cmplx.Abs(q.Alpha) + cmplx.Abs(q.Beta)*cmplx.Abs(q.Beta)
	if math.Abs(norm-1.0) > tolerance {
		t.Errorf("Normalization failed: %f, want 1.0", norm)
	}
}

// TestMeasure tests that measurement collapses and gives valid outcomes
func TestMeasure(t *testing.T) {
	// Test 1: Measure |0> should always give 0
	q1 := NewQubit()
	result1 := q1.Measure()
	if result1 != 0 {
		t.Errorf("Measure |0> = %d, want 0", result1)
	}
	if real(q1.Alpha) != 1.0 || real(q1.Beta) != 0.0 {
		t.Errorf("After measuring |0>: Alpha = %v, Beta = %v; want 1, 0", q1.Alpha, q1.Beta)
	}

	// Test 2: Measure after Hadamard gives 0 or 1, statistically ~50/50
	trials := 1000
	zeros := 0
	ones := 0
	for i := 0; i < trials; i++ {
		q := NewQubit()
		q.ApplyHadamard()
		result := q.Measure()
		if result == 0 {
			zeros++
		} else if result == 1 {
			ones++
		} else {
			t.Errorf("Measure gave invalid result: %d", result)
		}
	}

	// Check rough 50/50 distribution (within statistical noise)
	ratio := float64(zeros) / float64(trials)
	if math.Abs(ratio-0.5) > 0.1 { // Allow 10% deviation
		t.Errorf("Measurement ratio = %f, want ~0.5 (zeros=%d, ones=%d)", ratio, zeros, ones)
	}
}
