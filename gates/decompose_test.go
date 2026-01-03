package gates

import (
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/state"
)

func TestDecomposeSwapMatchesSwapGate(t *testing.T) {
	swapState := state.New(2)
	decomposedState := state.New(2)

	if err := swapState.ApplyGate(NewHadamard(), 0); err != nil {
		t.Fatalf("apply hadamard: %v", err)
	}
	if err := swapState.ApplyGate(NewPauliX(), 1); err != nil {
		t.Fatalf("apply pauli x: %v", err)
	}

	if err := decomposedState.ApplyGate(NewHadamard(), 0); err != nil {
		t.Fatalf("apply hadamard: %v", err)
	}
	if err := decomposedState.ApplyGate(NewPauliX(), 1); err != nil {
		t.Fatalf("apply pauli x: %v", err)
	}

	if err := swapState.ApplyGate(NewSwap(), 0, 1); err != nil {
		t.Fatalf("apply swap: %v", err)
	}

	for _, step := range DecomposeSwap(0, 1) {
		if err := decomposedState.ApplyGate(step.Gate, step.Targets...); err != nil {
			t.Fatalf("apply decomposed gate: %v", err)
		}
	}

	for basis := 0; basis < 4; basis++ {
		diff := cmplx.Abs(swapState.Amplitude(basis) - decomposedState.Amplitude(basis))
		if diff > 1e-12 {
			t.Fatalf("amplitude mismatch at %d: %v vs %v", basis, swapState.Amplitude(basis), decomposedState.Amplitude(basis))
		}
	}
}
