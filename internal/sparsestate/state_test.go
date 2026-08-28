package sparsestate

import (
	"errors"
	"math"
	"math/cmplx"
	"math/rand"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

const tolerance = 1e-10

// mockThreeQubitGate is a test gate that operates on 3 qubits.
type mockThreeQubitGate struct{}

func newMockThreeQubitGate() *mockThreeQubitGate {
	return &mockThreeQubitGate{}
}

func (g *mockThreeQubitGate) Name() string {
	return "Mock3Qubit"
}

func (g *mockThreeQubitGate) Matrix() [][]complex128 {
	// 8x8 identity matrix (3-qubit gate)
	matrix := make([][]complex128, 8)
	for i := range matrix {
		matrix[i] = make([]complex128, 8)
		matrix[i][i] = 1
	}
	return matrix
}

func TestSparseSingleQubitMatchesDense(t *testing.T) {
	tests := []struct {
		name      string
		gate      quantum.Gate
		targets   []int
		numQubits int
		setup     func(quantum.QuantumState) error
	}{
		{
			name:      "hadamard on middle qubit",
			gate:      gates.NewHadamard(),
			targets:   []int{1},
			numQubits: 3,
			setup: func(qs quantum.QuantumState) error {
				if err := qs.ApplyGate(gates.NewPauliX(), 0); err != nil {
					return err
				}
				return qs.ApplyGate(gates.NewPauliX(), 2)
			},
		},
		{
			name:      "phase gate on highest qubit",
			gate:      gates.NewS(),
			targets:   []int{2},
			numQubits: 3,
			setup: func(qs quantum.QuantumState) error {
				return qs.ApplyGate(gates.NewHadamard(), 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dense, err := state.New(tt.numQubits)
			if err != nil {
				t.Fatalf("dense state.New failed: %v", err)
			}
			sparse, err := New(tt.numQubits)
			if err != nil {
				t.Fatalf("sparse New failed: %v", err)
			}

			if tt.setup != nil {
				if err := tt.setup(dense); err != nil {
					t.Fatalf("dense setup failed: %v", err)
				}
				if err := tt.setup(sparse); err != nil {
					t.Fatalf("sparse setup failed: %v", err)
				}
			}

			if err := dense.ApplyGate(tt.gate, tt.targets...); err != nil {
				t.Fatalf("dense ApplyGate failed: %v", err)
			}
			if err := sparse.ApplyGate(tt.gate, tt.targets...); err != nil {
				t.Fatalf("sparse ApplyGate failed: %v", err)
			}

			assertStatesMatch(t, dense, sparse)
		})
	}
}

func TestSparseCNOTMatchesDense(t *testing.T) {
	dense, err := state.New(3)
	if err != nil {
		t.Fatalf("dense state.New failed: %v", err)
	}
	sparse, err := New(3)
	if err != nil {
		t.Fatalf("sparse New failed: %v", err)
	}

	if err := dense.ApplyGate(gates.NewHadamard(), 1); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewHadamard(), 1); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	if err := dense.ApplyGate(gates.NewCNOT(), 1, 2); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewCNOT(), 1, 2); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	assertStatesMatch(t, dense, sparse)
}

func TestSparseSwapMatchesDense(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("dense state.New failed: %v", err)
	}
	sparse, err := New(2)
	if err != nil {
		t.Fatalf("sparse New failed: %v", err)
	}

	if err := dense.ApplyGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	if err := dense.ApplyGate(gates.NewSwap(), 0, 1); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewSwap(), 0, 1); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	assertStatesMatch(t, dense, sparse)
}

type denseTwoQubitGate struct {
	matrix [][]complex128
}

func newDenseTwoQubitGate() *denseTwoQubitGate {
	return &denseTwoQubitGate{
		matrix: [][]complex128{
			{1, 0, 0, 0},
			{0, 0, 1, 0},
			{0, 1, 0, 0},
			{0, 0, 0, -1},
		},
	}
}

func (g *denseTwoQubitGate) Name() string {
	return "DenseTwoQubit"
}

func (g *denseTwoQubitGate) Matrix() [][]complex128 {
	return g.matrix
}

// impostorCNOTGate is named "CNOT" but carries a CZ matrix. The sparse
// backend must dispatch on the matrix content, not the gate name.
type impostorCNOTGate struct{}

func (g *impostorCNOTGate) Name() string {
	return "CNOT"
}

func (g *impostorCNOTGate) Matrix() [][]complex128 {
	return [][]complex128{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, -1},
	}
}

func TestSparseGateNamedCNOTUsesItsMatrix(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("dense state.New failed: %v", err)
	}
	sparse, err := New(2)
	if err != nil {
		t.Fatalf("sparse New failed: %v", err)
	}

	// Prepare (|00⟩+|01⟩+|10⟩+|11⟩)/2 so CZ and CNOT semantics diverge.
	for _, qs := range []quantum.QuantumState{dense, sparse} {
		for i := 0; i < 2; i++ {
			if err := qs.ApplyGate(gates.NewHadamard(), i); err != nil {
				t.Fatalf("ApplyGate Hadamard failed: %v", err)
			}
		}
	}

	impostor := &impostorCNOTGate{}
	if err := dense.ApplyGate(impostor, 0, 1); err != nil {
		t.Fatalf("dense ApplyGate failed: %v", err)
	}
	if err := sparse.ApplyGate(impostor, 0, 1); err != nil {
		t.Fatalf("sparse ApplyGate failed: %v", err)
	}

	assertStatesMatch(t, dense, sparse)
}

func TestSparseTwoQubitTargetOrderingMatchesDense(t *testing.T) {
	tests := []struct {
		name      string
		gate      quantum.Gate
		targets   []int
		numQubits int
		setup     func(quantum.QuantumState) error
	}{
		{
			name:      "cnot reversed non-adjacent targets",
			gate:      gates.NewCNOT(),
			targets:   []int{2, 0},
			numQubits: 3,
			setup: func(qs quantum.QuantumState) error {
				if err := qs.ApplyGate(gates.NewHadamard(), 2); err != nil {
					return err
				}
				return qs.ApplyGate(gates.NewPauliX(), 0)
			},
		},
		{
			name:      "swap reversed targets",
			gate:      gates.NewSwap(),
			targets:   []int{2, 0},
			numQubits: 3,
			setup: func(qs quantum.QuantumState) error {
				if err := qs.ApplyGate(gates.NewHadamard(), 2); err != nil {
					return err
				}
				return qs.ApplyGate(gates.NewPauliX(), 0)
			},
		},
		{
			name:      "generic two-qubit gate with non-sorted targets",
			gate:      newDenseTwoQubitGate(),
			targets:   []int{3, 1},
			numQubits: 4,
			setup: func(qs quantum.QuantumState) error {
				if err := qs.ApplyGate(gates.NewHadamard(), 1); err != nil {
					return err
				}
				return qs.ApplyGate(gates.NewPauliX(), 3)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dense, err := state.New(tt.numQubits)
			if err != nil {
				t.Fatalf("dense state.New failed: %v", err)
			}
			sparse, err := New(tt.numQubits)
			if err != nil {
				t.Fatalf("sparse New failed: %v", err)
			}

			if tt.setup != nil {
				if err := tt.setup(dense); err != nil {
					t.Fatalf("dense setup failed: %v", err)
				}
				if err := tt.setup(sparse); err != nil {
					t.Fatalf("sparse setup failed: %v", err)
				}
			}

			if err := dense.ApplyGate(tt.gate, tt.targets...); err != nil {
				t.Fatalf("dense ApplyGate failed: %v", err)
			}
			if err := sparse.ApplyGate(tt.gate, tt.targets...); err != nil {
				t.Fatalf("sparse ApplyGate failed: %v", err)
			}

			assertStatesMatch(t, dense, sparse)
		})
	}
}

func TestSparseSetAmplitudeNormalization(t *testing.T) {
	sparse, err := New(2)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if err := sparse.SetAmplitude(0, complex(0.25, 0)); err == nil {
		t.Fatalf("expected normalization error, got nil")
	}
	if amp := sparse.Amplitude(0); cmplx.Abs(amp-1) > tolerance {
		t.Fatalf("expected amplitude to remain 1, got %v", amp)
	}
}

// TestSparseSetAmplitudeRejectsNonFinite covers the amplitudes the
// normalization check cannot catch on its own: |NaN|² is NaN, and NaN fails
// every comparison against the tolerance, so a NaN amplitude would otherwise
// have to be caught by accident rather than by rule.
func TestSparseSetAmplitudeRejectsNonFinite(t *testing.T) {
	tests := []struct {
		name  string
		value complex128
	}{
		{"NaN real part", complex(math.NaN(), 0)},
		{"NaN imaginary part", complex(0, math.NaN())},
		{"positive infinity", complex(math.Inf(1), 0)},
		{"negative infinity", complex(0, math.Inf(-1))},
		{"infinity and NaN together", complex(math.Inf(1), math.NaN())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sparse, err := New(1)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			err = sparse.SetAmplitude(0, tt.value)

			var nonFinite *quantum.NonFiniteAmplitudeError
			if !errors.As(err, &nonFinite) {
				t.Fatalf("SetAmplitude(0, %v) = %v (%T), want *NonFiniteAmplitudeError", tt.value, err, err)
			}
			if nonFinite.BasisState != 0 {
				t.Errorf("reported basis state %d, want 0", nonFinite.BasisState)
			}
			if amp := sparse.Amplitude(0); cmplx.Abs(amp-1) > tolerance {
				t.Errorf("amplitude 0 = %v, want 1 (a rejected write must not touch the state)", amp)
			}
		})
	}
}

// The sparse backend is only usable by the algorithm package while it offers
// bulk amplitude writes, so the capability is pinned at compile time.
var _ quantum.BulkAmplitudeSetter = (*State)(nil)

func TestSparseSetAmplitudesRejectsWrongLength(t *testing.T) {
	tests := []struct {
		name   string
		values []complex128
	}{
		{"empty", nil},
		{"one short", []complex128{1, 0, 0}},
		{"one long", []complex128{1, 0, 0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sparse, err := New(2)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			if err := sparse.SetAmplitudes(tt.values); err == nil {
				t.Fatalf("SetAmplitudes with %d values succeeded, want a length error", len(tt.values))
			}
			if amp := sparse.Amplitude(0); cmplx.Abs(amp-1) > tolerance {
				t.Errorf("amplitude 0 = %v, want 1 (a rejected write must not touch the state)", amp)
			}
		})
	}
}

// TestSparseSetAmplitudesRejectsNonFinite mirrors the dense backend's test:
// |NaN|² is NaN and NaN fails every comparison, so the normalization check
// below cannot be what catches these.
func TestSparseSetAmplitudesRejectsNonFinite(t *testing.T) {
	tests := []struct {
		name  string
		value complex128
	}{
		{"NaN real part", complex(math.NaN(), 0)},
		{"NaN imaginary part", complex(0, math.NaN())},
		{"positive infinity", complex(math.Inf(1), 0)},
		{"negative infinity", complex(0, math.Inf(-1))},
		{"infinity and NaN together", complex(math.Inf(1), math.NaN())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sparse, err := New(1)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			// The finite half alone carries the whole probability, so only
			// the poisoned amplitude can be what makes this invalid.
			err = sparse.SetAmplitudes([]complex128{1, tt.value})

			var nonFinite *quantum.NonFiniteAmplitudeError
			if !errors.As(err, &nonFinite) {
				t.Fatalf("SetAmplitudes with %v = %v (%T), want *NonFiniteAmplitudeError", tt.value, err, err)
			}
			if nonFinite.BasisState != 1 {
				t.Errorf("reported basis state %d, want 1", nonFinite.BasisState)
			}
			if amp := sparse.Amplitude(0); cmplx.Abs(amp-1) > tolerance {
				t.Errorf("amplitude 0 = %v, want 1 (a rejected write must not touch the state)", amp)
			}
		})
	}
}

func TestSparseSetAmplitudesRejectsUnnormalized(t *testing.T) {
	sparse, err := New(2)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = sparse.SetAmplitudes([]complex128{0.5, 0.5, 0.5, 0})

	var normErr *quantum.NormalizationError
	if !errors.As(err, &normErr) {
		t.Fatalf("SetAmplitudes with probability sum 0.75 = %v (%T), want *NormalizationError", err, err)
	}
	if math.Abs(normErr.AttemptedSum-0.75) > tolerance {
		t.Errorf("reported attempted sum %v, want 0.75", normErr.AttemptedSum)
	}
	if amp := sparse.Amplitude(0); cmplx.Abs(amp-1) > tolerance {
		t.Errorf("amplitude 0 = %v, want 1 (a rejected write must not touch the state)", amp)
	}
}

// TestSparseSetAmplitudesErrorOrder pins the order the checks run in — length,
// then finiteness, then normalization — because that is what keeps this
// backend's diagnosis of a bad vector identical to the dense one's.
func TestSparseSetAmplitudesErrorOrder(t *testing.T) {
	nan := complex(math.NaN(), 0)

	tests := []struct {
		name   string
		values []complex128
		check  func(t *testing.T, err error)
	}{
		{
			name:   "wrong length wins over a non-finite value",
			values: []complex128{1, nan, 0},
			check: func(t *testing.T, err error) {
				var nonFinite *quantum.NonFiniteAmplitudeError
				if errors.As(err, &nonFinite) {
					t.Fatalf("got %v, want the length error to win", err)
				}
				if err == nil {
					t.Fatal("got nil, want a length error")
				}
			},
		},
		{
			name:   "a non-finite value wins over an unnormalized sum",
			values: []complex128{0.5, nan, 0, 0},
			check: func(t *testing.T, err error) {
				var nonFinite *quantum.NonFiniteAmplitudeError
				if !errors.As(err, &nonFinite) {
					t.Fatalf("got %v (%T), want *NonFiniteAmplitudeError", err, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sparse, err := New(2)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}
			tt.check(t, sparse.SetAmplitudes(tt.values))
		})
	}
}

// TestSparseSetAmplitudesPrunesAndForgets covers what makes this backend's
// bulk write more than a slice copy: amplitudes too small to keep are dropped
// rather than stored, and basis states the old vector held must not survive
// the write that zeroes them.
func TestSparseSetAmplitudesPrunesAndForgets(t *testing.T) {
	sparse, err := New(2)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Spread the state over all four basis states first, so the write below
	// has stale entries to clear.
	half := complex(0.5, 0)
	if err := sparse.SetAmplitudes([]complex128{half, half, half, half}); err != nil {
		t.Fatalf("SetAmplitudes over four basis states failed: %v", err)
	}
	if len(sparse.amplitudes) != 4 {
		t.Fatalf("stored %d amplitudes, want 4", len(sparse.amplitudes))
	}

	// A magnitude at the prune threshold carries 1e-24 of probability, far
	// below the 1e-10 normalization tolerance, so this vector is accepted.
	tiny := complex(pruneEpsilon, 0)
	if err := sparse.SetAmplitudes([]complex128{1, tiny, 0, 0}); err != nil {
		t.Fatalf("SetAmplitudes with a prunable amplitude failed: %v", err)
	}

	if len(sparse.amplitudes) != 1 {
		t.Errorf("stored %d amplitudes, want 1 (prunable and zero entries must not be kept)", len(sparse.amplitudes))
	}
	if amp := sparse.Amplitude(1); amp != 0 {
		t.Errorf("amplitude 1 = %v, want 0 (a pruned amplitude reads back as zero)", amp)
	}
	for _, index := range []int{2, 3} {
		if amp := sparse.Amplitude(index); amp != 0 {
			t.Errorf("amplitude %d = %v, want 0 (a stale entry survived the write)", index, amp)
		}
	}
}

// TestSparseSetAmplitudesMatchesDense is the point of adding the method: the
// two backends have to accept the same vectors and read back the same state.
func TestSparseSetAmplitudesMatchesDense(t *testing.T) {
	values := []complex128{
		complex(0.5, 0),
		complex(0, 0.5),
		complex(-0.5, 0),
		complex(0.25, math.Sqrt(0.1875)),
	}

	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	sparse, err := New(2)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if err := dense.SetAmplitudes(values); err != nil {
		t.Fatalf("dense SetAmplitudes failed: %v", err)
	}
	if err := sparse.SetAmplitudes(values); err != nil {
		t.Fatalf("sparse SetAmplitudes failed: %v", err)
	}

	assertStatesMatch(t, dense, sparse)
}

func assertStatesMatch(t *testing.T, dense quantum.QuantumState, sparse quantum.QuantumState) {
	t.Helper()

	if dense.NumQubits() != sparse.NumQubits() {
		t.Fatalf("mismatched qubit counts: %d vs %d", dense.NumQubits(), sparse.NumQubits())
	}

	states := 1 << dense.NumQubits()
	for i := 0; i < states; i++ {
		denseAmp := dense.Amplitude(i)
		sparseAmp := sparse.Amplitude(i)
		if cmplx.Abs(denseAmp-sparseAmp) > tolerance {
			t.Fatalf("amplitude mismatch at %d: dense=%v sparse=%v", i, denseAmp, sparseAmp)
		}
		denseProb := dense.Probability(i)
		sparseProb := sparse.Probability(i)
		if math.Abs(denseProb-sparseProb) > tolerance {
			t.Fatalf("probability mismatch at %d: dense=%v sparse=%v", i, denseProb, sparseProb)
		}
	}
}

func TestSparseBackendCapabilities(t *testing.T) {
	sparse, err := New(3)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Test SupportsGateQubits: the generic k-qubit path covers any width.
	for k := 1; k <= 5; k++ {
		if !sparse.SupportsGateQubits(k) {
			t.Errorf("expected sparse backend to support %d-qubit gates", k)
		}
	}
	if sparse.SupportsGateQubits(0) {
		t.Error("expected sparse backend to NOT support 0-qubit gates")
	}

	// Test MaxGateQubits: zero means no limit.
	if max := sparse.MaxGateQubits(); max != 0 {
		t.Errorf("expected MaxGateQubits=0 (no limit), got %d", max)
	}
}

// TestSparseGateMatrixSizeMismatch pins that a gate whose matrix does not
// match its target count is refused outright, never partially applied —
// the boundary check the backend still owns now that every gate width is
// supported.
func TestSparseGateMatrixSizeMismatch(t *testing.T) {
	sparse, err := New(3)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	for qubit := 0; qubit < 3; qubit++ {
		if err := sparse.ApplyGate(gates.NewPauliX(), qubit); err != nil {
			t.Fatalf("ApplyGate(PauliX, %d) failed: %v", qubit, err)
		}
	}

	// A 4x4 matrix cannot act on 3 targets.
	fourByFour, err := gates.NewMatrixGate("FourByFour", [][]complex128{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	})
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}

	err = sparse.ApplyGate(fourByFour, 0, 1, 2)
	if err == nil {
		t.Fatal("expected error for 4x4 matrix on 3 targets")
	}

	// The mismatch is caught by the target-count validation: a 4x4 matrix
	// is a 2-qubit gate, so three targets are rejected before the matrix
	// is ever read.
	var sizeErr *quantum.InvalidGateApplicationError
	if !errors.As(err, &sizeErr) {
		t.Fatalf("expected InvalidGateApplicationError, got %T: %v", err, err)
	}
	if sizeErr.RequiredLen != 2 || sizeErr.ActualLen != 3 {
		t.Errorf("error reports required %d actual %d, want 2 and 3", sizeErr.RequiredLen, sizeErr.ActualLen)
	}

	// The refusal must leave the state exactly as it was.
	if got := sparse.Amplitude(7); cmplx.Abs(got-1) > tolerance {
		t.Errorf("amplitude of |111⟩ = %v after refused gate, want 1", got)
	}
}

func TestSparseGateInterfaceAssertion(t *testing.T) {
	// Verify State implements both QuantumState and BackendCapabilities
	s1, _ := New(1)
	var _ quantum.QuantumState = s1
	var _ quantum.BackendCapabilities = s1
}

func TestNewInvalidQubitCount(t *testing.T) {
	tests := []struct {
		name      string
		numQubits int
		errCheck  func(error) bool
	}{
		{
			name:      "zero qubits",
			numQubits: 0,
			errCheck: func(err error) bool {
				var targetErr *quantum.InvalidQubitCountError
				return errors.As(err, &targetErr) &&
					targetErr.Requested == 0 &&
					targetErr.Reason == "must be positive"
			},
		},
		{
			name:      "negative qubits",
			numQubits: -1,
			errCheck: func(err error) bool {
				var targetErr *quantum.InvalidQubitCountError
				return errors.As(err, &targetErr) &&
					targetErr.Requested == -1 &&
					targetErr.Reason == "must be positive"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.numQubits)
			if err == nil {
				t.Fatalf("expected error for %d qubits, got nil", tt.numQubits)
			}
			if !tt.errCheck(err) {
				t.Fatalf("unexpected error: %T: %v", err, err)
			}
		})
	}
}

func TestNewValidQubitCount(t *testing.T) {
	s, err := New(3)
	if err != nil {
		t.Fatalf("expected no error for 3 qubits, got: %v", err)
	}
	if s.NumQubits() != 3 {
		t.Fatalf("expected 3 qubits, got %d", s.NumQubits())
	}
}

// stubRandSource returns a fixed sequence of values, cycling.
type stubRandSource struct {
	values []float64
	index  int
}

func (s *stubRandSource) Float64() float64 {
	v := s.values[s.index%len(s.values)]
	s.index++
	return v
}

func TestSparseMeasureUsesInjectedRandSource(t *testing.T) {
	// Measure returns 1 when the draw is >= prob0 (0.5 for |+⟩).
	cases := []struct {
		draw float64
		want int
	}{
		{0.7, 1},
		{0.3, 0},
		{0.5, 1},
	}
	for _, tc := range cases {
		s, err := New(1)
		if err != nil {
			t.Fatalf("New(1) failed: %v", err)
		}
		if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
			t.Fatalf("ApplyGate failed: %v", err)
		}
		s.SetRandSource(&stubRandSource{values: []float64{tc.draw}})
		got, err := s.Measure(0)
		if err != nil {
			t.Fatalf("Measure failed: %v", err)
		}
		if got != tc.want {
			t.Errorf("draw %.1f: Measure = %d, want %d", tc.draw, got, tc.want)
		}
	}
}

// TestSparseMeasureZeroProbabilityBranch drives Measure into the outcome
// that carries no probability, mirroring the dense backend's test. The |1⟩
// amplitude is small enough that this backend prunes it outright, so the
// state passes the normalization check while prob0 lands just short of 1 —
// close enough that the largest draw rand.Float64 can return selects outcome
// 1 anyway. Without the guard the collapse divides by that branch's zero
// normalization factor, leaving the state empty instead of normalized.
func TestSparseMeasureZeroProbabilityBranch(t *testing.T) {
	s, err := New(1)
	if err != nil {
		t.Fatalf("New(1) failed: %v", err)
	}
	if err := s.SetAmplitude(0, complex(1-1e-16, 0)); err != nil {
		t.Fatalf("SetAmplitude(0) failed: %v", err)
	}
	// Pruned on the way in, which is precisely what empties the |1⟩ branch.
	if err := s.SetAmplitude(1, complex(1e-200, 0)); err != nil {
		t.Fatalf("SetAmplitude(1) failed: %v", err)
	}
	s.SetRandSource(&stubRandSource{values: []float64{math.Nextafter(1, 0)}})

	got, err := s.Measure(0)
	if err != nil {
		t.Fatalf("Measure failed: %v", err)
	}
	if got != 0 {
		t.Errorf("Measure = %d, want 0 (the only outcome holding probability)", got)
	}

	for i := 0; i < 2; i++ {
		if amp := s.Amplitude(i); cmplx.IsNaN(amp) || cmplx.IsInf(amp) {
			t.Errorf("amplitude %d after collapse = %v, want a finite value", i, amp)
		}
	}
	if sum := s.probabilitySum(); math.Abs(sum-1.0) > tolerance {
		t.Errorf("probability sum after collapse = %v, want 1.0", sum)
	}
}

// TestSparseMeasureEmptyStateErrors covers the guard's other arm: when
// neither outcome holds probability there is nothing to collapse onto, so
// Measure reports the broken invariant instead of producing a NaN state.
// Only reachable by building a state that bypasses the normalization check.
func TestSparseMeasureEmptyStateErrors(t *testing.T) {
	s := &State{numQubits: 1, amplitudes: map[int]complex128{}}

	if _, err := s.Measure(0); err == nil {
		t.Fatal("Measure on an empty state returned no error, want one")
	}
}

// TestSeededMeasurementDenseSparseEquivalence verifies that identically
// seeded sources drive identical measurement outcomes on both backends.
func TestSeededMeasurementDenseSparseEquivalence(t *testing.T) {
	const rounds = 32
	denseSrc := rand.New(rand.NewSource(1234))
	sparseSrc := rand.New(rand.NewSource(1234))

	for i := 0; i < rounds; i++ {
		dense, err := state.New(2)
		if err != nil {
			t.Fatalf("state.New failed: %v", err)
		}
		sparse, err := New(2)
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}
		for _, s := range []quantum.QuantumState{dense, sparse} {
			if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
				t.Fatalf("ApplyGate(H, 0) failed: %v", err)
			}
			if err := s.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
				t.Fatalf("ApplyGate(CNOT) failed: %v", err)
			}
		}
		dense.SetRandSource(denseSrc)
		sparse.SetRandSource(sparseSrc)

		denseGot, err := dense.Measure(0)
		if err != nil {
			t.Fatalf("dense Measure failed: %v", err)
		}
		sparseGot, err := sparse.Measure(0)
		if err != nil {
			t.Fatalf("sparse Measure failed: %v", err)
		}
		if denseGot != sparseGot {
			t.Fatalf("round %d: dense measured %d, sparse measured %d with identical seeds", i, denseGot, sparseGot)
		}
	}
}
