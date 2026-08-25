/*
This `internal/examples/gates.go` file demonstrates the data-driven gate
catalog and gate decomposition:

 1. `GateCatalogDemo()` - Iterates the built-in gate registry and prints each
    gate's matrix, showing that every gate is just a named unitary matrix.
 2. `SwapDecompositionDemo()` - Verifies SWAP ≡ CNOT·CNOT·CNOT by applying
    both to identical two-qubit states and comparing amplitudes.
 3. `RunAllGatesDemos()` - A convenience function that runs all demonstrations.
*/

package examples

import (
	"fmt"
	"math/cmplx"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// printGateMatrix prints a gate matrix in rows of "a+bi" entries.
func printGateMatrix(matrix [][]complex128) {
	for _, row := range matrix {
		fmt.Print("  [")
		for j, v := range row {
			if j > 0 {
				fmt.Print("  ")
			}
			fmt.Printf("%6.3f%+.3fi", real(v), imag(v))
		}
		fmt.Println("]")
	}
}

// GateCatalogDemo lists every built-in gate with its matrix.
func GateCatalogDemo() {
	fmt.Println("\n=== Built-in Gate Catalog ===")
	fmt.Println("Every gate is a named unitary matrix in a registry.")

	registry := gates.Builtin()
	for _, name := range registry.Names() {
		gate, ok := registry.Lookup(name)
		if !ok {
			fmt.Printf("Error: gate %q missing from registry\n", name)
			return
		}
		fmt.Printf("\n%s:\n", name)
		printGateMatrix(gate.Matrix())
	}
}

// SwapDecompositionDemo shows that SWAP equals three alternating CNOTs.
func SwapDecompositionDemo() {
	fmt.Println("\n=== SWAP Decomposition Demonstration ===")
	fmt.Println("SWAP(a,b) = CNOT(a,b) · CNOT(b,a) · CNOT(a,b)")
	fmt.Println("Prepare (H⊗T)|00⟩ two ways and compare amplitudes.")

	direct, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	decomposed, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}

	// Identical non-trivial preparation on both states.
	for _, s := range []*state.State{direct, decomposed} {
		if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
			fmt.Printf("Error applying Hadamard gate: %v\n", err)
			return
		}
		if err := s.ApplyGate(gates.NewT(), 1); err != nil {
			fmt.Printf("Error applying T gate: %v\n", err)
			return
		}
	}

	if err := direct.ApplyGate(gates.NewSwap(), 0, 1); err != nil {
		fmt.Printf("Error applying SWAP gate: %v\n", err)
		return
	}
	for _, step := range gates.DecomposeSwap(0, 1) {
		if err := decomposed.ApplyGate(step.Gate, step.Targets...); err != nil {
			fmt.Printf("Error applying %s gate: %v\n", step.Gate.Name(), err)
			return
		}
	}

	fmt.Println("\nbasis  |  SWAP gate           |  CNOT decomposition")
	maxDiff := 0.0
	for basis := 0; basis < 4; basis++ {
		a := direct.Amplitude(basis)
		b := decomposed.Amplitude(basis)
		if diff := cmplx.Abs(a - b); diff > maxDiff {
			maxDiff = diff
		}
		fmt.Printf("|%02b⟩   | %8.4f%+.4fi  | %8.4f%+.4fi\n",
			basis, real(a), imag(a), real(b), imag(b))
	}

	if maxDiff < 1e-10 {
		fmt.Printf("\nAmplitudes match (max difference %.2e): the decomposition is exact.\n", maxDiff)
	} else {
		fmt.Printf("\nAmplitudes DO NOT match (max difference %.2e).\n", maxDiff)
	}
}

// RunAllGatesDemos runs all gate catalog demonstrations in sequence.
func RunAllGatesDemos() {
	fmt.Println("\n========================================")
	fmt.Println("       GATE CATALOG DEMONSTRATIONS")
	fmt.Println("========================================")

	GateCatalogDemo()
	SwapDecompositionDemo()

	fmt.Println("\n=== All gate demonstrations completed ===")
}
