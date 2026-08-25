package state_test

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// normalizationTolerance is the slack the dense backend documents around a
// probability sum of 1. The oracles below recompute that sum exactly the way
// the backend does — quantum.Probability over the amplitudes in ascending
// index order — so the accept/reject boundary is reproducible to the bit and
// can be asserted without a margin.
const normalizationTolerance = 1e-10

// fuzzAmplitudes decodes raw into size amplitudes, two bytes each, with
// components mapped onto roughly [-1, 1]. Bounded components keep the vector
// in the region a rescale can steer to the normalization boundary, instead
// of the wild exponents raw float bits would give. Bytes past the end of raw
// read as 0.
func fuzzAmplitudes(raw []byte, size int) []complex128 {
	amps := make([]complex128, size)
	for i := range amps {
		amps[i] = complex(unitComponent(byteAt(raw, 2*i)), unitComponent(byteAt(raw, 2*i+1)))
	}
	return amps
}

// unitComponent centres the byte range on 128 so that an amplitude can come
// out exactly zero, which is what most basis states of a real state vector
// hold.
func unitComponent(b byte) float64 {
	return (float64(b) - 128) / 127
}

func byteAt(raw []byte, i int) byte {
	if i < len(raw) {
		return raw[i]
	}
	return 0
}

// rescaleTo stretches amps so their probabilities sum to 1+delta, which is
// what aims the fuzzer at the normalization boundary rather than leaving it
// to chance. Vectors that cannot be scaled there (all-zero, or already
// non-finite) are left as they are; every oracle reads the sum back off the
// result, so steering that misses costs nothing but coverage.
func rescaleTo(amps []complex128, delta float64) {
	sum := 0.0
	for _, v := range amps {
		sum += quantum.Probability(v)
	}
	target := 1.0 + delta
	if sum <= 0 || math.IsInf(sum, 0) || target <= 0 || math.IsInf(target, 0) || math.IsNaN(target) {
		return
	}
	factor := complex(math.Sqrt(target/sum), 0)
	for i := range amps {
		amps[i] *= factor
	}
}

// nonFiniteValues are the amplitudes validation must refuse.
var nonFiniteValues = []complex128{
	complex(math.NaN(), 0),
	complex(0, math.NaN()),
	complex(math.Inf(1), 0),
	complex(0, math.Inf(-1)),
	complex(math.Inf(1), math.NaN()),
}

// nonFiniteValue picks one of them; a negative kind means "leave it finite".
func nonFiniteValue(kind int8) (complex128, bool) {
	if kind < 0 {
		return 0, false
	}
	return nonFiniteValues[int(kind)%len(nonFiniteValues)], true
}

func firstNonFinite(amps []complex128) (int, bool) {
	for i, v := range amps {
		if !quantum.IsFiniteAmplitude(v) {
			return i, true
		}
	}
	return 0, false
}

func snapshot(qs quantum.QuantumState) []complex128 {
	amps := make([]complex128, 1<<qs.NumQubits())
	for i := range amps {
		amps[i] = qs.Amplitude(i)
	}
	return amps
}

func assertUnchanged(t *testing.T, qs quantum.QuantumState, before []complex128, context string) {
	t.Helper()

	for i, want := range before {
		if got := qs.Amplitude(i); got != want {
			t.Fatalf("%s: amplitude %d = %v, want %v (a rejected write must not touch the state)",
				context, i, got, want)
		}
	}
}

func assertFinite(t *testing.T, qs quantum.QuantumState, context string) {
	t.Helper()

	for i := 0; i < 1<<qs.NumQubits(); i++ {
		if amp := qs.Amplitude(i); cmplx.IsNaN(amp) || cmplx.IsInf(amp) {
			t.Fatalf("%s: amplitude %d = %v, want a finite value", context, i, amp)
		}
	}
}

// assertUsable checks that a state the backend accepted can actually be
// computed with: a gate application and a collapse both have to leave finite,
// normalized amplitudes behind, whichever way the measurement draw falls.
func assertUsable(t *testing.T, s *state.State) {
	t.Helper()

	evolved := s.Clone().(*state.State)
	if err := evolved.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplyGate on an accepted state failed: %v", err)
	}
	assertFinite(t, evolved, "after Hadamard")

	// Both ends and the middle of the draw range: each outcome, plus the
	// largest draw, which is what selects a branch holding no probability.
	for _, draw := range []float64{0, 0.5, math.Nextafter(1, 0)} {
		collapsed := s.Clone().(*state.State)
		collapsed.SetRandSource(&stubRandSource{values: []float64{draw}})

		got, err := collapsed.Measure(0)
		if err != nil {
			t.Fatalf("draw %v: Measure on an accepted state failed: %v", draw, err)
		}
		if got != 0 && got != 1 {
			t.Fatalf("draw %v: Measure = %d, want 0 or 1", draw, got)
		}
		assertFinite(t, collapsed, "after collapse")
		assertNormalized(t, collapsed)
	}
}

// FuzzSetAmplitudes drives the bulk amplitude write across its validation
// boundary. The vector is steered to a probability sum of 1+delta and one
// entry can be poisoned with NaN or an infinity, so both halves of the
// contract are exercised: a non-finite amplitude is refused outright, and
// everything else is accepted exactly when the sum lands within the
// documented tolerance. Anything accepted must then be usable.
func FuzzSetAmplitudes(f *testing.F) {
	// Exactly normalized, over each supported width.
	f.Add(uint8(0), []byte{255, 128}, 0.0, uint8(0), int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250, 128, 128, 3, 9}, 0.0, uint8(0), int8(-1))
	f.Add(uint8(2), []byte{255, 0, 0, 255, 128, 129, 7, 200, 60, 61, 62, 63, 1, 2, 3, 4}, 0.0, uint8(0), int8(-1))
	// Exactly at the tolerance, on both sides.
	f.Add(uint8(0), []byte{255, 128}, 1e-10, uint8(0), int8(-1))
	f.Add(uint8(0), []byte{255, 128}, -1e-10, uint8(0), int8(-1))
	// Just beyond it, on both sides.
	f.Add(uint8(0), []byte{255, 128}, 1.1e-10, uint8(0), int8(-1))
	f.Add(uint8(0), []byte{255, 128}, -1.1e-10, uint8(0), int8(-1))
	// Comfortably outside.
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 1e-9, uint8(0), int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 1.0, uint8(0), int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, -0.5, uint8(0), int8(-1))
	// An all-zero vector: nothing to scale, probability sum 0.
	f.Add(uint8(0), []byte{128, 128, 128, 128}, 0.0, uint8(0), int8(-1))
	// Non-finite amplitudes in an otherwise normalized vector.
	f.Add(uint8(0), []byte{255, 128}, 0.0, uint8(0), int8(0))
	f.Add(uint8(0), []byte{255, 128}, 0.0, uint8(1), int8(1))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 0.0, uint8(2), int8(2))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 0.0, uint8(3), int8(3))
	f.Add(uint8(2), []byte{200, 40, 10, 250}, 0.0, uint8(5), int8(4))

	f.Fuzz(func(t *testing.T, qubitsRaw uint8, raw []byte, delta float64, poisonIndex uint8, poisonKind int8) {
		numQubits := 1 + int(qubitsRaw)%3
		size := 1 << numQubits

		amps := fuzzAmplitudes(raw, size)
		rescaleTo(amps, delta)
		if value, ok := nonFiniteValue(poisonKind); ok {
			amps[int(poisonIndex)%size] = value
		}

		s, err := state.New(numQubits)
		if err != nil {
			t.Fatalf("state.New(%d) failed: %v", numQubits, err)
		}
		before := snapshot(s)

		err = s.SetAmplitudes(amps)

		if index, found := firstNonFinite(amps); found {
			var nonFinite *quantum.NonFiniteAmplitudeError
			if !errors.As(err, &nonFinite) {
				t.Fatalf("amplitude %d is %v: SetAmplitudes returned %v (%T), want *NonFiniteAmplitudeError",
					index, amps[index], err, err)
			}
			if nonFinite.BasisState != index {
				t.Fatalf("SetAmplitudes reported basis state %d, want the first offender %d",
					nonFinite.BasisState, index)
			}
			assertUnchanged(t, s, before, "after a non-finite rejection")
			return
		}

		sum := 0.0
		for _, v := range amps {
			sum += quantum.Probability(v)
		}

		if math.Abs(sum-1.0) > normalizationTolerance {
			var normErr *quantum.NormalizationError
			if !errors.As(err, &normErr) {
				t.Fatalf("probability sum %v is outside the tolerance: SetAmplitudes returned %v (%T), want *NormalizationError",
					sum, err, err)
			}
			assertUnchanged(t, s, before, "after a normalization rejection")
			return
		}

		if err != nil {
			t.Fatalf("probability sum %v is within the tolerance: SetAmplitudes failed: %v", sum, err)
		}
		for i, want := range amps {
			if got := s.Amplitude(i); got != want {
				t.Fatalf("amplitude %d = %v, want the value written, %v", i, got, want)
			}
		}
		assertUsable(t, s)
	})
}

// FuzzSetAmplitude drives the single-amplitude write across the same
// boundary, from a state that a fuzzed vector has already put into a
// non-trivial superposition. On top of the accept/reject contract this pins
// down the rollback: a refused write must leave the state exactly as it was.
func FuzzSetAmplitude(f *testing.F) {
	// Phase-only changes to a normalized state, at each width.
	f.Add(uint8(0), []byte{255, 128}, 0, uint8(0), 0.0, int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250, 128, 128, 3, 9}, 2, uint8(64), 0.0, int8(-1))
	f.Add(uint8(2), []byte{255, 0, 0, 255, 128, 129, 7, 200}, 5, uint8(128), 0.0, int8(-1))
	// The resulting sum lands exactly at the tolerance, and just past it.
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 1, uint8(0), 1e-10, int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 1, uint8(0), -1e-10, int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 1, uint8(0), 1.1e-10, int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 1, uint8(0), -1.1e-10, int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 1, uint8(0), 0.5, int8(-1))
	// Out-of-range basis states, below and above.
	f.Add(uint8(0), []byte{255, 128}, -1, uint8(0), 0.0, int8(-1))
	f.Add(uint8(0), []byte{255, 128}, 2, uint8(0), 0.0, int8(-1))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 1<<20, uint8(0), 0.0, int8(-1))
	// Non-finite values, including at an out-of-range index where the range
	// check has to win.
	f.Add(uint8(0), []byte{255, 128}, 0, uint8(0), 0.0, int8(0))
	f.Add(uint8(0), []byte{255, 128}, 1, uint8(0), 0.0, int8(2))
	f.Add(uint8(1), []byte{200, 40, 10, 250}, 3, uint8(0), 0.0, int8(3))
	f.Add(uint8(0), []byte{255, 128}, 9, uint8(0), 0.0, int8(1))

	f.Fuzz(func(t *testing.T, qubitsRaw uint8, raw []byte, basisState int, phase uint8, delta float64, poisonKind int8) {
		numQubits := 1 + int(qubitsRaw)%3
		size := 1 << numQubits

		start := fuzzAmplitudes(raw, size)
		rescaleTo(start, 0)

		s, err := state.New(numQubits)
		if err != nil {
			t.Fatalf("state.New(%d) failed: %v", numQubits, err)
		}
		if err := s.SetAmplitudes(start); err != nil {
			// The rescale could not reach a normalized vector (an all-zero
			// input, say). There is nothing to write onto, so drop the input.
			return
		}
		before := snapshot(s)

		value := boundaryValue(before, basisState, phase, delta)
		if poisoned, ok := nonFiniteValue(poisonKind); ok {
			value = poisoned
		}

		err = s.SetAmplitude(basisState, value)

		if basisState < 0 || basisState >= size {
			var rangeErr *quantum.QubitsOutOfRangeError
			if !errors.As(err, &rangeErr) {
				t.Fatalf("basis state %d is outside [0,%d]: SetAmplitude returned %v (%T), want *QubitsOutOfRangeError",
					basisState, size-1, err, err)
			}
			assertUnchanged(t, s, before, "after an out-of-range rejection")
			return
		}

		if !quantum.IsFiniteAmplitude(value) {
			var nonFinite *quantum.NonFiniteAmplitudeError
			if !errors.As(err, &nonFinite) {
				t.Fatalf("value %v is not finite: SetAmplitude returned %v (%T), want *NonFiniteAmplitudeError",
					value, err, err)
			}
			if nonFinite.BasisState != basisState {
				t.Fatalf("SetAmplitude reported basis state %d, want %d", nonFinite.BasisState, basisState)
			}
			assertUnchanged(t, s, before, "after a non-finite rejection")
			return
		}

		sum := 0.0
		for i, amp := range before {
			if i == basisState {
				amp = value
			}
			sum += quantum.Probability(amp)
		}

		if math.Abs(sum-1.0) > normalizationTolerance {
			var normErr *quantum.NormalizationError
			if !errors.As(err, &normErr) {
				t.Fatalf("probability sum %v is outside the tolerance: SetAmplitude returned %v (%T), want *NormalizationError",
					sum, err, err)
			}
			assertUnchanged(t, s, before, "after a normalization rejection")
			return
		}

		if err != nil {
			t.Fatalf("probability sum %v is within the tolerance: SetAmplitude failed: %v", sum, err)
		}
		if got := s.Amplitude(basisState); got != value {
			t.Fatalf("amplitude %d = %v, want the value written, %v", basisState, got, value)
		}
		for i, want := range before {
			if i == basisState {
				continue
			}
			if got := s.Amplitude(i); got != want {
				t.Fatalf("amplitude %d = %v, want %v (an accepted write must touch one amplitude only)", i, got, want)
			}
		}
		assertUsable(t, s)
	})
}

// boundaryValue returns the amplitude that, written at basisState, moves the
// probability sum to 1+delta, carrying the phase the fuzzer chose. When no
// such amplitude exists — the rest of the vector already holds more than the
// target — it falls back to zero, which is a legitimate write in its own
// right.
func boundaryValue(amps []complex128, basisState int, phase uint8, delta float64) complex128 {
	if basisState < 0 || basisState >= len(amps) {
		return complex(math.Sqrt(0.5), math.Sqrt(0.5))
	}

	rest := 0.0
	for i, amp := range amps {
		if i != basisState {
			rest += quantum.Probability(amp)
		}
	}

	wanted := 1.0 + delta - rest
	if !(wanted >= 0) || math.IsInf(wanted, 0) {
		return 0
	}

	magnitude := math.Sqrt(wanted)
	angle := 2 * math.Pi * float64(phase) / 256
	return complex(magnitude*math.Cos(angle), magnitude*math.Sin(angle))
}
