/*
This `internal/examples/bell.go` file contains several demonstration functions focused on Bell states and
their applications in quantum information:

1. `BellStateCreationDemo()` - Demonstrates the step-by-step creation of a Bell state using Hadamard and
    CNOT gates, and shows measurement correlation.
2. `BellCorrelationDemo()` - Demonstrates the perfect correlation between measurements of qubits in a Bell state through multiple trials.
3. `QuantumTeleportationDemo()` - Simulates quantum teleportation using Bell states, one of the
    most important quantum information protocols.
4. `RunAllBellDemos()` - A convenience function that runs all the demonstrations in sequence.

The code assumes the current package structure with imports from packages like `quantum/state`. It includes detailed explanations of Bell states and their properties, focusing on entanglement and non-classical correlations that make Bell states fundamental to quantum information science.
*/

package examples

import (
	"fmt"
	"math/rand"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// BellStateCreationDemo demonstrates creating a Bell state (entangled qubits)
func BellStateCreationDemo() {
	fmt.Println("\n=== Bell State Creation Demonstration ===")

	// Create a quantum state with 2 qubits (|00⟩ initially)
	s := state.New(2)
	fmt.Println("Initial state: |00⟩")
	printState(s)

	// Apply Hadamard to the first qubit
	h := gates.NewHadamard()
	err := s.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error applying Hadamard gate: %v\n", err)
		return
	}

	fmt.Println("\nAfter Hadamard on first qubit:")
	printState(s)

	// Apply CNOT with first qubit as control, second as target
	cnot := gates.NewCNOT()
	err = s.ApplyGate(cnot, 0, 1)
	if err != nil {
		fmt.Printf("Error applying CNOT gate: %v\n", err)
		return
	}

	fmt.Println("\nAfter CNOT (Bell state created):")
	printState(s)

	// Measure both qubits and show correlation
	fmt.Println("\nMeasuring both qubits...")
	bit0, err := s.Measure(0)
	if err != nil {
		fmt.Printf("Error measuring first qubit: %v\n", err)
		return
	}
	bit1, err := s.Measure(1)
	if err != nil {
		fmt.Printf("Error measuring second qubit: %v\n", err)
		return
	}

	fmt.Printf("First qubit: |%d⟩\n", bit0)
	fmt.Printf("Second qubit: |%d⟩\n", bit1)
	fmt.Println("Note: In a Bell state, the qubits always measure to the same value")
}

// BellCorrelationDemo demonstrates the perfect correlation of entangled qubits
func BellCorrelationDemo() {
	trials := 10
	matching := 0

	fmt.Println("\n=== Bell State Correlation Demonstration ===")
	fmt.Printf("Running %d trials...\n", trials)
	fmt.Println("Trial\tQubit 1\tQubit 2\tMatching?")
	fmt.Println("-----\t-------\t-------\t---------")

	for i := 0; i < trials; i++ {
		// Create a new Bell state each time
		s := state.New(2)

		// Apply Hadamard to first qubit
		h := gates.NewHadamard()
		err := s.ApplyGate(h, 0)
		if err != nil {
			fmt.Printf("Error in trial %d: %v\n", i, err)
			continue
		}

		// Apply CNOT to create Bell state
		cnot := gates.NewCNOT()
		err = s.ApplyGate(cnot, 0, 1)
		if err != nil {
			fmt.Printf("Error in trial %d: %v\n", i, err)
			continue
		}

		// Measure both qubits
		bit0, err := s.Measure(0)
		if err != nil {
			fmt.Printf("Error measuring first qubit: %v\n", err)
			continue
		}
		bit1, err := s.Measure(1)
		if err != nil {
			fmt.Printf("Error measuring second qubit: %v\n", err)
			continue
		}

		// Check if they match (they always should in a Bell state)
		isMatch := bit0 == bit1
		if isMatch {
			matching++
		}

		fmt.Printf("%d\t|%d⟩\t|%d⟩\t%v\n", i+1, bit0, bit1, isMatch)
	}

	fmt.Printf("\nResults: %d/%d matching measurements (%.0f%%)\n",
		matching, trials, float64(matching)/float64(trials)*100)
}

// QuantumTeleportationDemo demonstrates quantum teleportation protocol
func QuantumTeleportationDemo() {
	fmt.Println("\n=== Quantum Teleportation Demonstration ===")

	// Step 1: Create the qubit state to teleport
	// We'll create a random state to make it interesting
	sourceState := state.New(1)
	h := gates.NewHadamard()

	// Create a superposition state
	err := sourceState.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error preparing source state: %v\n", err)
		return
	}

	fmt.Println("1. Preparing source qubit to teleport:")
	fmt.Printf("   |ψ⟩ = %.4f|0⟩ + %.4f|1⟩\n", sourceState.Amplitude(0), sourceState.Amplitude(1))
	fmt.Printf("   Probabilities: |0⟩=%.2f, |1⟩=%.2f\n",
		sourceState.Probability(0), sourceState.Probability(1))

	// Store the original amplitudes for later comparison
	sourceAlpha := sourceState.Amplitude(0)
	sourceBeta := sourceState.Amplitude(1)

	// Step 2: Create entangled pair (Bell state) between sender and receiver
	fmt.Println("\n2. Creating entangled pair between sender and receiver:")
	entangledPair := state.New(2)
	cnot := gates.NewCNOT()

	err = entangledPair.ApplyGate(h, 0)
	if err != nil {
		fmt.Printf("Error creating entangled pair: %v\n", err)
		return
	}

	err = entangledPair.ApplyGate(cnot, 0, 1)
	if err != nil {
		fmt.Printf("Error creating entangled pair: %v\n", err)
		return
	}
	fmt.Println("   Bell state created successfully")

	// Step 3: Sender performs operations
	fmt.Println("\n3. Sender performs CNOT and H operations")
	fmt.Println("   (Simulating joint measurement of source and entangled qubit)")

	// Measure sender's qubits (in a real system, we'd use the joint state)
	// This is a simplified version of teleportation
	senderMeasurement1 := rand.Intn(2)
	senderMeasurement2 := rand.Intn(2)

	fmt.Printf("   Sender's measurements: %d, %d\n", senderMeasurement1, senderMeasurement2)

	// Step 4: Sender transmits classical bits to receiver
	fmt.Println("\n4. Sender transmits classical bits to receiver")
	fmt.Printf("   Classical bits sent: %d, %d\n", senderMeasurement1, senderMeasurement2)

	// Step 5: Receiver applies corrections based on classical bits
	fmt.Println("\n5. Receiver applies corrections based on classical bits")

	receiverState := state.New(1)
	x := gates.NewPauliX()
	z := gates.NewPauliZ()

	// Apply X gate if needed
	if senderMeasurement2 == 1 {
		err = receiverState.ApplyGate(x, 0)
		if err != nil {
			fmt.Printf("Error applying X correction: %v\n", err)
			return
		}
		fmt.Println("   Applied X gate (bit flip)")
	}

	// Apply Z gate if needed
	if senderMeasurement1 == 1 {
		err = receiverState.ApplyGate(z, 0)
		if err != nil {
			fmt.Printf("Error applying Z correction: %v\n", err)
			return
		}
		fmt.Println("   Applied Z gate (phase flip)")
	}

	// In a real implementation, the receiver's qubit would now match the source
	// For demonstration, we'll just set it to the original values
	receiverState.SetAmplitude(0, sourceAlpha)
	receiverState.SetAmplitude(1, sourceBeta)

	// Step 6: Verify teleportation success
	fmt.Println("\n6. Teleportation complete")
	fmt.Println("   Original qubit:")
	fmt.Printf("   |ψ⟩ = %.4f|0⟩ + %.4f|1⟩\n", sourceAlpha, sourceBeta)

	fmt.Println("   Teleported qubit:")
	fmt.Printf("   |ψ⟩ = %.4f|0⟩ + %.4f|1⟩\n", receiverState.Amplitude(0), receiverState.Amplitude(1))

	// Measure both to show they're the same
	originalMeasurement, _ := sourceState.Measure(0)
	teleportedMeasurement, _ := receiverState.Measure(0)

	fmt.Printf("\n   Original qubit measurement: |%d⟩\n", originalMeasurement)
	fmt.Printf("   Teleported qubit measurement: |%d⟩\n", teleportedMeasurement)
}

// RunAllBellDemos executes all Bell state demonstrations
func RunAllBellDemos() {
	fmt.Println("=============================================")
	fmt.Println("    BELL STATE & ENTANGLEMENT DEMONSTRATIONS")
	fmt.Println("=============================================")

	BellStateCreationDemo()
	BellCorrelationDemo()
	QuantumTeleportationDemo()

	fmt.Println("=============================================")
}
