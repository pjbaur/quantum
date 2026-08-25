package state

import (
	"errors"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)

func BenchmarkApplySingleQubitGate(b *testing.B) {
	state, _ := New(12)
	gate := gates.NewHadamard()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := state.ApplyGate(gate, 3); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkApplyMultiQubitGate(b *testing.B) {
	state, _ := New(12)
	gate := gates.NewCNOT()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := state.ApplyGate(gate, 2, 7); err != nil {
			b.Fatal(err)
		}
	}
}

type denseTwoQubitGate struct {
	matrix [][]complex128
}

func newDenseTwoQubitGate() *denseTwoQubitGate {
	const half = 0.5
	return &denseTwoQubitGate{
		matrix: [][]complex128{
			{half, half, half, half},
			{half, -half, half, -half},
			{half, half, -half, -half},
			{half, -half, -half, half},
		},
	}
}

func (g *denseTwoQubitGate) Apply(q quantum.Qubit) error {
	return errors.New("dense two-qubit gate requires multi-qubit apply")
}

func (g *denseTwoQubitGate) Name() string {
	return "DenseTwoQubit"
}

func (g *denseTwoQubitGate) Matrix() [][]complex128 {
	return g.matrix
}

func BenchmarkApplyGenericTwoQubitGate(b *testing.B) {
	state, _ := New(12)
	gate := newDenseTwoQubitGate()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := state.ApplyGate(gate, 2, 7); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkApplyThreeQubitGate exercises the combo-mask path at a width the
// sparse backend cannot reach, so the dense-only 3+ qubit branch has a
// timing baseline of its own.
func BenchmarkApplyThreeQubitGate(b *testing.B) {
	state, _ := New(12)
	gate, err := gates.NewMatrixGate("Toffoli", [][]complex128{
		{1, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 0, 0, 0, 1, 0},
	})
	if err != nil {
		b.Fatalf("setup NewMatrixGate failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := state.ApplyGate(gate, 2, 7, 9); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMeasure(b *testing.B) {
	gate := gates.NewHadamard()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		state, _ := New(12)
		if err := state.ApplyGate(gate, 0); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if _, err := state.Measure(0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClone(b *testing.B) {
	state, _ := New(12)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.Clone()
	}
}
