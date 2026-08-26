package examples

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// TGateSingleQubitDemo demonstrates applying a T gate to a single qubit
// and shows the phase change effect
func TGateSingleQubitDemo() {
	fmt.Println("\n=== T Gate Demonstration ===")

	// Create a 1-qubit state and apply Hadamard to create superposition
	s, err := state.New(1)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	h := gates.NewHadamard()
	err = s.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error applying Hadamard gate: %v\n", err)
		return
	}

	fmt.Printf("Initial superposition: |+⟩ (α=%v, β=%v)\n", s.Amplitude(0), s.Amplitude(1))
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		s.Probability(0), s.Probability(1))

	// Apply T gate (π/8 phase rotation)
	t := gates.NewT()
	err = s.ApplyGate(t, 0)
	if err != nil {
		fmt.Printf("Error applying T gate: %v\n", err)
		return
	}

	fmt.Printf("\nAfter T gate: (α=%v, β=%v)\n", s.Amplitude(0), s.Amplitude(1))
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		s.Probability(0), s.Probability(1))
	fmt.Println("Note: The probabilities remain unchanged, but the phase of |1⟩ changed by π/4")

	// Apply T gate again to demonstrate cumulative effect
	err = s.ApplyGate(t, 0)
	if err != nil {
		fmt.Printf("Error applying second T gate: %v\n", err)
		return
	}

	fmt.Printf("\nAfter second T gate: (α=%v, β=%v)\n", s.Amplitude(0), s.Amplitude(1))
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		s.Probability(0), s.Probability(1))
	fmt.Println("Note: Two T gates equal one S gate (π/4 phase)")
}

// TGatePhaseRotationDemo demonstrates how the T gate rotates phase on the Bloch sphere
func TGatePhaseRotationDemo() {
	fmt.Println("\n=== T Gate Phase Rotation Demonstration ===")

	// Create a 1-qubit state and apply Hadamard to create superposition
	s, err := state.New(1)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	h := gates.NewHadamard()
	err = s.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error creating superposition: %v\n", err)
		return
	}

	// Show initial state
	fmt.Println("Initial superposition state:")
	printStatePhaseInfo(s)

	// Apply T gate multiple times to show rotation
	t := gates.NewT()
	for i := 1; i <= 8; i++ {
		err = s.ApplyGate(t, 0)
		if err != nil {
			fmt.Printf("Error applying T gate: %v\n", err)
			return
		}

		fmt.Printf("\nAfter %d T gate(s) (rotation by %dπ/4):\n", i, i)
		printStatePhaseInfo(s)
	}

	fmt.Println("\nNote: After 8 T gates, we've rotated by 2π and returned to the original state")
}

// Helper function to print phase information for a state
func printStatePhaseInfo(s *state.State) {
	alpha := s.Amplitude(0)
	beta := s.Amplitude(1)

	// Calculate phase angle in degrees
	phase := cmplx.Phase(beta/alpha) * 180 / math.Pi

	fmt.Printf("State: α=%v, β=%v\n", alpha, beta)
	fmt.Printf("Phase angle: %.2f degrees\n", phase)
	fmt.Printf("Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		s.Probability(0), s.Probability(1))
}

// TGateMultiQubitDemo demonstrates T gates in a multi-qubit system
func TGateMultiQubitDemo() {
	fmt.Println("\n=== T Gate in Multi-Qubit System ===")

	// Create a 2-qubit system
	s, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
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
	tErr := s.ApplyGate(t, 0)
	if tErr != nil {
		fmt.Printf("Error applying T gate: %v\n", tErr)
		return
	}

	fmt.Println("\nAfter T gate on first qubit:")
	printState(s)

	// Apply T gate to second qubit
	tErr = s.ApplyGate(t, 1)
	if tErr != nil {
		fmt.Printf("Error applying T gate: %v\n", tErr)
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

	// Initialize two 1-qubit states
	s1, err := state.New(1) // For T gate demo
	if err != nil {
		fmt.Printf("Error creating state s1: %v\n", err)
		return
	}
	s2, err := state.New(1) // For Hadamard demo
	if err != nil {
		fmt.Printf("Error creating state s2: %v\n", err)
		return
	}

	// Create gates
	h := gates.NewHadamard()
	t := gates.NewT()

	// Apply Hadamard to both states first
	err = s1.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error applying Hadamard to s1: %v\n", err)
		return
	}

	err = s2.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error applying Hadamard to s2: %v\n", err)
		return
	}

	fmt.Println("Both states start in superposition state |+⟩:")
	fmt.Printf("s1: α=%v, β=%v\n", s1.Amplitude(0), s1.Amplitude(1))
	fmt.Printf("s2: α=%v, β=%v\n", s2.Amplitude(0), s2.Amplitude(1))

	// Apply T to s1 and H again to s2
	err = s1.ApplyGate(t, 0)
	if err != nil {
		fmt.Printf("Error applying T gate: %v\n", err)
		return
	}

	err = s2.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error applying second Hadamard: %v\n", err)
		return
	}

	fmt.Println("\nAfter applying T to s1 and second H to s2:")
	fmt.Printf("s1 (T|+⟩): α=%v, β=%v\n", s1.Amplitude(0), s1.Amplitude(1))
	fmt.Printf("s2 (H|+⟩): α=%v, β=%v\n", s2.Amplitude(0), s2.Amplitude(1))

	fmt.Println("\nProbabilities:")
	fmt.Printf("s1 (T|+⟩): |0⟩=%.2f, |1⟩=%.2f\n", s1.Probability(0), s1.Probability(1))
	fmt.Printf("s2 (H|+⟩): |0⟩=%.2f, |1⟩=%.2f\n", s2.Probability(0), s2.Probability(1))

	fmt.Println("\nKey difference: T gate changes phase but preserves superposition probabilities.")
	fmt.Println("Hadamard applied twice returns to the original state (H²=I).")
}

// RunAllTGateDemos executes all T gate demonstrations
func RunAllTGateDemos() {
	PrintBanner(45, "    T GATE DEMONSTRATIONS")

	TGateSingleQubitDemo()
	TGatePhaseRotationDemo()
	TGateMultiQubitDemo()
	TGateVsHadamardDemo()

	PrintRule(45)
}
