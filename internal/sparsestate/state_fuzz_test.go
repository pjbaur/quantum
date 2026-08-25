package sparsestate

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

const (
	// normalizationTolerance is the slack this backend documents around a
	// probability sum of 1.
	normalizationTolerance = 1e-10

	// boundaryMargin is the band around that tolerance in which the
	// single-amplitude oracle below asserts nothing about acceptance. Two
	// things put the sparse backend's own sum a hair away from the oracle's:
	// it accumulates in Go map iteration order, which differs between runs,
	// and it prunes amplitudes at or below pruneEpsilon, dropping up to
	// pruneEpsilon² from the sum. Both are many orders of magnitude smaller
	// than this margin, which is itself a thousandth of the tolerance, so the
	// band costs almost nothing in strictness. The bulk oracle needs no such
	// band; FuzzSparseSetAmplitudes says why.
	boundaryMargin = 1e-13

	// backendAgreement is how far the two backends may differ per basis
	// state. It is wider than the package's tolerance const because the
	// sparse backend drops amplitudes at or below pruneEpsilon that the
	// dense backend keeps.
	backendAgreement = 1e-9
)

func byteAt(raw []byte, i int) byte {
	if i < len(raw) {
		return raw[i]
	}
	return 0
}

// angleFromByte maps a fuzz byte onto [0, 2π). The mapping is exact on
// whole fractions of a turn, so an all-zero input builds the identity and a
// seed can name π/2 (byte 64) or π (byte 128) precisely.
func angleFromByte(b byte) float64 {
	return 2 * math.Pi * float64(b) / 256
}

// oneQubitUnitary builds U(θ, φ, λ) carrying global phase γ. Every choice of
// angles yields a unitary, which is the point: raw random numbers poured
// into a matrix are essentially never unitary, and a non-unitary "gate"
// cannot tell a genuine backend disagreement from its own nonsense.
func oneQubitUnitary(theta, phi, lambda, gamma float64) [][]complex128 {
	cos := complex(math.Cos(theta/2), 0)
	sin := complex(math.Sin(theta/2), 0)
	global := cmplx.Exp(complex(0, gamma))

	return [][]complex128{
		{global * cos, -global * cmplx.Exp(complex(0, lambda)) * sin},
		{global * cmplx.Exp(complex(0, phi)) * sin, global * cmplx.Exp(complex(0, phi+lambda)) * cos},
	}
}

// unitaryFromBytes reads the four angles of one 1-qubit unitary starting at
// offset, so a byte stream describes a sequence of them. Missing bytes read
// as zero, which is the identity.
func unitaryFromBytes(raw []byte, offset int) [][]complex128 {
	return oneQubitUnitary(
		angleFromByte(byteAt(raw, offset)),
		angleFromByte(byteAt(raw, offset+1)),
		angleFromByte(byteAt(raw, offset+2)),
		angleFromByte(byteAt(raw, offset+3)),
	)
}

// twoQubitUnitary builds an entangling 2-qubit unitary as a product of
// unitary factors: local rotations on either side of a CNOT. All-zero angles
// collapse it to exactly the canonical CNOT, which is what puts the sparse
// backend's CNOT fast path under the same scrutiny as its general path.
func twoQubitUnitary(raw []byte) [][]complex128 {
	left := kron(unitaryFromBytes(raw, 0), unitaryFromBytes(raw, 4))
	right := kron(unitaryFromBytes(raw, 8), unitaryFromBytes(raw, 12))
	return matmul(matmul(left, gates.NewCNOT().Matrix()), right)
}

func kron(a, b [][]complex128) [][]complex128 {
	rows, inner := len(a), len(b)
	out := make([][]complex128, rows*inner)
	for i := range out {
		out[i] = make([]complex128, rows*inner)
	}
	for i := range a {
		for j := range a[i] {
			for k := range b {
				for l := range b[k] {
					out[i*inner+k][j*inner+l] = a[i][j] * b[k][l]
				}
			}
		}
	}
	return out
}

func matmul(a, b [][]complex128) [][]complex128 {
	size := len(a)
	out := make([][]complex128, size)
	for i := range out {
		out[i] = make([]complex128, size)
		for j := 0; j < size; j++ {
			sum := complex(0, 0)
			for k := 0; k < size; k++ {
				sum += a[i][k] * b[k][j]
			}
			out[i][j] = sum
		}
	}
	return out
}

// assertUnitary guards the generator rather than the code under test: an
// equivalence check fed a non-unitary matrix would still compare two
// backends, but it would stop describing anything a quantum state can do.
func assertUnitary(t *testing.T, matrix [][]complex128) {
	t.Helper()

	size := len(matrix)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			sum := complex(0, 0)
			for k := 0; k < size; k++ {
				sum += matrix[i][k] * cmplx.Conj(matrix[j][k])
			}
			want := complex(0, 0)
			if i == j {
				want = 1
			}
			if cmplx.Abs(sum-want) > 1e-9 {
				t.Fatalf("generated matrix is not unitary: (U·U†)[%d][%d] = %v, want %v", i, j, sum, want)
			}
		}
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

func snapshot(qs quantum.QuantumState) []complex128 {
	amps := make([]complex128, 1<<qs.NumQubits())
	for i := range amps {
		amps[i] = qs.Amplitude(i)
	}
	return amps
}

func probabilitySum(qs quantum.QuantumState) float64 {
	sum := 0.0
	for i := 0; i < 1<<qs.NumQubits(); i++ {
		sum += qs.Probability(i)
	}
	return sum
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

// prepareSparse puts the state into a fuzzed product state by applying one
// 1-qubit unitary per qubit. Preparing with gates rather than with
// SetAmplitudes is deliberate: it is the same preparation prepareDense runs on
// the dense backend, which is what lets the equivalence target below start
// both backends from a state neither one's bulk writer produced.
func prepareSparse(t *testing.T, sparse *State, numQubits int, prep []byte) {
	t.Helper()

	for qubit := 0; qubit < numQubits; qubit++ {
		gate, err := gates.NewMatrixGate("Prep", unitaryFromBytes(prep, 4*qubit))
		if err != nil {
			t.Fatalf("building the preparation unitary for qubit %d failed: %v", qubit, err)
		}
		if err := sparse.ApplyGate(gate, qubit); err != nil {
			t.Fatalf("preparing qubit %d failed: %v", qubit, err)
		}
	}
}

// boundaryValue returns the amplitude that, written at basisState, moves the
// probability sum to 1+delta, carrying the phase the fuzzer chose. When no
// such amplitude exists — the rest of the vector already holds more than the
// target — it falls back to zero, a legitimate write in its own right.
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
	angle := angleFromByte(phase)
	return complex(magnitude*math.Cos(angle), magnitude*math.Sin(angle))
}

// FuzzSparseSetAmplitude drives the sparse backend's single-amplitude writer
// across its validation boundary, starting from a state a fuzzed layer of
// unitaries has already put into superposition. The contract has three
// layers, checked in the order the backend applies them: the basis state
// must be in range, the value must be finite, and the resulting probability
// sum must stay within tolerance — and a write refused at any layer must
// leave the state exactly as it was.
//
// FuzzSparseSetAmplitudes below covers the bulk writer against the same
// contract.
func FuzzSparseSetAmplitude(f *testing.F) {
	// Phase-only changes to a prepared state, at each width.
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 0, uint8(0), 0.0, int8(-1))
	f.Add(uint8(1), []byte{64, 0, 128, 0, 32, 16, 200, 7}, 2, uint8(64), 0.0, int8(-1))
	f.Add(uint8(2), []byte{64, 0, 128, 0, 32, 16, 200, 7, 90, 91, 92, 93}, 5, uint8(128), 0.0, int8(-1))
	// The resulting sum lands exactly at the tolerance, and just past it.
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 1, uint8(0), 1e-10, int8(-1))
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 1, uint8(0), -1e-10, int8(-1))
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 1, uint8(0), 1e-9, int8(-1))
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 1, uint8(0), -1e-9, int8(-1))
	f.Add(uint8(1), []byte{64, 0, 128, 0}, 3, uint8(0), 0.5, int8(-1))
	// A magnitude the backend prunes away instead of storing.
	f.Add(uint8(0), []byte{0, 0, 0, 0}, 1, uint8(0), -1.0, int8(-1))
	// Out-of-range basis states, below and above.
	f.Add(uint8(0), []byte{64, 0, 128, 0}, -1, uint8(0), 0.0, int8(-1))
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 2, uint8(0), 0.0, int8(-1))
	f.Add(uint8(1), []byte{64, 0, 128, 0}, 1<<20, uint8(0), 0.0, int8(-1))
	// Non-finite values, including at an out-of-range index where the range
	// check has to win.
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 0, uint8(0), 0.0, int8(0))
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 1, uint8(0), 0.0, int8(1))
	f.Add(uint8(1), []byte{64, 0, 128, 0}, 3, uint8(0), 0.0, int8(2))
	f.Add(uint8(1), []byte{64, 0, 128, 0}, 2, uint8(0), 0.0, int8(3))
	f.Add(uint8(0), []byte{64, 0, 128, 0}, 9, uint8(0), 0.0, int8(4))

	f.Fuzz(func(t *testing.T, qubitsRaw uint8, prep []byte, basisState int, phase uint8, delta float64, poisonKind int8) {
		numQubits := 1 + int(qubitsRaw)%3
		size := 1 << numQubits

		sparse, err := New(numQubits)
		if err != nil {
			t.Fatalf("New(%d) failed: %v", numQubits, err)
		}
		prepareSparse(t, sparse, numQubits, prep)
		before := snapshot(sparse)

		value := boundaryValue(before, basisState, phase, delta)
		if poisoned, ok := nonFiniteValue(poisonKind); ok {
			value = poisoned
		}

		err = sparse.SetAmplitude(basisState, value)

		if basisState < 0 || basisState >= size {
			var rangeErr *quantum.QubitsOutOfRangeError
			if !errors.As(err, &rangeErr) {
				t.Fatalf("basis state %d is outside [0,%d]: SetAmplitude returned %v (%T), want *QubitsOutOfRangeError",
					basisState, size-1, err, err)
			}
			assertUnchanged(t, sparse, before, "after an out-of-range rejection")
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
			assertUnchanged(t, sparse, before, "after a non-finite rejection")
			return
		}

		sum := 0.0
		for i, amp := range before {
			if i == basisState {
				amp = value
			}
			sum += quantum.Probability(amp)
		}
		deviation := math.Abs(sum - 1.0)

		switch {
		case deviation <= normalizationTolerance-boundaryMargin:
			if err != nil {
				t.Fatalf("probability sum %v is within tolerance: SetAmplitude failed: %v", sum, err)
			}
		case deviation >= normalizationTolerance+boundaryMargin:
			var normErr *quantum.NormalizationError
			if !errors.As(err, &normErr) {
				t.Fatalf("probability sum %v is outside tolerance: SetAmplitude returned %v (%T), want *NormalizationError",
					sum, err, err)
			}
		}

		if err != nil {
			assertUnchanged(t, sparse, before, "after a normalization rejection")
			return
		}

		// A magnitude at or below the prune threshold is dropped rather
		// than stored, so it reads back as zero.
		want := value
		if isNearZero(value) {
			want = 0
		}
		if got := sparse.Amplitude(basisState); got != want {
			t.Fatalf("amplitude %d = %v, want %v", basisState, got, want)
		}
		for i, other := range before {
			if i == basisState {
				continue
			}
			if got := sparse.Amplitude(i); got != other {
				t.Fatalf("amplitude %d = %v, want %v (an accepted write must touch one amplitude only)", i, got, other)
			}
		}
		assertSparseUsable(t, sparse)
	})
}

// fuzzAmplitudes decodes raw into size amplitudes, two bytes each, with
// components mapped onto roughly [-1, 1]. It mirrors the generator the dense
// bulk-write target uses, so the two backends' bulk writers are explored over
// the same space rather than over two spaces that merely look alike.
func fuzzAmplitudes(raw []byte, size int) []complex128 {
	amps := make([]complex128, size)
	for i := range amps {
		amps[i] = complex(unitComponent(byteAt(raw, 2*i)), unitComponent(byteAt(raw, 2*i+1)))
	}
	return amps
}

// unitComponent centres the byte range on 128 so that an amplitude can come
// out exactly zero, which is what most basis states of a real state vector
// hold — and, for this backend, what most of the map does not hold at all.
func unitComponent(b byte) float64 {
	return (float64(b) - 128) / 127
}

// rescaleTo stretches amps so their probabilities sum to 1+delta, which aims
// the fuzzer at the normalization boundary rather than leaving it to chance.
// Vectors that cannot be scaled there (all-zero, or already non-finite) are
// left as they are; the oracle reads the sum back off the result, so a steer
// that misses costs nothing but coverage.
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

func firstNonFinite(amps []complex128) (int, bool) {
	for i, v := range amps {
		if !quantum.IsFiniteAmplitude(v) {
			return i, true
		}
	}
	return 0, false
}

// FuzzSparseSetAmplitudes drives this backend's bulk amplitude writer across
// the same validation boundary the dense target explores, with the same
// generator and the same oracle: the vector is steered to a probability sum of
// 1+delta and one entry can be poisoned with NaN or an infinity, so a
// non-finite amplitude has to be refused outright and everything else accepted
// exactly when the sum lands within the documented tolerance.
//
// Unlike FuzzSparseSetAmplitude, this oracle needs no boundaryMargin. That
// margin exists because the single-amplitude write is judged against a sum the
// backend accumulates over its map — in an iteration order that varies, over
// amplitudes it may have pruned. SetAmplitudes decides on the caller's slice
// instead, summed in ascending index order before anything is stored, which is
// bit-for-bit what the oracle computes and what the dense backend computes.
// Only the readback afterwards has to account for pruning.
func FuzzSparseSetAmplitudes(f *testing.F) {
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
	// A vector holding one amplitude far below the prune threshold, which
	// this backend must accept and then decline to store.
	f.Add(uint8(1), []byte{255, 128, 128, 128, 128, 128, 129, 128}, 0.0, uint8(0), int8(-1))
	// Four bytes past the end of a 1-qubit vector, which put the state into a
	// superposition first: these are the seeds where "a rejected write leaves
	// the state alone" has something to say.
	f.Add(uint8(0), []byte{255, 128, 128, 128, 64, 0, 128, 0}, 0.0, uint8(0), int8(-1))
	f.Add(uint8(0), []byte{255, 128, 128, 128, 64, 0, 128, 0}, 1.0, uint8(0), int8(-1))
	f.Add(uint8(0), []byte{255, 128, 128, 128, 64, 0, 128, 0}, 0.0, uint8(1), int8(0))
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

		sparse, err := New(numQubits)
		if err != nil {
			t.Fatalf("New(%d) failed: %v", numQubits, err)
		}
		// Whatever bytes the vector did not consume describe a preparation, so
		// a rejected write has a real superposition to fail to disturb rather
		// than only |0…0⟩. Bytes past the end read as zero, which is the
		// identity, so short inputs simply start from the ground state.
		prepareSparse(t, sparse, numQubits, raw[min(len(raw), 2*size):])
		before := snapshot(sparse)

		err = sparse.SetAmplitudes(amps)

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
			assertUnchanged(t, sparse, before, "after a non-finite rejection")
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
			assertUnchanged(t, sparse, before, "after a normalization rejection")
			return
		}

		if err != nil {
			t.Fatalf("probability sum %v is within the tolerance: SetAmplitudes failed: %v", sum, err)
		}
		for i, want := range amps {
			// A magnitude at or below the prune threshold is dropped rather
			// than stored, so it reads back as zero. That is the one place
			// this backend's bulk write differs from the dense one's.
			if isNearZero(want) {
				want = 0
			}
			if got := sparse.Amplitude(i); got != want {
				t.Fatalf("amplitude %d = %v, want %v", i, got, want)
			}
		}
		assertSparseUsable(t, sparse)
	})
}

// assertSparseUsable checks that a state the backend accepted can actually
// be computed with: a gate application and a collapse both have to leave
// finite, normalized amplitudes behind, whichever way the draw falls.
func assertSparseUsable(t *testing.T, s *State) {
	t.Helper()

	evolved := s.Clone().(*State)
	if err := evolved.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplyGate on an accepted state failed: %v", err)
	}
	assertBackendFinite(t, evolved, "after Hadamard")

	// Both ends and the middle of the draw range: each outcome, plus the
	// largest draw, which is what selects a branch holding no probability.
	for _, draw := range []float64{0, 0.5, math.Nextafter(1, 0)} {
		collapsed := s.Clone().(*State)
		collapsed.SetRandSource(&stubRandSource{values: []float64{draw}})

		got, err := collapsed.Measure(0)
		if err != nil {
			t.Fatalf("draw %v: Measure on an accepted state failed: %v", draw, err)
		}
		if got != 0 && got != 1 {
			t.Fatalf("draw %v: Measure = %d, want 0 or 1", draw, got)
		}
		assertBackendFinite(t, collapsed, "after collapse")
		if sum := probabilitySum(collapsed); math.Abs(sum-1.0) > backendAgreement {
			t.Fatalf("draw %v: probability sum after collapse = %v, want 1", draw, sum)
		}
	}
}

func assertBackendFinite(t *testing.T, qs quantum.QuantumState, context string) {
	t.Helper()

	for i := 0; i < 1<<qs.NumQubits(); i++ {
		if amp := qs.Amplitude(i); cmplx.IsNaN(amp) || cmplx.IsInf(amp) {
			t.Fatalf("%s: amplitude %d = %v, want a finite value", context, i, amp)
		}
	}
}

// assertBackendsAgree compares the two representations basis state by basis
// state. It carries a context string and checks for non-finite amplitudes,
// which is what separates it from assertStatesMatch: when a fuzz input fails
// there is no test name to say which stage diverged.
func assertBackendsAgree(t *testing.T, dense, sparse quantum.QuantumState, context string) {
	t.Helper()

	assertBackendFinite(t, dense, context+" (dense)")
	assertBackendFinite(t, sparse, context+" (sparse)")

	for i := 0; i < 1<<dense.NumQubits(); i++ {
		denseAmp, sparseAmp := dense.Amplitude(i), sparse.Amplitude(i)
		if cmplx.Abs(denseAmp-sparseAmp) > backendAgreement {
			t.Fatalf("%s: amplitude %d: dense %v, sparse %v", context, i, denseAmp, sparseAmp)
		}
		denseProb, sparseProb := dense.Probability(i), sparse.Probability(i)
		if math.Abs(denseProb-sparseProb) > backendAgreement {
			t.Fatalf("%s: probability %d: dense %v, sparse %v", context, i, denseProb, sparseProb)
		}
	}
}

// FuzzDenseSparseGateEquivalence applies one fuzzed unitary to the same
// prepared state in both backends and requires the results to agree. The two
// implementations share nothing but the gate matrix — dense walks a slice
// and swaps a scratch buffer, sparse rebuilds a map and prunes it, with a
// separate fast path for the canonical CNOT — so a disagreement pins the
// blame on whichever of them changed. That is what makes this the safety net
// for a refactor that pulls their gate math into shared code.
//
// Both the preparation and the gate come from angles rather than from raw
// matrix entries, so every matrix involved is unitary by construction.
func FuzzDenseSparseGateEquivalence(f *testing.F) {
	// Identity on |0⟩: the smallest possible case.
	f.Add(uint8(0), uint8(0), uint8(0), uint8(0), []byte(nil), []byte(nil))
	// Hadamard on |+⟩ (byte 64 is π/2, byte 128 is π).
	f.Add(uint8(0), uint8(0), uint8(0), uint8(0), []byte{64, 0, 128, 0}, []byte{64, 0, 128, 0})
	// All-zero angles collapse the 2-qubit construction to exactly the
	// canonical CNOT, which is the sparse backend's fast path. The
	// preparation puts qubit 0 into |+⟩ first, so the two target orders
	// give different states — on |00⟩ they would both be a no-op.
	f.Add(uint8(1), uint8(1), uint8(0), uint8(1), []byte{64, 0, 128, 0}, []byte(nil))
	f.Add(uint8(1), uint8(1), uint8(1), uint8(0), []byte{64, 0, 128, 0}, []byte(nil))
	// The same shape with a general 2-qubit unitary, in both target
	// orders: the ordering is what an index-juggling bug gets wrong.
	f.Add(uint8(1), uint8(1), uint8(0), uint8(1),
		[]byte{64, 0, 128, 0, 30, 90, 200, 0},
		[]byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160})
	f.Add(uint8(1), uint8(1), uint8(1), uint8(0),
		[]byte{64, 0, 128, 0, 30, 90, 200, 0},
		[]byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160})
	// Repeated targets: both backends have to refuse.
	f.Add(uint8(1), uint8(1), uint8(0), uint8(0), []byte(nil), []byte(nil))
	// A 2-qubit gate on the far pair of a 3-qubit register, entangled
	// during preparation (the trailing flag byte drives the CNOT chain).
	f.Add(uint8(2), uint8(1), uint8(2), uint8(1),
		[]byte{64, 0, 128, 0, 33, 77, 11, 5, 200, 100, 50, 25, 3},
		[]byte{7, 14, 21, 28, 35, 42, 49, 56, 63, 70, 77, 84, 91, 98, 105, 112})
	// A 1-qubit gate on the top qubit of a 4-qubit register.
	f.Add(uint8(3), uint8(0), uint8(3), uint8(0),
		[]byte{64, 0, 128, 0, 33, 77, 11, 5, 200, 100, 50, 25, 9, 8, 7, 6, 15},
		[]byte{200, 100, 50, 25})

	f.Fuzz(func(t *testing.T, qubitsRaw, arityRaw, target0Raw, target1Raw uint8, prep, angles []byte) {
		numQubits := 1 + int(qubitsRaw)%4
		arity := 1 + int(arityRaw)%2
		if arity > numQubits {
			arity = 1
		}

		matrix := unitaryFromBytes(angles, 0)
		if arity == 2 {
			matrix = twoQubitUnitary(angles)
		}
		assertUnitary(t, matrix)

		gate, err := gates.NewMatrixGate("FuzzUnitary", matrix)
		if err != nil {
			t.Fatalf("the generated %d-qubit matrix was rejected as a gate: %v", arity, err)
		}

		targets := []int{int(target0Raw) % numQubits}
		if arity == 2 {
			targets = append(targets, int(target1Raw)%numQubits)
		}

		dense, err := state.New(numQubits)
		if err != nil {
			t.Fatalf("state.New(%d) failed: %v", numQubits, err)
		}
		sparse, err := New(numQubits)
		if err != nil {
			t.Fatalf("New(%d) failed: %v", numQubits, err)
		}

		prepareSparse(t, sparse, numQubits, prep)
		prepareDense(t, dense, numQubits, prep)
		entangleBoth(t, dense, sparse, numQubits, byteAt(prep, 4*numQubits))
		assertBackendsAgree(t, dense, sparse, "after preparation")

		denseErr := dense.ApplyGate(gate, targets...)
		sparseErr := sparse.ApplyGate(gate, targets...)
		if (denseErr == nil) != (sparseErr == nil) {
			t.Fatalf("targets %v: dense returned %v but sparse returned %v", targets, denseErr, sparseErr)
		}
		if denseErr != nil {
			// Repeated targets are the only rejection reachable here, and
			// a rejection must leave both states alone.
			assertBackendsAgree(t, dense, sparse, "after a rejected gate")
			return
		}

		assertBackendsAgree(t, dense, sparse, "after the gate")
		if sum := probabilitySum(dense); math.Abs(sum-1.0) > backendAgreement {
			t.Fatalf("dense probability sum after a unitary = %v, want 1", sum)
		}
		if sum := probabilitySum(sparse); math.Abs(sum-1.0) > backendAgreement {
			t.Fatalf("sparse probability sum after a unitary = %v, want 1", sum)
		}
	})
}

func prepareDense(t *testing.T, dense *state.State, numQubits int, prep []byte) {
	t.Helper()

	for qubit := 0; qubit < numQubits; qubit++ {
		gate, err := gates.NewMatrixGate("Prep", unitaryFromBytes(prep, 4*qubit))
		if err != nil {
			t.Fatalf("building the preparation unitary for qubit %d failed: %v", qubit, err)
		}
		if err := dense.ApplyGate(gate, qubit); err != nil {
			t.Fatalf("preparing qubit %d failed: %v", qubit, err)
		}
	}
}

// entangleBoth runs a CNOT down the register wherever flags says so, so the
// gate under test can meet a state that no product of single-qubit states
// can express.
func entangleBoth(t *testing.T, dense *state.State, sparse *State, numQubits int, flags byte) {
	t.Helper()

	cnot := gates.NewCNOT()
	for qubit := 0; qubit+1 < numQubits; qubit++ {
		if flags>>qubit&1 == 0 {
			continue
		}
		if err := dense.ApplyGate(cnot, qubit, qubit+1); err != nil {
			t.Fatalf("dense CNOT(%d,%d) during preparation failed: %v", qubit, qubit+1, err)
		}
		if err := sparse.ApplyGate(cnot, qubit, qubit+1); err != nil {
			t.Fatalf("sparse CNOT(%d,%d) during preparation failed: %v", qubit, qubit+1, err)
		}
	}
}
