package algorithm

import (
	"errors"
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// backends is what taking a quantum.QuantumState buys: the same algorithm has
// to produce the same answer on either state representation, even though one
// stores a slice of every amplitude and the other a map of the non-zero ones.
var backends = []struct {
	name     string
	newState func(numQubits int) (quantum.QuantumState, error)
}{
	{
		name: "dense",
		newState: func(numQubits int) (quantum.QuantumState, error) {
			s, err := state.New(numQubits)
			if err != nil {
				return nil, err
			}
			return s, nil
		},
	},
	{
		name: "sparse",
		newState: func(numQubits int) (quantum.QuantumState, error) {
			s, err := sparsestate.New(numQubits)
			if err != nil {
				return nil, err
			}
			return s, nil
		},
	},
}

func TestDeutschJozsaOutputDistribution(t *testing.T) {
	tests := []struct {
		name            string
		numInputQubits  int
		oracle          Oracle
		wantZeroProbMin float64
		wantZeroProbMax float64
	}{
		{
			name:           "constant zero oracle",
			numInputQubits: 3,
			oracle: func(int) int {
				return 0
			},
			wantZeroProbMin: 1.0,
			wantZeroProbMax: 1.0,
		},
		{
			name:           "balanced parity oracle",
			numInputQubits: 3,
			oracle: func(input int) int {
				parity := 0
				for bit := input; bit > 0; bit >>= 1 {
					parity ^= bit & 1
				}
				return parity
			},
			wantZeroProbMin: 0.0,
			wantZeroProbMax: 0.0,
		},
	}

	for _, backend := range backends {
		for _, tt := range tests {
			t.Run(backend.name+"/"+tt.name, func(t *testing.T) {
				start, err := backend.newState(tt.numInputQubits + 1)
				if err != nil {
					t.Fatalf("creating the %s state failed: %v", backend.name, err)
				}

				s, err := DeutschJozsa(start, tt.oracle)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				zeroProb := inputRegisterZeroProbability(s, tt.numInputQubits)
				if zeroProb < tt.wantZeroProbMin-1e-9 || zeroProb > tt.wantZeroProbMax+1e-9 {
					t.Fatalf("zero probability %.6f out of expected range", zeroProb)
				}
			})
		}
	}
}

func TestGroverOutputDistribution(t *testing.T) {
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			start, err := backend.newState(3)
			if err != nil {
				t.Fatalf("creating the %s state failed: %v", backend.name, err)
			}

			s, err := Grover(start, []int{5})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if prob := s.Probability(5); prob < 0.9 {
				t.Fatalf("expected marked state probability > 0.9, got %.4f", prob)
			}

			total := 0.0
			for i := 0; i < 8; i++ {
				total += s.Probability(i)
			}
			if math.Abs(total-1.0) > 1e-9 {
				t.Fatalf("state not normalized, sum %.6f", total)
			}
		})
	}
}

// TestGroverBackendsAgree pins the two backends to each other rather than to a
// threshold: an amplitude-level disagreement is what a backend-specific bug in
// the bulk write would look like, and either backend alone cannot show it.
func TestGroverBackendsAgree(t *testing.T) {
	const numQubits = 4

	dense, err := state.New(numQubits)
	if err != nil {
		t.Fatalf("state.New(%d) failed: %v", numQubits, err)
	}
	sparse, err := sparsestate.New(numQubits)
	if err != nil {
		t.Fatalf("sparsestate.New(%d) failed: %v", numQubits, err)
	}

	marked := []int{3, 11}
	denseResult, err := Grover(dense, marked)
	if err != nil {
		t.Fatalf("Grover on the dense backend failed: %v", err)
	}
	sparseResult, err := Grover(sparse, marked)
	if err != nil {
		t.Fatalf("Grover on the sparse backend failed: %v", err)
	}

	for i := 0; i < 1<<numQubits; i++ {
		denseAmp, sparseAmp := denseResult.Amplitude(i), sparseResult.Amplitude(i)
		// The sparse backend prunes amplitudes at or below 1e-12, so the two
		// need not agree to the bit.
		if diff := denseAmp - sparseAmp; math.Hypot(real(diff), imag(diff)) > 1e-9 {
			t.Fatalf("amplitude %d: dense %v, sparse %v", i, denseAmp, sparseAmp)
		}
	}
}

// TestDeutschJozsaBackendsAgree is the Deutsch-Jozsa counterpart: the oracle
// swap is index arithmetic over the whole vector, which the two backends store
// differently.
func TestDeutschJozsaBackendsAgree(t *testing.T) {
	const numInputQubits = 3

	dense, err := state.New(numInputQubits + 1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	sparse, err := sparsestate.New(numInputQubits + 1)
	if err != nil {
		t.Fatalf("sparsestate.New failed: %v", err)
	}

	// Balanced on the low bit, so half the input states swap their pair.
	oracle := func(input int) int { return input & 1 }

	denseResult, err := DeutschJozsa(dense, oracle)
	if err != nil {
		t.Fatalf("DeutschJozsa on the dense backend failed: %v", err)
	}
	sparseResult, err := DeutschJozsa(sparse, oracle)
	if err != nil {
		t.Fatalf("DeutschJozsa on the sparse backend failed: %v", err)
	}

	for i := 0; i < 1<<(numInputQubits+1); i++ {
		denseAmp, sparseAmp := denseResult.Amplitude(i), sparseResult.Amplitude(i)
		if diff := denseAmp - sparseAmp; math.Hypot(real(diff), imag(diff)) > 1e-9 {
			t.Fatalf("amplitude %d: dense %v, sparse %v", i, denseAmp, sparseAmp)
		}
	}
}

// TestAlgorithmsReturnTheStateTheyWereGiven documents that these algorithms
// evolve the caller's state in place; the return value is a convenience, not a
// second state the caller has to reconcile with the first.
func TestAlgorithmsReturnTheStateTheyWereGiven(t *testing.T) {
	grover, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	if got, err := Grover(grover, []int{1}); err != nil {
		t.Fatalf("Grover failed: %v", err)
	} else if got != quantum.QuantumState(grover) {
		t.Errorf("Grover returned a different state than it was given")
	}

	dj, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	if got, err := DeutschJozsa(dj, func(int) int { return 0 }); err != nil {
		t.Fatalf("DeutschJozsa failed: %v", err)
	} else if got != quantum.QuantumState(dj) {
		t.Errorf("DeutschJozsa returned a different state than it was given")
	}
}

func inputRegisterZeroProbability(s quantum.QuantumState, numInputQubits int) float64 {
	ancillaBit := 1 << numInputQubits
	return s.Probability(0) + s.Probability(ancillaBit)
}

// Phase 3.1: Negative path tests for DeutschJozsa
//
// These reject their arguments before any amplitude is touched, so they are
// backend-independent and run on the dense state alone.

func TestDeutschJozsaNegativePaths(t *testing.T) {
	tests := []struct {
		name    string
		state   quantum.QuantumState
		oracle  Oracle
		wantErr string
	}{
		{
			name:    "nil state",
			state:   nil,
			oracle:  func(int) int { return 0 },
			wantErr: "state must not be nil",
		},
		{
			name:    "no room for an input register",
			state:   newDenseTestState(t, 1),
			oracle:  func(int) int { return 0 },
			wantErr: "state needs at least 2 qubits (one input qubit plus the ancilla), got 1",
		},
		{
			name:    "zero qubits",
			state:   &noBulkState{numQubits: 0},
			oracle:  func(int) int { return 0 },
			wantErr: "state needs at least 2 qubits (one input qubit plus the ancilla), got 0",
		},
		{
			name:    "negative qubits",
			state:   &noBulkState{numQubits: -1},
			oracle:  func(int) int { return 0 },
			wantErr: "state needs at least 2 qubits (one input qubit plus the ancilla), got -1",
		},
		{
			name:    "nil oracle",
			state:   newDenseTestState(t, 3),
			oracle:  nil,
			wantErr: "oracle must not be nil",
		},
		{
			name:    "oracle returns invalid value 2",
			state:   newDenseTestState(t, 3),
			oracle:  func(int) int { return 2 },
			wantErr: "oracle returned 2 for input 0",
		},
		{
			name:    "oracle returns invalid negative value",
			state:   newDenseTestState(t, 3),
			oracle:  func(int) int { return -1 },
			wantErr: "oracle returned -1 for input 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeutschJozsa(tt.state, tt.oracle)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

// applyGroverOracle and applyGroverDiffusion must surface SetAmplitudes
// failures instead of silently discarding them (mirrors applyDeutschJozsaOracle).
// Taking a bulkState rather than a concrete *state.State is what lets the
// failing case be provoked at all: the dense backend accepts every write these
// helpers make.
func TestGroverHelpersReturnSetAmplitudesErrors(t *testing.T) {
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	hGate := gates.NewHadamard()
	for i := 0; i < 2; i++ {
		if err := s.ApplyGate(hGate, i); err != nil {
			t.Fatalf("ApplyGate: %v", err)
		}
	}

	if err := applyGroverOracle(s, []int{3}); err != nil {
		t.Fatalf("applyGroverOracle on normalized state: %v", err)
	}
	if err := applyGroverDiffusion(s); err != nil {
		t.Fatalf("applyGroverDiffusion on normalized state: %v", err)
	}

	rejecting := &rejectingBulkState{inner: s, err: errRejected}
	if err := applyGroverOracle(rejecting, []int{3}); !errors.Is(err, errRejected) {
		t.Errorf("applyGroverOracle with a rejecting backend = %v, want %v", err, errRejected)
	}
	if err := applyGroverDiffusion(rejecting); !errors.Is(err, errRejected) {
		t.Errorf("applyGroverDiffusion with a rejecting backend = %v, want %v", err, errRejected)
	}
}

// algorithmRuns reduces each public algorithm to one func of the state, so the
// contracts they share — which backends they accept, what they do with a
// refused write — can be asserted once per algorithm instead of written out
// twice per contract. Both are given a two-qubit state and a balanced oracle,
// neither of which any of these tests depends on succeeding.
var algorithmRuns = []struct {
	name string
	run  func(quantum.QuantumState) error
}{
	{
		name: "Grover",
		run: func(s quantum.QuantumState) error {
			_, err := Grover(s, []int{1})
			return err
		},
	},
	{
		name: "DeutschJozsa",
		run: func(s quantum.QuantumState) error {
			_, err := DeutschJozsa(s, func(input int) int { return input & 1 })
			return err
		},
	},
}

// TestAlgorithmsPropagateBulkWriteErrors covers the same contract end to end:
// a rejected bulk write must abort the run, not leave it to finish on a state
// the backend refused to update.
func TestAlgorithmsPropagateBulkWriteErrors(t *testing.T) {
	for _, tt := range algorithmRuns {
		t.Run(tt.name, func(t *testing.T) {
			inner, err := state.New(2)
			if err != nil {
				t.Fatalf("state.New: %v", err)
			}
			rejecting := &rejectingBulkState{inner: inner, err: errRejected}

			if err := tt.run(rejecting); !errors.Is(err, errRejected) {
				t.Fatalf("%s with a rejecting backend = %v, want %v", tt.name, err, errRejected)
			}
			if rejecting.calls != 1 {
				t.Errorf("SetAmplitudes called %d times, want 1 (the run must stop at the first refusal)", rejecting.calls)
			}
		})
	}
}

// TestAlgorithmsRejectBackendsWithoutBulkWrites pins the promise the optional
// interface makes: a backend that is a QuantumState but cannot take a whole
// amplitude vector is turned away with a typed error, not a panic.
func TestAlgorithmsRejectBackendsWithoutBulkWrites(t *testing.T) {
	for _, tt := range algorithmRuns {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(&noBulkState{numQubits: 2})

			var unsupported *quantum.UnsupportedOperationError
			if !errors.As(err, &unsupported) {
				t.Fatalf("%s on a backend without SetAmplitudes = %v (%T), want *UnsupportedOperationError",
					tt.name, err, err)
			}
			if unsupported.Backend != "*algorithm.noBulkState" {
				t.Errorf("error names backend %q, want the offending type", unsupported.Backend)
			}
		})
	}
}

// TestAlgorithmsRejectUsedStates guards the precondition the signature change
// handed to callers: both algorithms assume they start from |0…0⟩, and a state
// that has already been evolved would otherwise produce a plausible-looking
// wrong answer.
func TestAlgorithmsRejectUsedStates(t *testing.T) {
	for _, backend := range backends {
		for _, tt := range algorithmRuns {
			t.Run(backend.name+"/"+tt.name, func(t *testing.T) {
				s, err := backend.newState(2)
				if err != nil {
					t.Fatalf("creating the %s state failed: %v", backend.name, err)
				}
				if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
					t.Fatalf("ApplyGate: %v", err)
				}

				err = tt.run(s)
				if err == nil {
					t.Fatalf("%s accepted a state that had already been evolved", tt.name)
				}
				const want = "state must start in |0...0> but holds probability 0.500000 there"
				if err.Error() != want {
					t.Fatalf("expected error %q, got %q", want, err.Error())
				}
			})
		}
	}
}

// Phase 3.1: Negative path tests for Grover

func TestGroverNegativePaths(t *testing.T) {
	tests := []struct {
		name    string
		state   quantum.QuantumState
		marked  []int
		wantErr string
	}{
		{
			name:    "nil state",
			state:   nil,
			marked:  []int{0},
			wantErr: "state must not be nil",
		},
		{
			name:    "zero qubits",
			state:   &noBulkState{numQubits: 0},
			marked:  []int{0},
			wantErr: "numQubits must be positive",
		},
		{
			name:    "negative qubits",
			state:   &noBulkState{numQubits: -1},
			marked:  []int{0},
			wantErr: "numQubits must be positive",
		},
		{
			name:    "empty marked set",
			state:   newDenseTestState(t, 2),
			marked:  []int{},
			wantErr: "marked set must not be empty",
		},
		{
			name:    "nil marked set",
			state:   newDenseTestState(t, 2),
			marked:  nil,
			wantErr: "marked set must not be empty",
		},
		{
			name:    "marked state negative",
			state:   newDenseTestState(t, 2),
			marked:  []int{-1},
			wantErr: "marked state -1 out of range",
		},
		{
			name:    "marked state exceeds total states",
			state:   newDenseTestState(t, 2),
			marked:  []int{4},
			wantErr: "marked state 4 out of range",
		},
		{
			name:    "marked state way out of range",
			state:   newDenseTestState(t, 3),
			marked:  []int{100},
			wantErr: "marked state 100 out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Grover(tt.state, tt.marked)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func newDenseTestState(t *testing.T, numQubits int) quantum.QuantumState {
	t.Helper()

	s, err := state.New(numQubits)
	if err != nil {
		t.Fatalf("state.New(%d) failed: %v", numQubits, err)
	}
	return s
}

// noBulkState is a QuantumState and nothing more: the backend that has to be
// turned away rather than panicked on. Its qubit count is settable so it can
// also stand in for the counts no real constructor will hand out.
type noBulkState struct {
	numQubits int
}

// The assertion is what keeps this stub honest — it must remain a complete
// QuantumState, or it would be refused for the wrong reason.
var _ quantum.QuantumState = (*noBulkState)(nil)

func (s *noBulkState) NumQubits() int { return s.numQubits }

func (s *noBulkState) Amplitude(basisState int) complex128 {
	if basisState == 0 {
		return 1
	}
	return 0
}

func (s *noBulkState) SetAmplitude(int, complex128) error   { return nil }
func (s *noBulkState) ApplyGate(quantum.Gate, ...int) error { return nil }
func (s *noBulkState) Measure(int) (int, error)             { return 0, nil }
func (s *noBulkState) Clone() quantum.QuantumState          { return s }
func (s *noBulkState) Probability(basisState int) float64 {
	return quantum.Probability(s.Amplitude(basisState))
}

var errRejected = errors.New("backend refused the write")

// rejectingBulkState is a real dense state whose bulk writes always fail,
// which is how the error paths above are reached without waiting for float
// drift to break normalization. It wraps rather than embeds: an embedded
// *state.State would promote its own SetAmplitudes and defeat the point.
type rejectingBulkState struct {
	inner *state.State
	err   error
	calls int
}

var _ bulkState = (*rejectingBulkState)(nil)

func (s *rejectingBulkState) SetAmplitudes([]complex128) error {
	s.calls++
	return s.err
}

func (s *rejectingBulkState) NumQubits() int { return s.inner.NumQubits() }
func (s *rejectingBulkState) Amplitude(basisState int) complex128 {
	return s.inner.Amplitude(basisState)
}
func (s *rejectingBulkState) SetAmplitude(basisState int, value complex128) error {
	return s.inner.SetAmplitude(basisState, value)
}
func (s *rejectingBulkState) ApplyGate(gate quantum.Gate, targets ...int) error {
	return s.inner.ApplyGate(gate, targets...)
}
func (s *rejectingBulkState) Measure(qubitIndex int) (int, error) { return s.inner.Measure(qubitIndex) }
func (s *rejectingBulkState) Probability(basisState int) float64 {
	return s.inner.Probability(basisState)
}
func (s *rejectingBulkState) Clone() quantum.QuantumState { return s.inner.Clone() }
