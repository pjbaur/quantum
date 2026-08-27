package quantum

import (
	"math/cmplx"
	"testing"
)

func bellAmplitudes() []complex128 {
	v := invSqrt2()
	return []complex128{v, 0, 0, v}
}

func plusYAmplitudes() []complex128 {
	return []complex128{invSqrt2(), complex(0, 0.7071067811865476)}
}

func minusYAmplitudes() []complex128 {
	return []complex128{invSqrt2(), complex(0, -0.7071067811865476)}
}

func TestExpectationPauliAxesOnSingleQubit(t *testing.T) {
	cases := []struct {
		name string
		amps []complex128
		axis PauliAxis
		want float64
	}{
		{"Z on |0>", []complex128{1, 0}, PauliZ, 1},
		{"Z on |1>", []complex128{0, 1}, PauliZ, -1},
		{"Z on |+>", []complex128{invSqrt2(), invSqrt2()}, PauliZ, 0},
		{"X on |+>", []complex128{invSqrt2(), invSqrt2()}, PauliX, 1},
		{"X on |->", []complex128{invSqrt2(), -invSqrt2()}, PauliX, -1},
		{"X on |0>", []complex128{1, 0}, PauliX, 0},
		{"Y on |+_y>", plusYAmplitudes(), PauliY, 1},
		{"Y on |-_y>", minusYAmplitudes(), PauliY, -1},
		{"Y on |0>", []complex128{1, 0}, PauliY, 0},
		{"I on |0>", []complex128{1, 0}, PauliI, 1},
		{"I on |+>", []complex128{invSqrt2(), invSqrt2()}, PauliI, 1},
	}

	for _, tc := range cases {
		s := &fakeSampleState{numQubits: 1, amplitudes: tc.amps}
		got, err := Expectation(s, []PauliAxis{tc.axis})
		if err != nil {
			t.Errorf("%s: Expectation error: %v", tc.name, err)
			continue
		}
		if diff := got - tc.want; diff < -1e-10 || diff > 1e-10 {
			t.Errorf("%s: Expectation = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestExpectationOnBellState(t *testing.T) {
	cases := []struct {
		name string
		axes []PauliAxis
		want float64
	}{
		{"<XX>", []PauliAxis{PauliX, PauliX}, 1},
		{"<YY>", []PauliAxis{PauliY, PauliY}, -1},
		{"<ZZ>", []PauliAxis{PauliZ, PauliZ}, 1},
		{"<ZX>", []PauliAxis{PauliZ, PauliX}, 0},
		{"<XZ>", []PauliAxis{PauliX, PauliZ}, 0},
		{"<ZI>", []PauliAxis{PauliZ, PauliI}, 0},
		{"<II>", []PauliAxis{PauliI, PauliI}, 1},
	}

	for _, tc := range cases {
		s := &fakeSampleState{numQubits: 2, amplitudes: bellAmplitudes()}
		got, err := Expectation(s, tc.axes)
		if err != nil {
			t.Errorf("%s: Expectation error: %v", tc.name, err)
			continue
		}
		if diff := got - tc.want; diff < -1e-10 || diff > 1e-10 {
			t.Errorf("%s: Expectation = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestExpectationRejectsWrongAxisLength(t *testing.T) {
	s := &fakeSampleState{numQubits: 2, amplitudes: bellAmplitudes()}

	_, err := Expectation(s, []PauliAxis{PauliZ})
	e, ok := err.(*IncompatibleQubitCountError)
	if !ok {
		t.Fatalf("Expectation error = %T (%v), want *IncompatibleQubitCountError", err, err)
	}
	if e.Expected != 2 || e.Actual != 1 {
		t.Errorf("IncompatibleQubitCountError = expected %d actual %d, want 2 and 1", e.Expected, e.Actual)
	}
}

func TestExpectationRejectsInvalidAxis(t *testing.T) {
	s := &fakeSampleState{numQubits: 2, amplitudes: bellAmplitudes()}

	_, err := Expectation(s, []PauliAxis{PauliZ, PauliAxis(9)})
	e, ok := err.(*InvalidPauliAxisError)
	if !ok {
		t.Fatalf("Expectation error = %T (%v), want *InvalidPauliAxisError", err, err)
	}
	if e.Qubit != 1 || e.Axis != PauliAxis(9) {
		t.Errorf("InvalidPauliAxisError = qubit %d axis %d, want 1 and 9", e.Qubit, e.Axis)
	}
}

func TestExpectationRejectsNilState(t *testing.T) {
	axes := []PauliAxis{PauliZ}

	if _, err := Expectation(nil, axes); err == nil || err.Error() != "state must not be nil" {
		t.Errorf("Expectation(nil) error = %v, want \"state must not be nil\"", err)
	}

	var typedNil *fakeSampleState
	if _, err := Expectation(typedNil, axes); err == nil || err.Error() != "state must not be nil" {
		t.Errorf("Expectation(typed nil) error = %v, want \"state must not be nil\"", err)
	}
}

func TestExpectationRejectsUnnormalizedState(t *testing.T) {
	s := &fakeSampleState{numQubits: 1, amplitudes: []complex128{2, 0}}

	_, err := Expectation(s, []PauliAxis{PauliZ})
	if _, ok := err.(*UnnormalizedStateError); !ok {
		t.Fatalf("Expectation(unnormalized) error = %T (%v), want *UnnormalizedStateError", err, err)
	}

	nan := &fakeSampleState{numQubits: 1, amplitudes: []complex128{cmplx.NaN(), 0}}
	_, err = Expectation(nan, []PauliAxis{PauliZ})
	if _, ok := err.(*UnnormalizedStateError); !ok {
		t.Fatalf("Expectation(NaN) error = %T (%v), want *UnnormalizedStateError", err, err)
	}
}
