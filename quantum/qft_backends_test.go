package quantum_test

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

func TestQFTDenseSparseAgree(t *testing.T) {
	dense, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	sparse, err := sparsestate.New(3)
	if err != nil {
		t.Fatalf("sparsestate.New: %v", err)
	}

	// Same generic input on both backends: squared magnitudes sum to 52,
	// so scale by 1/sqrt(52) for a normalized vector.
	amps := []complex128{1, 2i, 3, -2i, 4, -4i, 1, 1}
	for i := range amps {
		amps[i] /= complex(math.Sqrt(52), 0)
	}
	var denseQS, sparseQS quantum.QuantumState = dense, sparse
	if err := denseQS.(quantum.BulkAmplitudeSetter).SetAmplitudes(amps); err != nil {
		t.Fatalf("dense SetAmplitudes: %v", err)
	}
	if err := sparseQS.(quantum.BulkAmplitudeSetter).SetAmplitudes(amps); err != nil {
		t.Fatalf("sparse SetAmplitudes: %v", err)
	}

	if err := quantum.QFT(dense); err != nil {
		t.Fatalf("dense QFT: %v", err)
	}
	if err := quantum.QFT(sparse); err != nil {
		t.Fatalf("sparse QFT: %v", err)
	}

	for i := 0; i < 8; i++ {
		d, s := dense.Amplitude(i), sparse.Amplitude(i)
		if cmplx.Abs(d-s) > 1e-10 {
			t.Errorf("amplitude %d: dense %v vs sparse %v", i, d, s)
		}
	}

	// And the roundtrip closes on both.
	if err := quantum.InverseQFT(dense); err != nil {
		t.Fatalf("dense InverseQFT: %v", err)
	}
	if err := quantum.InverseQFT(sparse); err != nil {
		t.Fatalf("sparse InverseQFT: %v", err)
	}
	for i, want := range amps {
		if diff := cmplx.Abs(dense.Amplitude(i) - want); diff > 1e-9 {
			t.Errorf("dense roundtrip amplitude %d = %v, want %v", i, dense.Amplitude(i), want)
		}
		if diff := cmplx.Abs(sparse.Amplitude(i) - want); diff > 1e-9 {
			t.Errorf("sparse roundtrip amplitude %d = %v, want %v", i, sparse.Amplitude(i), want)
		}
	}
}

func TestQFTSparseOutputUsable(t *testing.T) {
	sparse, err := sparsestate.New(2)
	if err != nil {
		t.Fatalf("sparsestate.New: %v", err)
	}

	if err := quantum.QFT(sparse); err != nil {
		t.Fatalf("QFT on sparse |00>: %v", err)
	}
	// QFT|00> is uniform; the sparse backend must now hold four entries.
	for i := 0; i < 4; i++ {
		if p := sparse.Probability(i); math.Abs(p-0.25) > 1e-10 {
			t.Errorf("sparse probability %d after QFT = %v, want 0.25", i, p)
		}
	}
}
