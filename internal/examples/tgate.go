package examples

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/state"
)

// TGateSingleQubitDemo demonstrates applying a T gate to a single qubit
// and shows the phase change effect
func TGateSingleQubitDemo() {
	fmt.Println("\n=== T Gate Demonstration ===")

	// Create a qubit in superposition using Hadamard
	q := qubit.New()
	h := gates.NewHadamard()
	err := h.Apply(q)
	if err != nil {
		fmt.Printf("Error applying Hadamard gate: %v\n", err)
		return
	}

	fmt.Printf("Initial superposition: |+⟩ (α=%v, β=%v)\n", q.Alpha(), q.Beta())
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		q.Probability0(), q.Probability1())

	// Apply T gate (π/8 phase rotation)
	t := gates.NewT()
	err = t.Apply(q)
	if err != nil {
		fmt.Printf("Error applying T gate: %v\n", err)
		return
	}

	fmt.Printf("\nAfter T gate: (α=%v, β=%v)\n", q.Alpha(), q.Beta())
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		q.Probability0(), q.Probability1())
	fmt.Println("Note: The probabilities remain unchanged, but the phase of |1⟩ changed by π/4")

	// Apply T gate again to demonstrate cumulative effect
	err = t.Apply(q)
	if err != nil {
		fmt.Printf("Error applying second T gate: %v\n", err)
		return
	}

	fmt.Printf("\nAfter second T gate: (α=%v, β=%v)\n", q.Alpha(), q.Beta())
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		q.Probability0(), q.Probability1())
	fmt.Println("Note: Two T gates equal one S gate (π/4 phase)")
}

// TGatePhaseRotationDemo demonstrates how the T gate rotates phase on the Bloch sphere
func TGatePhaseRotationDemo() {
	fmt.Println("\n=== T Gate Phase Rotation Demonstration ===")

	// Create a qubit in superposition
	q := qubit.New()
	h := gates.NewHadamard()
	err := h.Apply(q)
	if err != nil {
		fmt.Printf("Error creating superposition: %v\n", err)
		return
	}

	// Show initial state
	fmt.Println("Initial superposition state:")
	printPhaseInfo(q)

	// Apply T gate multiple times to show rotation
	t := gates.NewT()
	for i := 1; i <= 8; i++ {
		err = t.Apply(q)
		if err != nil {
			fmt.Printf("Error applying T gate: %v\n", err)
			return
		}

		fmt.Printf("\nAfter %d T gate(s) (rotation by %dπ/4):\n", i, i)
		printPhaseInfo(q)
	}

	fmt.Println("\nNote: After 8 T gates, we've rotated by 2π and returned to the original state")
}

// Helper function to print phase information
func printPhaseInfo(q qubit.Qubit) {
	alpha := q.Alpha()
	beta := q.Beta()

	// Calculate phase angle in degrees
	phase := cmplx.Phase(beta/alpha) * 180 / math.Pi

	fmt.Printf("State: α=%v, β=%v\n", alpha, beta)
	fmt.Printf("Phase angle: %.2f degrees\n", phase)
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		q.Probability0(), q.Probability1())
}

// TGateMultiQubitDemo demonstrates T gates in a multi-qubit system
func TGateMultiQubitDemo() {
	fmt.Println("\n=== T Gate in Multi-Qubit System ===")

	// Create a 2-qubit system
	s := state.New(2)
	h := gates.NewHadamard()
	t := gates.NewT()

	// Apply Hadamard to both qubits
	for i := 0; i < 2; i++ {
		err := s.ApplyGate(h, i)
		if err != nil {
			fmt.Printf("Error applying Hadamard to qubit %d: %v\n", i, err)
			return
		}
	}

	fmt.Println("Initial state (both qubits in superposition):")
	printState(s)

	// Apply T gate to first qubit
	err := s.ApplyGate(t, 0)
	if err != nil {
		fmt.Printf("Error applying T gate: %v\n", err)
		return
	}

	fmt.Println("\nAfter T gate on first qubit:")
	printState(s)

	// Apply T gate to second qubit
	err = s.ApplyGate(t, 1)
	if err != nil {
		fmt.Printf("Error applying T gate: %v\n", err)
		return
	}

	fmt.Println("\nAfter T gate on second qubit:")
	printState(s)

	// Measure both qubits
	results := make([]int, 2)
	for i := 0; i < 2; i++ {
		bit, err := s.Measure(i)
		if err != nil {
			fmt.Printf("Error measuring qubit %d: %v\n", i, err)
			return
		}
		results[i] = bit
	}

	fmt.Printf("\nMeasurement results: |%d%d⟩\n", results[0], results[1])
}

// TGateVsHadamardDemo compares T gate and Hadamard effects
func TGateVsHadamardDemo() {
	fmt.Println("\n=== T Gate vs. Hadamard Comparison ===")

	// Initialize two qubits
	q1 := qubit.New() // For T gate demo
	q2 := qubit.New() // For Hadamard demo

	// Create gates
	h := gates.NewHadamard()
	t := gates.NewT()

	// Apply Hadamard to both qubits first
	err := h.Apply(q1)
	if err != nil {
		fmt.Printf("Error applying Hadamard to q1: %v\n", err)
		return
	}

	err = h.Apply(q2)
	if err != nil {
		fmt.Printf("Error applying Hadamard to q2: %v\n", err)
		return
	}

	fmt.Println("Both qubits start in superposition state |+⟩:")
	fmt.Printf("q1: α=%v, β=%v\n", q1.Alpha(), q1.Beta())
	fmt.Printf("q2: α=%v, β=%v\n", q2.Alpha(), q2.Beta())

	// Apply T to q1 and H again to q2
	err = t.Apply(q1)
	if err != nil {
		fmt.Printf("Error applying T gate: %v\n", err)
		return
	}

	err = h.Apply(q2)
	if err != nil {
		fmt.Printf("Error applying second Hadamard: %v\n", err)
		return
	}

	fmt.Println("\nAfter applying T to q1 and second H to q2:")
	fmt.Printf("q1 (T|+⟩): α=%v, β=%v\n", q1.Alpha(), q1.Beta())
	fmt.Printf("q2 (H|+⟩): α=%v, β=%v\n", q2.Alpha(), q2.Beta())

	fmt.Println("\nProbabilities:")
	fmt.Printf("q1 (T|+⟩): |0⟩=%.2f, |1⟩=%.2f\n", q1.Probability0(), q1.Probability1())
	fmt.Printf("q2 (H|+⟩): |0⟩=%.2f, |1⟩=%.2f\n", q2.Probability0(), q2.Probability1())

	fmt.Println("\nKey difference: T gate changes phase but preserves superposition probabilities.")
	fmt.Println("Hadamard applied twice returns to the original state (H²=I).")
}

// RunAllTGateDemos executes all T gate demonstrations
func RunAllTGateDemos() {
	fmt.Println("=============================================")
	fmt.Println("    T GATE DEMONSTRATIONS")
	fmt.Println("=============================================")

	TGateSingleQubitDemo()
	TGatePhaseRotationDemo()
	TGateMultiQubitDemo()
	TGateVsHadamardDemo()

	fmt.Println("=============================================")
}

// Helper function to print state information
func printState(s state.State) {
	fmt.Println("State vector:")
	s.Print()
}
