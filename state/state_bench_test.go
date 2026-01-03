package state

import (
	"testing"

	"github.com/pjbaur/quantum/gates"
)

func BenchmarkApplySingleQubitGate(b *testing.B) {
	state := New(12)
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
	state := New(12)
	gate := gates.NewCNOT()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := state.ApplyGate(gate, 2, 7); err != nil {
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
		state := New(12)
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
	state := New(12)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.Clone()
	}
}
