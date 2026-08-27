package quantum

import (
	"math"
	"testing"
)

func TestFidelityKnownValues(t *testing.T) {
	v := invSqrt2()
	cases := []struct {
		name string
		a, b []complex128
		want float64
	}{
		{"identical |0>", []complex128{1, 0}, []complex128{1, 0}, 1},
		{"orthogonal |0> |1>", []complex128{1, 0}, []complex128{0, 1}, 0},
		{"|0> vs |+>", []complex128{1, 0}, []complex128{v, v}, 0.5},
		{"|+> vs |->_ symmetric half", []complex128{v, v}, []complex128{v, -v}, 0},
		{"|+_y> vs |+>", []complex128{v, complex(0, real(v))}, []complex128{v, v}, 0.5},
		{"global phase ignored", []complex128{1, 0}, []complex128{-1, 0}, 1},
	}

	for _, tc := range cases {
		a := &fakeSampleState{numQubits: 1, amplitudes: tc.a}
		b := &fakeSampleState{numQubits: 1, amplitudes: tc.b}
		got, err := Fidelity(a, b)
		if err != nil {
			t.Errorf("%s: Fidelity error: %v", tc.name, err)
			continue
		}
		if math.Abs(got-tc.want) > 1e-10 {
			t.Errorf("%s: Fidelity = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestFidelityIsSymmetric(t *testing.T) {
	v := invSqrt2()
	a := &fakeSampleState{numQubits: 2, amplitudes: []complex128{v, 0, 0, v}}
	b := &fakeSampleState{numQubits: 2, amplitudes: []complex128{0, v, v, 0}}

	ab, err := Fidelity(a, b)
	if err != nil {
		t.Fatalf("Fidelity(a,b) error: %v", err)
	}
	ba, err := Fidelity(b, a)
	if err != nil {
		t.Fatalf("Fidelity(b,a) error: %v", err)
	}
	if math.Abs(ab-ba) > 1e-12 {
		t.Errorf("Fidelity not symmetric: %v vs %v", ab, ba)
	}
	if ab != 0 {
		t.Errorf("Fidelity(|Phi+>, |Psi+>) = %v, want 0", ab)
	}
}

func TestFidelityRejectsNilStates(t *testing.T) {
	s := &fakeSampleState{numQubits: 1, amplitudes: []complex128{1, 0}}

	if _, err := Fidelity(nil, s); err == nil || err.Error() != "first state must not be nil" {
		t.Errorf("Fidelity(nil, s) error = %v, want \"first state must not be nil\"", err)
	}
	if _, err := Fidelity(s, nil); err == nil || err.Error() != "second state must not be nil" {
		t.Errorf("Fidelity(s, nil) error = %v, want \"second state must not be nil\"", err)
	}

	var typedNil *fakeSampleState
	if _, err := Fidelity(typedNil, s); err == nil || err.Error() != "first state must not be nil" {
		t.Errorf("Fidelity(typed nil, s) error = %v, want \"first state must not be nil\"", err)
	}
}

func TestFidelityRejectsQubitCountMismatch(t *testing.T) {
	a := &fakeSampleState{numQubits: 2, amplitudes: []complex128{1, 0, 0, 0}}
	b := &fakeSampleState{numQubits: 1, amplitudes: []complex128{1, 0}}

	_, err := Fidelity(a, b)
	e, ok := err.(*IncompatibleQubitCountError)
	if !ok {
		t.Fatalf("Fidelity error = %T (%v), want *IncompatibleQubitCountError", err, err)
	}
	if e.Expected != 2 || e.Actual != 1 {
		t.Errorf("IncompatibleQubitCountError = expected %d actual %d, want 2 and 1", e.Expected, e.Actual)
	}
}

func TestFidelityRejectsUnnormalizedStates(t *testing.T) {
	good := &fakeSampleState{numQubits: 1, amplitudes: []complex128{1, 0}}

	badA := &fakeSampleState{numQubits: 1, amplitudes: []complex128{2, 0}}
	_, err := Fidelity(badA, good)
	if _, ok := err.(*UnnormalizedStateError); !ok {
		t.Errorf("unnormalized first: error = %T (%v), want *UnnormalizedStateError", err, err)
	}

	badB := &fakeSampleState{numQubits: 1, amplitudes: []complex128{0, 2}}
	_, err = Fidelity(good, badB)
	if _, ok := err.(*UnnormalizedStateError); !ok {
		t.Errorf("unnormalized second: error = %T (%v), want *UnnormalizedStateError", err, err)
	}
}
