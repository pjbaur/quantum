package sparsestate

import (
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

func BenchmarkSparseSingleQubitGate(b *testing.B) {
	gate := gates.NewHadamard()
	qs := New(12)
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
	gate := gates.NewHadamard()
	qs := state.New(12)
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

func BenchmarkSparseCNOT(b *testing.B) {
	gate := gates.NewCNOT()
	qs := New(12)
	if err := qs.ApplyGate(gates.NewHadamard(), 3); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := qs.ApplyGate(gate, 3, 7); err != nil {
			b.Fatalf("ApplyGate failed: %v", err)
		}
	}
}

func BenchmarkDenseCNOT(b *testing.B) {
	gate := gates.NewCNOT()
	qs := state.New(12)
	if err := qs.ApplyGate(gates.NewHadamard(), 3); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := qs.ApplyGate(gate, 3, 7); err != nil {
			b.Fatalf("ApplyGate failed: %v", err)
		}
	}
}

func BenchmarkSparseProbabilityLookup(b *testing.B) {
	qs := New(14)
	if err := qs.ApplyGate(gates.NewHadamard(), 10); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = qs.Probability(1 << 10)
	}
}

func BenchmarkDenseProbabilityLookup(b *testing.B) {
	qs := state.New(14)
	if err := qs.ApplyGate(gates.NewHadamard(), 10); err != nil {
		b.Fatalf("setup ApplyGate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = qs.Probability(1 << 10)
	}
}
