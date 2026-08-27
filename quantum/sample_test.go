package quantum

import (
	"math/cmplx"
	"testing"
)

// fakeSampleState is a minimal QuantumState standing in for a real backend
// so the sampling helper can be tested without an import cycle. Only the
// read paths (NumQubits, Amplitude) return real data; every mutating
// method panics, which also proves Sample never calls them.
type fakeSampleState struct {
	numQubits  int
	amplitudes []complex128
}

func (f *fakeSampleState) NumQubits() int { return f.numQubits }
func (f *fakeSampleState) Amplitude(basisState int) complex128 {
	return f.amplitudes[basisState]
}
func (f *fakeSampleState) SetAmplitude(int, complex128) error {
	panic("SetAmplitude must not be called")
}
func (f *fakeSampleState) ApplyGate(Gate, ...int) error {
	panic("ApplyGate must not be called")
}
func (f *fakeSampleState) Measure(int) (int, error) {
	panic("Measure must not be called")
}
func (f *fakeSampleState) Probability(basisState int) float64 {
	return Probability(f.amplitudes[basisState])
}
func (f *fakeSampleState) Clone() QuantumState { panic("Clone must not be called") }

// sequenceRand replays a fixed sequence of draws so sampling outcomes are
// exact and assertable.
type sequenceRand struct {
	draws []float64
	next  int
}

func (r *sequenceRand) Float64() float64 {
	v := r.draws[r.next]
	r.next++
	return v
}

func invSqrt2() complex128 {
	return complex(0.7071067811865476, 0)
}

func TestSampleRejectsInvalidShotCount(t *testing.T) {
	s := &fakeSampleState{numQubits: 1, amplitudes: []complex128{1, 0}}

	for _, shots := range []int{0, -1, -100} {
		counts, err := Sample(s, shots, &sequenceRand{draws: []float64{0.5}})
		if counts != nil {
			t.Errorf("Sample(shots=%d) returned counts %v, want nil", shots, counts)
		}
		e, ok := err.(*InvalidShotCountError)
		if !ok {
			t.Fatalf("Sample(shots=%d) error = %T (%v), want *InvalidShotCountError", shots, err, err)
		}
		if e.Requested != shots {
			t.Errorf("Sample(shots=%d) error Requested = %d, want %d", shots, e.Requested, shots)
		}
	}
}

func TestSampleRejectsNilState(t *testing.T) {
	rng := &sequenceRand{draws: []float64{0.5}}

	if _, err := Sample(nil, 1, rng); err == nil || err.Error() != "state must not be nil" {
		t.Errorf("Sample(nil state) error = %v, want \"state must not be nil\"", err)
	}

	var typedNil *fakeSampleState
	if _, err := Sample(typedNil, 1, rng); err == nil || err.Error() != "state must not be nil" {
		t.Errorf("Sample(typed nil state) error = %v, want \"state must not be nil\"", err)
	}
}

func TestSampleRejectsUnnormalizedState(t *testing.T) {
	s := &fakeSampleState{numQubits: 1, amplitudes: []complex128{2, 0}}

	_, err := Sample(s, 1, &sequenceRand{draws: []float64{0.5}})
	e, ok := err.(*UnnormalizedStateError)
	if !ok {
		t.Fatalf("Sample(unnormalized) error = %T (%v), want *UnnormalizedStateError", err, err)
	}
	if e.Sum != 4.0 {
		t.Errorf("UnnormalizedStateError.Sum = %v, want 4", e.Sum)
	}
}

func TestSampleRejectsStateWithNaNSum(t *testing.T) {
	s := &fakeSampleState{numQubits: 1, amplitudes: []complex128{cmplx.NaN(), 0}}

	_, err := Sample(s, 1, &sequenceRand{draws: []float64{0.5}})
	if _, ok := err.(*UnnormalizedStateError); !ok {
		t.Fatalf("Sample(NaN amplitude) error = %T (%v), want *UnnormalizedStateError", err, err)
	}
}

func TestSampleDeterministicOutcomes(t *testing.T) {
	// Equal superposition of |0> and |1>: probabilities [0.5, 0.5].
	s := &fakeSampleState{numQubits: 1, amplitudes: []complex128{invSqrt2(), invSqrt2()}}

	// Draw 0.25 lands below 0.5 (outcome 0); 0.75 and 0.6 land above
	// (outcome 1). Draws stay clear of the 0.5 boundary itself: invSqrt2
	// squared is 0.5000000000000001 in float, so exact-boundary probes
	// would test rounding, not bucketing.
	rng := &sequenceRand{draws: []float64{0.25, 0.75, 0.6}}
	counts, err := Sample(s, 3, rng)
	if err != nil {
		t.Fatalf("Sample returned error: %v", err)
	}
	want := map[string]int{"0": 1, "1": 2}
	for key, wantCount := range want {
		if counts[key] != wantCount {
			t.Errorf("counts[%q] = %d, want %d (full histogram %v)", key, counts[key], wantCount, counts)
		}
	}
	if len(counts) != len(want) {
		t.Errorf("histogram has %d keys, want %d: %v", len(counts), len(want), counts)
	}
}

func TestSampleBitstringMatchesBasisIndex(t *testing.T) {
	// Basis state 5 of a 3-qubit register is |101>: qubit 0 and qubit 2
	// are 1, qubit 1 is 0, so the key is "101" — zero-padded to the qubit
	// count, most significant bit last qubit, like FormatStateView.
	s := &fakeSampleState{numQubits: 3, amplitudes: []complex128{0, 0, 0, 0, 0, 1, 0, 0}}

	counts, err := Sample(s, 4, &sequenceRand{draws: []float64{0.0, 0.3, 0.6, 0.999}})
	if err != nil {
		t.Fatalf("Sample returned error: %v", err)
	}
	if len(counts) != 1 || counts["101"] != 4 {
		t.Errorf("counts = %v, want map[101:4]", counts)
	}
}

func TestSampleIgnoresPhase(t *testing.T) {
	// (|0> + i|1>)/sqrt(2): the relative phase must not affect probabilities.
	s := &fakeSampleState{numQubits: 1, amplitudes: []complex128{invSqrt2(), complex(0, 0.7071067811865476)}}

	counts, err := Sample(s, 2, &sequenceRand{draws: []float64{0.2, 0.8}})
	if err != nil {
		t.Fatalf("Sample returned error: %v", err)
	}
	if counts["0"] != 1 || counts["1"] != 1 {
		t.Errorf("counts = %v, want one 0 and one 1", counts)
	}
}
