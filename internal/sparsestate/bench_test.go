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
