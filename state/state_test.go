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

const tolerance = 1e-10

func TestApplyGateCNOTControlBehavior(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(quantum.QuantumState) error
		targets   []int
		wantIndex int
	}{
		{
			name: "control |0> leaves |00> unchanged",
			setup: func(quantum.QuantumState) error {
				return nil
			},
			targets:   []int{1, 0},
			wantIndex: 0,
		},
		{
			name: "control |1> flips target",
			setup: func(qs quantum.QuantumState) error {
				return qs.ApplyGate(gates.NewPauliX(), 1)
			},
			targets:   []int{1, 0},
			wantIndex: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var qs quantum.QuantumState = state.New(2)
			if err := tt.setup(qs); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			if err := qs.ApplyGate(gates.NewCNOT(), tt.targets...); err != nil {
				t.Fatalf("ApplyGate returned error: %v", err)
			}

			assertBasisState(t, qs, tt.wantIndex)
			assertNormalized(t, qs)
		})
	}
}

func TestApplyGateSwap(t *testing.T) {
	var qs quantum.QuantumState = state.New(2)
	if err := qs.ApplyGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("ApplyGate(PauliX) returned error: %v", err)
	}

	if err := qs.ApplyGate(gates.NewSwap(), 0, 1); err != nil {
		t.Fatalf("ApplyGate(SWAP) returned error: %v", err)
	}

	assertBasisState(t, qs, 2)
	assertNormalized(t, qs)
}

func TestApplyGateBellStates(t *testing.T) {
	invSqrt2 := 1 / math.Sqrt(2)
	tests := []struct {
		name      string
		postGates []struct {
			gate   quantum.Gate
			target int
		}
		want [4]complex128
	}{
		{
			name: "Phi+",
			want: [4]complex128{
				complex(invSqrt2, 0),
				0,
				0,
				complex(invSqrt2, 0),
			},
		},
		{
			name: "Phi-",
			postGates: []struct {
				gate   quantum.Gate
				target int
			}{
				{gate: gates.NewPauliZ(), target: 1},
			},
			want: [4]complex128{
				complex(invSqrt2, 0),
				0,
				0,
				complex(-invSqrt2, 0),
			},
		},
		{
			name: "Psi+",
			postGates: []struct {
				gate   quantum.Gate
				target int
			}{
				{gate: gates.NewPauliX(), target: 1},
			},
			want: [4]complex128{
				0,
				complex(invSqrt2, 0),
				complex(invSqrt2, 0),
				0,
			},
		},
		{
			name: "Psi-",
			postGates: []struct {
				gate   quantum.Gate
				target int
			}{
				{gate: gates.NewPauliX(), target: 1},
				{gate: gates.NewPauliZ(), target: 1},
			},
			want: [4]complex128{
				0,
				complex(invSqrt2, 0),
				complex(-invSqrt2, 0),
				0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var qs quantum.QuantumState = state.New(2)
			if err := qs.ApplyGate(gates.NewHadamard(), 0); err != nil {
				t.Fatalf("ApplyGate(Hadamard) returned error: %v", err)
			}
			if err := qs.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
				t.Fatalf("ApplyGate(CNOT) returned error: %v", err)
			}

			for _, step := range tt.postGates {
				if err := qs.ApplyGate(step.gate, step.target); err != nil {
					t.Fatalf("ApplyGate(%s) returned error: %v", step.gate.Name(), err)
				}
			}

			for i, want := range tt.want {
				assertAmplitude(t, qs, i, want)
			}

			for i := 0; i < len(tt.want); i++ {
				wantProb := cmplx.Abs(tt.want[i])
				wantProb *= wantProb
				assertProbability(t, qs, i, wantProb)
			}
			assertNormalized(t, qs)
		})
	}
}

func TestApplyGateErrors(t *testing.T) {
	tests := []struct {
		name     string
		gate     quantum.Gate
		targets  []int
		errCheck func(error) bool
	}{
		{
			name:    "target out of range",
			gate:    gates.NewHadamard(),
			targets: []int{2},
			errCheck: func(err error) bool {
				var targetErr *quantum.QubitsOutOfRangeError
				return errors.As(err, &targetErr)
			},
		},
		{
			name:    "invalid target count for CNOT",
			gate:    gates.NewCNOT(),
			targets: []int{0},
			errCheck: func(err error) bool {
				var targetErr *quantum.InvalidGateApplicationError
				if !errors.As(err, &targetErr) {
					return false
				}
				return targetErr.RequiredLen == 2 && targetErr.ActualLen == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var qs quantum.QuantumState = state.New(2)
			err := qs.ApplyGate(tt.gate, tt.targets...)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.errCheck(err) {
				t.Fatalf("unexpected error type: %T", err)
			}
		})
	}
}

func assertBasisState(t *testing.T, qs quantum.QuantumState, wantIndex int) {
	t.Helper()

	totalStates := 1 << qs.NumQubits()
	for i := 0; i < totalStates; i++ {
		amp := qs.Amplitude(i)
		if i == wantIndex {
			if cmplx.Abs(amp-1) > tolerance {
				t.Fatalf("expected basis state %d amplitude 1, got %v", wantIndex, amp)
			}
			continue
		}
		if cmplx.Abs(amp) > tolerance {
			t.Fatalf("expected basis state %d amplitude 0, got %v", i, amp)
		}
	}
}

func assertAmplitude(t *testing.T, qs quantum.QuantumState, index int, want complex128) {
	t.Helper()

	amp := qs.Amplitude(index)
	if cmplx.Abs(amp-want) > tolerance {
		t.Fatalf("expected amplitude %v at index %d, got %v", want, index, amp)
	}
}

func assertProbability(t *testing.T, qs quantum.QuantumState, index int, want float64) {
	t.Helper()

	prob := qs.Probability(index)
	if math.Abs(prob-want) > tolerance {
		t.Fatalf("expected probability %v at index %d, got %v", want, index, prob)
	}
}

func assertNormalized(t *testing.T, qs quantum.QuantumState) {
	t.Helper()

	sum := 0.0
	totalStates := 1 << qs.NumQubits()
	for i := 0; i < totalStates; i++ {
		sum += qs.Probability(i)
	}
	if math.Abs(sum-1.0) > tolerance {
		t.Fatalf("expected normalized state, probability sum = %v", sum)
	}
}
