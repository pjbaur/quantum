package quantum

import (
	"testing"
)

// oneQubitState is a self-contained single-qubit QuantumState double whose
// ApplyGate multiplies by a 1-qubit matrix and records the call, and whose
// Measure returns 0 or 1 from a fixed draw against |a0|².
type oneQubitState struct {
	amplitudes []complex128
	draw       float64
	applied    int
}

func (o *oneQubitState) NumQubits() int                     { return 1 }
func (o *oneQubitState) Amplitude(b int) complex128         { return o.amplitudes[b] }
func (o *oneQubitState) SetAmplitude(int, complex128) error { panic("not needed") }
func (o *oneQubitState) Probability(b int) float64          { return Probability(o.amplitudes[b]) }
func (o *oneQubitState) Clone() QuantumState                { panic("not needed") }

func (o *oneQubitState) ApplyGate(g Gate, targets ...int) error {
	m := g.Matrix()
	a0, a1 := o.amplitudes[0], o.amplitudes[1]
	o.amplitudes[0] = m[0][0]*a0 + m[0][1]*a1
	o.amplitudes[1] = m[1][0]*a0 + m[1][1]*a1
	o.applied++
	return nil
}

func (o *oneQubitState) Measure(int) (int, error) {
	if o.draw < Probability(o.amplitudes[0]) {
		return 0, nil
	}
	return 1, nil
}

func TestClassicalRegisterRoundtrip(t *testing.T) {
	creg, err := NewClassicalRegister(4)
	if err != nil {
		t.Fatalf("NewClassicalRegister: %v", err)
	}
	if creg.NumBits() != 4 {
		t.Errorf("NumBits = %d, want 4", creg.NumBits())
	}

	// Bits start cleared.
	for i := 0; i < 4; i++ {
		got, err := creg.Bit(i)
		if err != nil {
			t.Fatalf("Bit(%d): %v", i, err)
		}
		if got {
			t.Errorf("fresh bit %d = true, want false", i)
		}
	}

	if err := creg.Set(2, true); err != nil {
		t.Fatalf("Set(2, true): %v", err)
	}
	got, err := creg.Bit(2)
	if err != nil {
		t.Fatalf("Bit(2): %v", err)
	}
	if !got {
		t.Errorf("Bit(2) after Set true = false, want true")
	}

	if err := creg.Set(2, false); err != nil {
		t.Fatalf("Set(2, false): %v", err)
	}
	if got, _ := creg.Bit(2); got {
		t.Errorf("Bit(2) after Set false = true, want false")
	}
}

func TestClassicalRegisterRejectsInvalidBitCount(t *testing.T) {
	for _, n := range []int{0, -1} {
		creg, err := NewClassicalRegister(n)
		if creg != nil {
			t.Errorf("NewClassicalRegister(%d) returned %v, want nil", n, creg)
		}
		e, ok := err.(*InvalidBitCountError)
		if !ok {
			t.Fatalf("NewClassicalRegister(%d) error = %T (%v), want *InvalidBitCountError", n, err, err)
		}
		if e.Requested != n {
			t.Errorf("InvalidBitCountError.Requested = %d, want %d", e.Requested, n)
		}
	}
}

func TestClassicalRegisterRejectsOutOfRange(t *testing.T) {
	creg, err := NewClassicalRegister(2)
	if err != nil {
		t.Fatalf("NewClassicalRegister: %v", err)
	}

	err = creg.Set(2, true)
	if e, ok := err.(*BitOutOfRangeError); !ok {
		t.Errorf("Set(2) error = %T (%v), want *BitOutOfRangeError", err, err)
	} else if e.Index != 2 || e.MaxIndex != 1 {
		t.Errorf("BitOutOfRangeError = index %d max %d, want 2 and 1", e.Index, e.MaxIndex)
	}

	_, err = creg.Bit(-1)
	if _, ok := err.(*BitOutOfRangeError); !ok {
		t.Errorf("Bit(-1) error = %T (%v), want *BitOutOfRangeError", err, err)
	}
}

func TestApplyIfSetAppliesOnlyWhenSet(t *testing.T) {
	creg, err := NewClassicalRegister(1)
	if err != nil {
		t.Fatalf("NewClassicalRegister: %v", err)
	}
	x := rotationGate{name: "X", matrix: [][]complex128{{0, 1}, {1, 0}}}
	s := &oneQubitState{amplitudes: []complex128{1, 0}}

	// Bit clear: gate must not run.
	if err := ApplyIfSet(creg, 0, s, x, 0); err != nil {
		t.Fatalf("ApplyIfSet (clear): %v", err)
	}
	if s.applied != 0 {
		t.Errorf("gate ran %d times with bit clear, want 0", s.applied)
	}

	// Bit set: gate runs on the state.
	if err := creg.Set(0, true); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := ApplyIfSet(creg, 0, s, x, 0); err != nil {
		t.Fatalf("ApplyIfSet (set): %v", err)
	}
	if s.applied != 1 {
		t.Errorf("gate ran %d times with bit set, want 1", s.applied)
	}
	if got := s.Amplitude(1); got != 1 {
		t.Errorf("amplitude 1 after X on |0> = %v, want 1", got)
	}
}

func TestApplyIfSetRejectsBadArguments(t *testing.T) {
	creg, err := NewClassicalRegister(1)
	if err != nil {
		t.Fatalf("NewClassicalRegister: %v", err)
	}
	x := rotationGate{name: "X", matrix: [][]complex128{{0, 1}, {1, 0}}}

	if err := ApplyIfSet(nil, 0, &oneQubitState{amplitudes: []complex128{1, 0}}, x, 0); err == nil {
		t.Error("ApplyIfSet(nil register) = nil error, want error")
	}

	err = ApplyIfSet(creg, 5, &oneQubitState{amplitudes: []complex128{1, 0}}, x, 0)
	if _, ok := err.(*BitOutOfRangeError); !ok {
		t.Errorf("ApplyIfSet(bit 5 of 1) error = %T (%v), want *BitOutOfRangeError", err, err)
	}
}

func TestMeasureIntoStoresOutcome(t *testing.T) {
	creg, err := NewClassicalRegister(2)
	if err != nil {
		t.Fatalf("NewClassicalRegister: %v", err)
	}

	// Equal superposition; the stub draw 0.75 >= |a0|² picks outcome 1.
	s := &oneQubitState{amplitudes: []complex128{invSqrt2(), invSqrt2()}, draw: 0.75}

	if err := MeasureInto(s, 0, creg, 1); err != nil {
		t.Fatalf("MeasureInto: %v", err)
	}
	got, err := creg.Bit(1)
	if err != nil {
		t.Fatalf("Bit(1): %v", err)
	}
	if !got {
		t.Error("MeasureInto stored false for outcome 1, want true")
	}
}

func TestMeasureIntoRejectsBadArguments(t *testing.T) {
	creg, err := NewClassicalRegister(1)
	if err != nil {
		t.Fatalf("NewClassicalRegister: %v", err)
	}
	s := &oneQubitState{amplitudes: []complex128{1, 0}, draw: 0.1}

	if err := MeasureInto(nil, 0, creg, 0); err == nil {
		t.Error("MeasureInto(nil state) = nil error, want error")
	}
	if err := MeasureInto(s, 0, nil, 0); err == nil {
		t.Error("MeasureInto(nil register) = nil error, want error")
	}
	err = MeasureInto(s, 3, creg, 0)
	if _, ok := err.(*QubitsOutOfRangeError); !ok {
		t.Errorf("MeasureInto(qubit 3 of 1) error = %T (%v), want *QubitsOutOfRangeError", err, err)
	}
	err = MeasureInto(s, 0, creg, 9)
	if _, ok := err.(*BitOutOfRangeError); !ok {
		t.Errorf("MeasureInto(bit 9 of 1) error = %T (%v), want *BitOutOfRangeError", err, err)
	}
}
