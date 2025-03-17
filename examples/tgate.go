package examples

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"

	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/state"
)

// TGateSingleQubitDemo demonstrates applying a T-gate to a single qubit
// and shows the resulting phase shift
func TGateSingleQubitDemo() {
	fmt.Println("Single Qubit T-Gate Demonstration")
	fmt.Println("--------------------------------")

	// Create a new qubit in state |0⟩
	q := qubit.New()
	fmt.Printf("Initial state |0⟩: α=%.3f+%.3fi, β=%.3f+%.3fi\n",
		real(q.Alpha), imag(q.Alpha), real(q.Beta), imag(q.Beta))

	// Apply Hadamard to get superposition
	q.ApplyHadamard()
	fmt.Printf("After Hadamard: α=%.3f+%.3fi, β=%.3f+%.3fi\n",
		real(q.Alpha), imag(q.Alpha), real(q.Beta), imag(q.Beta))

	// Apply T-gate (π/4 phase shift)
	q.ApplyT()
	fmt.Printf("After T-gate: α=%.3f+%.3fi, β=%.3f+%.3fi\n",
		real(q.Alpha), imag(q.Alpha), real(q.Beta), imag(q.Beta))

	// The T-gate adds a π/4 phase to the |1⟩ component
	fmt.Println("\nNote: The T-gate applied a π/4 phase shift to the |1⟩ component.")
	fmt.Printf("The phase of β is now: %.3f radians (%.1f degrees)\n",
		cmplx.Phase(q.Beta), cmplx.Phase(q.Beta)*180/math.Pi)
}

// TGateMultipleDemo demonstrates applying T-gates multiple times
// to observe the cumulative phase shift
func TGateMultipleDemo() {
	fmt.Println("\nMultiple T-Gate Applications Demonstration")
	fmt.Println("---------------------------------------")

	// Create a new qubit in state |0⟩
	q := qubit.New()

	// Apply Hadamard to get superposition
	q.ApplyHadamard()
	fmt.Printf("After Hadamard: α=%.3f+%.3fi, β=%.3f+%.3fi\n",
		real(q.Alpha), imag(q.Alpha), real(q.Beta), imag(q.Beta))

	// Apply T-gate multiple times
	for i := 1; i <= 8; i++ {
		q.ApplyT()
		phase := cmplx.Phase(q.Beta)
		fmt.Printf("After T-gate #%d: α=%.3f+%.3fi, β=%.3f+%.3fi (phase=%.1f°)\n",
			i, real(q.Alpha), imag(q.Alpha), real(q.Beta), imag(q.Beta),
			phase*180/math.Pi)

		// After 8 T-gates (equivalent to a full 2π rotation), we should be back to the original state
		if i == 8 {
			fmt.Println("\nNote: After 8 applications of the T-gate, we've completed a full 2π rotation")
			fmt.Println("and returned close to our initial superposition state.")
		}
	}
}

// TGateAndHadamardInterferenceDemo demonstrates the interference effects
// of combining T-gates with Hadamard gates
func TGateAndHadamardInterferenceDemo() {
	trials := 1000
	outcomes := make(map[int]int)

	fmt.Println("\nT-Gate and Hadamard Interference Demonstration")
	fmt.Println("-------------------------------------------")
	fmt.Printf("Running %d trials\n", trials)

	for i := 0; i < trials; i++ {
		q := qubit.New()

		// Apply Hadamard to create superposition
		q.ApplyHadamard()

		// Apply T-gate
		q.ApplyT()

		// Apply Hadamard again
		q.ApplyHadamard()

		// Measure the qubit
		result := q.Measure()
		outcomes[result]++
	}

	// Display the results
	probability0 := float64(outcomes[0]) / float64(trials) * 100
	probability1 := float64(outcomes[1]) / float64(trials) * 100

	fmt.Println("\nResults after H → T → H → Measure:")
	fmt.Printf("|0⟩: %d occurrences (%.1f%%)\n", outcomes[0], probability0)
	fmt.Printf("|1⟩: %d occurrences (%.1f%%)\n", outcomes[1], probability1)

	// The theoretical probabilities for H→T→H are approximately 85% for |0⟩ and 15% for |1⟩
	fmt.Println("\nNote: The theoretical probabilities for this sequence are approximately")
	fmt.Println("85% for |0⟩ and 15% for |1⟩, due to quantum interference effects.")
}

// TGateInQuantumStateDemo demonstrates using T-gate in a multi-qubit system
func TGateInQuantumStateDemo() {
	fmt.Println("\nT-Gate in Multi-Qubit System Demonstration")
	fmt.Println("----------------------------------------")

	// Create a 2-qubit quantum state
	qs := state.New(2)
	fmt.Println("Initial state:")
	qs.PrintState()

	// Apply Hadamard to both qubits
	qs.ApplyHadamard(0)
	qs.ApplyHadamard(1)
	fmt.Println("\nAfter Hadamard on both qubits:")
	qs.PrintState()

	// Apply T-gate to the first qubit
	qs.ApplyT(0)
	fmt.Println("\nAfter T-gate on qubit 0:")
	qs.PrintState()

	// Apply T-gate to the second qubit
	qs.ApplyT(1)
	fmt.Println("\nAfter T-gate on qubit 1:")
	qs.PrintState()

	// Notice the phase changes in the state vector
	fmt.Println("\nNote: Observe how the T-gates have added phase shifts to different")
	fmt.Println("components of the state vector based on qubit values.")
}

// TDaggerDemo demonstrates the T-dagger gate (T†), which is the inverse of the T-gate
func TDaggerDemo() {
	fmt.Println("\nT-Dagger Gate Demonstration")
	fmt.Println("--------------------------")

	// Create a qubit in state |0⟩
	q := qubit.New()

	// Apply Hadamard to get into superposition
	q.ApplyHadamard()
	fmt.Printf("After Hadamard: α=%.3f+%.3fi, β=%.3f+%.3fi\n",
		real(q.Alpha), imag(q.Alpha), real(q.Beta), imag(q.Beta))

	// Apply T-gate
	q.ApplyT()
	fmt.Printf("After T-gate: α=%.3f+%.3fi, β=%.3f+%.3fi\n",
		real(q.Alpha), imag(q.Alpha), real(q.Beta), imag(q.Beta))

	// Apply T-dagger
	q.ApplyTDagger()
	fmt.Printf("After T-dagger: α=%.3f+%.3fi, β=%.3f+%.3fi\n",
		real(q.Alpha), imag(q.Alpha), real(q.Beta), imag(q.Beta))

	fmt.Println("\nNote: The T-dagger gate cancels out the effect of the T-gate,")
	fmt.Println("returning us to our original state after the Hadamard.")
}

// RunAllTGateDemos executes all T-gate demonstrations
func RunAllTGateDemos() {
	// Seed the random number generator
	rand.Seed(42) // Use a fixed seed for reproducible demos

	fmt.Println("=============================================")
	fmt.Println("       T-GATE DEMONSTRATIONS")
	fmt.Println("=============================================")

	// Run each demonstration
	TGateSingleQubitDemo()
	TGateMultipleDemo()
	TGateAndHadamardInterferenceDemo()
	TGateInQuantumStateDemo()
	TDaggerDemo()

	fmt.Println("=============================================")
}
