package examples

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// DeutschJozsaDemo shows how the algorithm distinguishes constant vs balanced oracles.
func DeutschJozsaDemo() {
	fmt.Println("\n=== Deutsch-Jozsa Demonstration ===")

	// Two input qubits plus the ancilla. The algorithm runs on whatever
	// backend it is handed; the dense one is the general-purpose choice.
	constantStart, err := state.New(3)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	constantOracle := func(int) int { return 0 }
	constantState, err := algorithm.DeutschJozsa(constantStart, constantOracle)
	if err != nil {
		fmt.Printf("Error running Deutsch-Jozsa: %v\n", err)
		return
	}

	balancedStart, err := state.New(3)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	balancedOracle := func(input int) int { return input & 1 }
	balancedState, err := algorithm.DeutschJozsa(balancedStart, balancedOracle)
	if err != nil {
		fmt.Printf("Error running Deutsch-Jozsa: %v\n", err)
		return
	}

	constantProb := constantState.Probability(0) + constantState.Probability(1<<2)
	balancedProb := balancedState.Probability(0) + balancedState.Probability(1<<2)

	fmt.Printf("Constant oracle zero-register probability: %.2f\n", constantProb)
	fmt.Printf("Balanced oracle zero-register probability: %.2f\n", balancedProb)
}

// GroverDemo shows probability amplification for a marked state.
func GroverDemo() {
	fmt.Println("\n=== Grover Demonstration ===")

	start, err := state.New(3)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	searchState, err := algorithm.Grover(start, []int{5})
	if err != nil {
		fmt.Printf("Error running Grover: %v\n", err)
		return
	}

	fmt.Printf("Marked state probability: %.2f\n", searchState.Probability(5))
}

// Sizes for the diffusion demonstration below. Three qubits is the smallest
// register where the gate sequence needs a genuinely multi-controlled Z
// rather than a plain CZ, and it stays small enough to print in full.
const (
	diffusionQubits = 3
	diffusionMarked = 5 // |101⟩
	// diffusionIterations is what algorithm.Grover runs for one marked state
	// in a 3-qubit register: round((π/4)·√8) = 2. The gate circuit has to run
	// the same schedule for the end-to-end comparison to mean anything, so if
	// the algorithm's schedule ever changes, this demo starts reporting a
	// mismatch instead of quietly comparing two different circuits.
	diffusionIterations = 2
)

// applyToEveryQubit applies a single-qubit gate across the whole register.
func applyToEveryQubit(s quantum.QuantumState, gate quantum.Gate) error {
	for qubit := 0; qubit < s.NumQubits(); qubit++ {
		if err := s.ApplyGate(gate, qubit); err != nil {
			return fmt.Errorf("apply %s to qubit %d: %w", gate.Name(), qubit, err)
		}
	}
	return nil
}

// descendingTargets lists every qubit of the register most significant first,
// the order this library's multi-qubit gates take their targets in.
func descendingTargets(numQubits int) []int {
	targets := make([]int, numQubits)
	for i := range targets {
		targets[i] = numQubits - 1 - i
	}
	return targets
}

// multiControlledZ builds the gate that flips the sign of |1…1⟩ and leaves
// every other basis state alone: a Z gate with numQubits-1 controls stacked
// onto it, one gates.NewControlled call per control.
func multiControlledZ(numQubits int) (quantum.Gate, error) {
	gate := quantum.Gate(gates.NewPauliZ())
	for control := 1; control < numQubits; control++ {
		controlled, err := gates.NewControlled(gate)
		if err != nil {
			return nil, fmt.Errorf("stack control %d: %w", control, err)
		}
		gate = controlled
	}
	return gate, nil
}

// applyGateDiffusion applies Grover's diffusion operator, D = 2|s⟩⟨s| - I,
// as the standard gate sequence H^⊗n · X^⊗n · MCZ · X^⊗n · H^⊗n.
//
// That sequence is actually -D: sandwiching MCZ between the X layers gives
// I - 2|0…0⟩⟨0…0|, and conjugating by the Hadamards turns |0…0⟩ into the
// uniform superposition |s⟩, leaving the whole operator negated. The sign is
// a global phase and unobservable, so hardware ignores it; here Rz(2π) = -I
// on a single qubit puts it back, which lets the demo compare amplitudes
// entry for entry instead of only comparing probabilities.
func applyGateDiffusion(s quantum.QuantumState) error {
	mcz, err := multiControlledZ(s.NumQubits())
	if err != nil {
		return err
	}

	if err := applyToEveryQubit(s, gates.NewHadamard()); err != nil {
		return err
	}
	if err := applyToEveryQubit(s, gates.NewPauliX()); err != nil {
		return err
	}
	if err := s.ApplyGate(mcz, descendingTargets(s.NumQubits())...); err != nil {
		return fmt.Errorf("apply %s: %w", mcz.Name(), err)
	}
	if err := applyToEveryQubit(s, gates.NewPauliX()); err != nil {
		return err
	}
	if err := applyToEveryQubit(s, gates.NewHadamard()); err != nil {
		return err
	}
	if err := s.ApplyGate(gates.NewRz(2*math.Pi), 0); err != nil {
		return fmt.Errorf("apply global phase: %w", err)
	}
	return nil
}

// applyGatePhaseOracle flips the sign of the marked basis state, the other
// half of a Grover iteration, out of the same multi-controlled Z: the X gates
// move the marked state onto |1…1⟩, the one state MCZ can reach, and move it
// back afterwards.
func applyGatePhaseOracle(s quantum.QuantumState, marked int) error {
	mcz, err := multiControlledZ(s.NumQubits())
	if err != nil {
		return err
	}

	flipZeroBits := func() error {
		for qubit := 0; qubit < s.NumQubits(); qubit++ {
			if marked&(1<<qubit) != 0 {
				continue
			}
			if err := s.ApplyGate(gates.NewPauliX(), qubit); err != nil {
				return fmt.Errorf("apply PauliX to qubit %d: %w", qubit, err)
			}
		}
		return nil
	}

	if err := flipZeroBits(); err != nil {
		return err
	}
	if err := s.ApplyGate(mcz, descendingTargets(s.NumQubits())...); err != nil {
		return fmt.Errorf("apply %s: %w", mcz.Name(), err)
	}
	return flipZeroBits()
}

// applyClosedFormDiffusion applies the same operator as arithmetic:
// new_amp = 2·mean - old_amp. This is the O(2ⁿ) inversion about the average
// that the algorithm package uses, and it stays the fast path — the gate
// sequence above is the illustration, not a replacement for it.
func applyClosedFormDiffusion(s *state.State) error {
	size := 1 << s.NumQubits()
	amplitudes := make([]complex128, size)
	var sum complex128
	for basis := range amplitudes {
		amplitudes[basis] = s.Amplitude(basis)
		sum += amplitudes[basis]
	}

	mean := sum / complex(float64(size), 0)
	for basis := range amplitudes {
		amplitudes[basis] = 2*mean - amplitudes[basis]
	}
	return s.SetAmplitudes(amplitudes)
}

// midSearchState returns the register as the first Grover iteration finds it:
// a uniform superposition with the marked amplitude already sign-flipped.
// Both diffusion implementations start from this, so the comparison isolates
// the diffusion step.
func midSearchState() (*state.State, error) {
	s, err := state.New(diffusionQubits)
	if err != nil {
		return nil, err
	}
	if err := applyToEveryQubit(s, gates.NewHadamard()); err != nil {
		return nil, err
	}
	if err := applyGatePhaseOracle(s, diffusionMarked); err != nil {
		return nil, err
	}
	return s, nil
}

// printAmplitudeComparison prints two amplitude vectors side by side and
// reports the largest disagreement between them.
func printAmplitudeComparison(leftLabel, rightLabel string, left, right quantum.QuantumState) {
	numQubits := left.NumQubits()
	fmt.Printf("\nbasis  | %-20s | %s\n", leftLabel, rightLabel)

	maxDiff := 0.0
	for basis := 0; basis < 1<<numQubits; basis++ {
		leftAmp, rightAmp := left.Amplitude(basis), right.Amplitude(basis)
		if diff := cmplx.Abs(leftAmp - rightAmp); diff > maxDiff {
			maxDiff = diff
		}
		fmt.Printf("|%0*b⟩  | %8.4f%+.4fi     | %8.4f%+.4fi\n",
			numQubits, basis, real(leftAmp), imag(leftAmp), real(rightAmp), imag(rightAmp))
	}

	if maxDiff < 1e-10 {
		fmt.Printf("Amplitudes match (max difference %.2e).\n", maxDiff)
	} else {
		fmt.Printf("Amplitudes DO NOT match (max difference %.2e).\n", maxDiff)
	}
}

// GroverDiffusionGatesDemo builds Grover's diffusion operator out of gates and
// checks it against the closed-form inversion about the average that the
// algorithm package applies as its fast path — first as a single operator,
// then as the diffusion step of a whole search run against algorithm.Grover.
func GroverDiffusionGatesDemo() {
	fmt.Println("\n=== Grover Diffusion as a Gate Sequence ===")
	fmt.Printf("Diffusion D = 2|s⟩⟨s| - I on %d qubits, built two ways:\n", diffusionQubits)
	fmt.Println("  gate sequence: H⊗n · X⊗n · CCZ · X⊗n · H⊗n, then Rz(2π) = -I for the global phase")
	fmt.Println("  closed form:   new_amp = 2·mean - old_amp (the algorithm package's fast path)")

	gateState, err := midSearchState()
	if err != nil {
		fmt.Printf("Error preparing state: %v\n", err)
		return
	}
	closedState, err := midSearchState()
	if err != nil {
		fmt.Printf("Error preparing state: %v\n", err)
		return
	}

	if err := applyGateDiffusion(gateState); err != nil {
		fmt.Printf("Error applying gate diffusion: %v\n", err)
		return
	}
	if err := applyClosedFormDiffusion(closedState); err != nil {
		fmt.Printf("Error applying closed-form diffusion: %v\n", err)
		return
	}
	printAmplitudeComparison("gate sequence", "closed form", gateState, closedState)

	// The same diffusion inside a full search, this time against the
	// algorithm package itself rather than a restatement of its formula.
	fmt.Printf("\nNow the whole search for |%0*b⟩, %d iterations of oracle then diffusion:\n",
		diffusionQubits, diffusionMarked, diffusionIterations)

	start, err := state.New(diffusionQubits)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	fastPath, err := algorithm.Grover(start, []int{diffusionMarked})
	if err != nil {
		fmt.Printf("Error running Grover: %v\n", err)
		return
	}

	gatePath, err := state.New(diffusionQubits)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := applyToEveryQubit(gatePath, gates.NewHadamard()); err != nil {
		fmt.Printf("Error applying Hadamard gate: %v\n", err)
		return
	}
	for iteration := 0; iteration < diffusionIterations; iteration++ {
		if err := applyGatePhaseOracle(gatePath, diffusionMarked); err != nil {
			fmt.Printf("Error applying gate oracle: %v\n", err)
			return
		}
		if err := applyGateDiffusion(gatePath); err != nil {
			fmt.Printf("Error applying gate diffusion: %v\n", err)
			return
		}
	}

	printAmplitudeComparison("algorithm.Grover", "gate circuit", fastPath, gatePath)
	fmt.Printf("Marked state probability: %.4f (fast path), %.4f (gate circuit)\n",
		fastPath.Probability(diffusionMarked), gatePath.Probability(diffusionMarked))
}

// RunAllAlgorithmDemos executes all algorithm demonstrations.
func RunAllAlgorithmDemos() {
	DeutschJozsaDemo()
	GroverDemo()
	GroverDiffusionGatesDemo()
}
