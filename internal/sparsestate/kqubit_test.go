package sparsestate

import (
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// spreadSparseInputs prepares a state with two nonzero amplitudes
// (0.36 + 0.64 = 1) on both backends, so the sparse map path — not a
// dense-style full vector — is exercised.
func spreadSparseInputs(t *testing.T, dense quantum.QuantumState, sparse quantum.QuantumState) {
	t.Helper()
	amps := make([]complex128, 8)
	amps[0] = complex(0.6, 0)
	amps[3] = complex(0, 0.8)

	for _, s := range []quantum.QuantumState{dense, sparse} {
		setter, ok := s.(quantum.BulkAmplitudeSetter)
		if !ok {
			t.Fatalf("%T does not implement BulkAmplitudeSetter", s)
		}
		if err := setter.SetAmplitudes(amps); err != nil {
			t.Fatalf("SetAmplitudes on %T: %v", s, err)
		}
	}
}

func assertSparseMatchesDense(t *testing.T, dense quantum.QuantumState, sparse quantum.QuantumState, label string) {
	t.Helper()
	for i := 0; i < 8; i++ {
		if diff := cmplx.Abs(dense.Amplitude(i) - sparse.Amplitude(i)); diff > 1e-10 {
			t.Errorf("%s: amplitude %d: dense %v vs sparse %v", label, i, dense.Amplitude(i), sparse.Amplitude(i))
		}
	}
}

func TestSparseToffoliMatchesDense(t *testing.T) {
	dense, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	sparse, err := New(3)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	spreadSparseInputs(t, dense, sparse)

	if err := dense.ApplyGate(gates.NewToffoli(), 0, 1, 2); err != nil {
		t.Fatalf("dense Toffoli: %v", err)
	}
	if err := sparse.ApplyGate(gates.NewToffoli(), 0, 1, 2); err != nil {
		t.Fatalf("sparse Toffoli: %v", err)
	}
	assertSparseMatchesDense(t, dense, sparse, "Toffoli")
}

func TestSparseToffoliTruthTable(t *testing.T) {
	// Toffoli controls are the first two targets and the flip lands on
	// the third, so |011> (controls set, target clear) flips to |111>.
	s, err := New(3)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, q := range []int{0, 1} { // basis 0b011 = 3
		if err := s.ApplyGate(gates.NewPauliX(), q); err != nil {
			t.Fatalf("PauliX on %d: %v", q, err)
		}
	}
	if err := s.ApplyGate(gates.NewToffoli(), 0, 1, 2); err != nil {
		t.Fatalf("Toffoli: %v", err)
	}
	if amp := s.Amplitude(7); amp != 1 {
		t.Errorf("amplitude |111> after Toffoli|011> = %v, want 1", amp)
	}
	if amp := s.Amplitude(3); amp != 0 {
		t.Errorf("amplitude |011> after Toffoli|011> = %v, want 0", amp)
	}
}

func TestSparseGenericThreeQubitGateMatchesDense(t *testing.T) {
	// CC-S: a generic (non-fast-path) 3-qubit gate built from NewControlled.
	cs, err := gates.NewControlled(gates.NewS())
	if err != nil {
		t.Fatalf("NewControlled(S): %v", err)
	}
	ccs, err := gates.NewControlled(cs)
	if err != nil {
		t.Fatalf("NewControlled(CS): %v", err)
	}

	dense, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	sparse, err := New(3)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	spreadSparseInputs(t, dense, sparse)

	if err := dense.ApplyGate(ccs, 2, 1, 0); err != nil {
		t.Fatalf("dense CC-S: %v", err)
	}
	if err := sparse.ApplyGate(ccs, 2, 1, 0); err != nil {
		t.Fatalf("sparse CC-S: %v", err)
	}
	assertSparseMatchesDense(t, dense, sparse, "CC-S target order 2,1,0")
}

func TestSparseFourQubitGateMatchesDense(t *testing.T) {
	dense, err := state.New(4)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	sparse, err := New(4)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Uniform-ish input via H on every qubit.
	for q := 0; q < 4; q++ {
		if err := dense.ApplyGate(gates.NewHadamard(), q); err != nil {
			t.Fatalf("dense H on %d: %v", q, err)
		}
		if err := sparse.ApplyGate(gates.NewHadamard(), q); err != nil {
			t.Fatalf("sparse H on %d: %v", q, err)
		}
	}

	// Controlled-Toffoli: a 4-qubit gate.
	ccx, err := gates.NewControlled(gates.NewToffoli())
	if err != nil {
		t.Fatalf("NewControlled(Toffoli): %v", err)
	}
	if err := dense.ApplyGate(ccx, 3, 2, 1, 0); err != nil {
		t.Fatalf("dense controlled-Toffoli: %v", err)
	}
	if err := sparse.ApplyGate(ccx, 3, 2, 1, 0); err != nil {
		t.Fatalf("sparse controlled-Toffoli: %v", err)
	}

	for i := 0; i < 16; i++ {
		if diff := cmplx.Abs(dense.Amplitude(i) - sparse.Amplitude(i)); diff > 1e-10 {
			t.Errorf("4-qubit: amplitude %d: dense %v vs sparse %v", i, dense.Amplitude(i), sparse.Amplitude(i))
		}
	}
}
