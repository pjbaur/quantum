package quantum

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

func TestQFTOnZeroStateIsUniform(t *testing.T) {
	s := &fakeBulkState{fakeSampleState{numQubits: 3, amplitudes: unitVector(8, 0)}}

	if err := QFT(s); err != nil {
		t.Fatalf("QFT: %v", err)
	}
	want := complex(1/math.Sqrt(8), 0)
	for i := 0; i < 8; i++ {
		if diff := cmplx.Abs(s.amplitudes[i] - want); diff > 1e-12 {
			t.Errorf("amplitude %d after QFT|000> = %v, want %v", i, s.amplitudes[i], want)
		}
	}
}

func TestQFTOnBasisStatesExact(t *testing.T) {
	// n=2, N=4, omega = e^{2 pi i / 4} = i: QFT|j> has amplitudes
	// omega^{jk} / 2 for k = 0..3.
	cases := []struct {
		j    int
		want []complex128
	}{
		{1, []complex128{1, 1i, -1, -1i}},
		{2, []complex128{1, -1, 1, -1}},
		{3, []complex128{1, -1i, -1, 1i}},
	}
	for _, tc := range cases {
		s := &fakeBulkState{fakeSampleState{numQubits: 2, amplitudes: unitVector(4, tc.j)}}
		if err := QFT(s); err != nil {
			t.Fatalf("QFT|j=%d>: %v", tc.j, err)
		}
		for k, want := range tc.want {
			wantScaled := want / complex(2, 0)
			if diff := cmplx.Abs(s.amplitudes[k] - wantScaled); diff > 1e-12 {
				t.Errorf("QFT|%d> amplitude %d = %v, want %v", tc.j, k, s.amplitudes[k], wantScaled)
			}
		}
	}
}

func TestInverseQFTOnPhaseGradientExact(t *testing.T) {
	// The state sum_y omega^{jy}|y> / sqrt(N) is QFT|j>, so the inverse
	// transform must return |j> exactly.
	for j := 0; j < 4; j++ {
		amps := make([]complex128, 4)
		for y := range amps {
			amps[y] = cmplx.Exp(complex(0, 2*math.Pi*float64(j*y)/4)) / complex(2, 0)
		}
		s := &fakeBulkState{fakeSampleState{numQubits: 2, amplitudes: amps}}

		if err := InverseQFT(s); err != nil {
			t.Fatalf("InverseQFT (j=%d): %v", j, err)
		}
		for y := 0; y < 4; y++ {
			want := complex(0, 0)
			if y == j {
				want = 1
			}
			if diff := cmplx.Abs(s.amplitudes[y] - want); diff > 1e-12 {
				t.Errorf("InverseQFT(QFT|%d>) amplitude %d = %v, want %v", j, y, s.amplitudes[y], want)
			}
		}
	}
}

func TestQFTInverseRoundtrip(t *testing.T) {
	// A fixed generic state, built deterministically.
	amps := make([]complex128, 8)
	rngGen := rand.New(rand.NewSource(3))
	rng := func() float64 { return rngGen.NormFloat64() }
	for i := range amps {
		amps[i] = complex(rng(), rng())
	}
	norm := 0.0
	for _, a := range amps {
		norm += real(a)*real(a) + imag(a)*imag(a)
	}
	scale := complex(1/math.Sqrt(norm), 0)
	for i := range amps {
		amps[i] *= scale
	}

	// Forward then inverse (and inverse then forward) must restore the
	// state up to float error.
	for _, order := range []struct {
		name string
		fns  []func(QuantumState) error
	}{
		{"QFT then InverseQFT", []func(QuantumState) error{QFT, InverseQFT}},
		{"InverseQFT then QFT", []func(QuantumState) error{InverseQFT, QFT}},
	} {
		s := &fakeBulkState{fakeSampleState{numQubits: 3, amplitudes: append([]complex128(nil), amps...)}}
		for _, fn := range order.fns {
			if err := fn(s); err != nil {
				t.Fatalf("%s: %v", order.name, err)
			}
		}
		for i, want := range amps {
			if diff := cmplx.Abs(s.amplitudes[i] - want); diff > 1e-9 {
				t.Errorf("%s: amplitude %d = %v, want %v", order.name, i, s.amplitudes[i], want)
			}
		}
	}
}

func TestQFTPreservesNormalization(t *testing.T) {
	s := &fakeBulkState{fakeSampleState{numQubits: 3, amplitudes: unitVector(8, 5)}}

	if err := QFT(s); err != nil {
		t.Fatalf("QFT: %v", err)
	}
	sum := 0.0
	for _, a := range s.amplitudes {
		sum += Probability(a)
	}
	if math.Abs(sum-1) > 1e-10 {
		t.Errorf("probability sum after QFT = %v, want 1", sum)
	}
}

func TestQFTRejectsNilState(t *testing.T) {
	if err := QFT(nil); err == nil || err.Error() != "state must not be nil" {
		t.Errorf("QFT(nil) error = %v, want \"state must not be nil\"", err)
	}
	var typedNil *fakeSampleState
	if err := InverseQFT(typedNil); err == nil || err.Error() != "state must not be nil" {
		t.Errorf("InverseQFT(typed nil) error = %v, want \"state must not be nil\"", err)
	}
}

func TestQFTRejectsBackendWithoutBulkWrites(t *testing.T) {
	// fakeSampleState implements QuantumState but not BulkAmplitudeSetter.
	s := &fakeSampleState{numQubits: 1, amplitudes: unitVector(2, 0)}

	err := QFT(s)
	e, ok := err.(*UnsupportedOperationError)
	if !ok {
		t.Fatalf("QFT error = %T (%v), want *UnsupportedOperationError", err, err)
	}
	if e.Operation != "QFT" {
		t.Errorf("UnsupportedOperationError.Operation = %q, want \"QFT\"", e.Operation)
	}
}

// fakeBulkState adds BulkAmplitudeSetter to the read-only fake so QFT's
// write-back path can run without a real backend.
type fakeBulkState struct {
	fakeSampleState
}

func (f *fakeBulkState) SetAmplitudes(values []complex128) error {
	if len(values) != len(f.amplitudes) {
		return fmt.Errorf("values slice length %d does not match state size %d",
			len(values), len(f.amplitudes))
	}
	copy(f.amplitudes, values)
	return nil
}

// unitVector returns a normalized basis vector of the given size with all
// amplitude on index basis.
func unitVector(size, basis int) []complex128 {
	amps := make([]complex128, size)
	amps[basis] = 1
	return amps
}
