package sparsestate

import (
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

func BenchmarkSparseSingleQubitGate(b *testing.B) {
	gate := gates.NewHadamard()
	qs, err := New(12)
	if err != nil {
		b.Fatalf("setup New failed: %v", err)
	}
	if err := qs.ApplyGate(gate, 0); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := qs.ApplyGate(gate, 0); err != nil {
			b.Fatalf("ApplyGate failed: %v", err)
		}
	}
}

func BenchmarkDenseSingleQubitGate(b *testing.B) {
	qs, err := state.New(12)
	if err != nil {
		b.Fatalf("setup New failed: %v", err)
	}
	gate := gates.NewHadamard()
	if err := qs.ApplyGate(gate, 0); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := qs.ApplyGate(gate, 0); err != nil {
			b.Fatalf("ApplyGate failed: %v", err)
		}
	}
}
func BenchmarkSparseCNOTGate(b *testing.B) {
	qs, err := New(12)
	if err != nil {
		b.Fatalf("setup New failed: %v", err)
	}
	gate := gates.NewCNOT()
	if err := qs.ApplyGate(gate, 0, 1); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := qs.ApplyGate(gate, 0, 1); err != nil {
			b.Fatalf("ApplyGate failed: %v", err)
		}
	}
}
func BenchmarkDenseCNOTGate(b *testing.B) {
	qs, err := state.New(12)
	if err != nil {
		b.Fatalf("setup New failed: %v", err)
	}
	gate := gates.NewCNOT()
	if err := qs.ApplyGate(gate, 0, 1); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := qs.ApplyGate(gate, 0, 1); err != nil {
			b.Fatalf("ApplyGate failed: %v", err)
		}
	}
}

// BenchmarkSparseMeasure covers the collapse-and-renormalize path, which the
// gate benchmarks never touch. Six Hadamards spread the amplitude over 64 map
// entries so the collapse loop, not the fixed per-call overhead, dominates.
func BenchmarkSparseMeasure(b *testing.B) {
	gate := gates.NewHadamard()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		qs, err := New(12)
		if err != nil {
			b.Fatalf("setup New failed: %v", err)
		}
		for qubit := 0; qubit < 6; qubit++ {
			if err := qs.ApplyGate(gate, qubit); err != nil {
				b.Fatalf("setup ApplyGate failed: %v", err)
			}
		}
		b.StartTimer()
		if _, err := qs.Measure(0); err != nil {
			b.Fatalf("Measure failed: %v", err)
		}
	}
}

func BenchmarkSparseSwapGate(b *testing.B) {
	qs, err := New(12)
	if err != nil {
		b.Fatalf("setup New failed: %v", err)
	}
	gate := gates.NewSwap()
	if err := qs.ApplyGate(gate, 0, 1); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := qs.ApplyGate(gate, 0, 1); err != nil {
			b.Fatalf("ApplyGate failed: %v", err)
		}
	}
}
func BenchmarkDenseSwapGate(b *testing.B) {
	qs, err := state.New(12)
	if err != nil {
		b.Fatalf("setup New failed: %v", err)
	}
	gate := gates.NewSwap()
	if err := qs.ApplyGate(gate, 0, 1); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := qs.ApplyGate(gate, 0, 1); err != nil {
			b.Fatalf("ApplyGate failed: %v", err)
		}
	}
}
