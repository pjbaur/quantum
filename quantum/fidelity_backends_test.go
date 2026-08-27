package quantum_test

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

func TestFidelityDenseBellMatchesSparse(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	sparse, err := sparsestate.New(2)
	if err != nil {
		t.Fatalf("sparsestate.New: %v", err)
	}
	bellState(t, dense)
	bellState(t, sparse)

	got, err := quantum.Fidelity(dense, sparse)
	if err != nil {
		t.Fatalf("Fidelity: %v", err)
	}
	if math.Abs(got-1.0) > 1e-10 {
		t.Errorf("Fidelity(dense Bell, sparse Bell) = %v, want 1", got)
	}
}

func TestFidelityDistinguishesBellFromProduct(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	bellState(t, dense)

	product, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := product.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplyGate H: %v", err)
	}

	got, err := quantum.Fidelity(dense, product)
	if err != nil {
		t.Fatalf("Fidelity: %v", err)
	}
	// Overlap <Phi+|0+> = 1/2, so fidelity is 1/4.
	if math.Abs(got-0.25) > 1e-10 {
		t.Errorf("Fidelity(Bell, |0+>) = %v, want 0.25", got)
	}
}

func TestFidelityIsNonDestructive(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	bellState(t, dense)
	before0, before3 := dense.Amplitude(0), dense.Amplitude(3)

	product, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}

	if _, err := quantum.Fidelity(dense, product); err != nil {
		t.Fatalf("Fidelity: %v", err)
	}
	if got := dense.Amplitude(0); got != before0 {
		t.Errorf("amplitude 0 changed: %v -> %v", before0, got)
	}
	if got := dense.Amplitude(3); got != before3 {
		t.Errorf("amplitude 3 changed: %v -> %v", before3, got)
	}
}
