package gates

import "testing"

func TestRegistryRegisterLookup(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(NewHadamard()); err != nil {
		t.Fatalf("register hadamard: %v", err)
	}

	gate, ok := registry.Lookup("Hadamard")
	if !ok {
		t.Fatal("expected hadamard to be registered")
	}
	if gate.Name() != "Hadamard" {
		t.Fatalf("unexpected gate: %s", gate.Name())
	}
}

func TestRegistryRejectsDuplicate(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(NewPauliX()); err != nil {
		t.Fatalf("register pauli x: %v", err)
	}

	if err := registry.Register(NewPauliX()); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}
