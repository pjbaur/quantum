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

func TestBuiltinRegistryContents(t *testing.T) {
	registry := Builtin()
	want := []string{"CNOT", "Hadamard", "PauliX", "PauliY", "PauliZ", "S", "SWAP", "T", "Toffoli"}

	names := registry.Names()
	if len(names) != len(want) {
		t.Fatalf("Names() = %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("Names() = %v, want %v (sorted)", names, want)
		}
	}

	for _, name := range want {
		gate, ok := registry.Lookup(name)
		if !ok {
			t.Errorf("Lookup(%q) not found", name)
			continue
		}
		if gate.Name() != name {
			t.Errorf("Lookup(%q).Name() = %q", name, gate.Name())
		}
	}
}

func TestBuiltinRegistriesAreIndependent(t *testing.T) {
	first := Builtin()
	second := Builtin()

	extra, err := NewMatrixGate("CZ", [][]complex128{
		{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1},
	})
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}
	if err := first.Register(extra); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if _, ok := second.Lookup("CZ"); ok {
		t.Error("registering on one Builtin() registry leaked into another")
	}
}

func TestNamesOnNilRegistry(t *testing.T) {
	var registry *Registry
	if names := registry.Names(); names != nil {
		t.Errorf("nil registry Names() = %v, want nil", names)
	}
}
