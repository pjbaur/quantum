package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/examples"
	"github.com/pjbaur/quantum/quantum"
)

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintln(out, "Quantum Computing in Go")
	fmt.Fprintln(out, "======================")
	fmt.Fprintln(out, "Usage: quantum [options] <demo>")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Options:")
	flag.PrintDefaults()
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Available Demos:")
	fmt.Fprintln(out, "  all       - Run all demonstrations")
	fmt.Fprintln(out, "  hadamard  - Hadamard gate demonstrations")
	fmt.Fprintln(out, "  tgate     - T-gate demonstrations")
	fmt.Fprintln(out, "  bell      - Bell state demonstrations")
	fmt.Fprintln(out, "  algorithm - Algorithm demonstrations (Deutsch-Jozsa, Grover)")
	fmt.Fprintln(out, "  visual    - Visualization demonstrations")
	fmt.Fprintln(out, "  noise     - Noise channel demonstrations (density matrices)")
	fmt.Fprintln(out, "  gates     - Gate catalog and decomposition demonstrations")
	fmt.Fprintln(out, "  gate      - Look up one built-in gate by name (no name lists them all)")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  quantum hadamard       - Run Hadamard gate examples")
	fmt.Fprintln(out, "  quantum -demo bell     - Run Bell state examples")
	fmt.Fprintln(out, "  quantum visual         - Run visualization examples")
	fmt.Fprintln(out, "  quantum gate Hadamard  - Print the Hadamard gate's matrix")
	fmt.Fprintln(out, "  quantum gate           - List the registered gate names")
	fmt.Fprintln(out, "  quantum all            - Run all examples")
}

// runGate prints one gate from the built-in registry, or the registry's names
// when no gate was named.
func runGate(out io.Writer, name string) error {
	registry := gates.Builtin()

	if name == "" {
		fmt.Fprintln(out, "\n=== Registered Gates ===")
		for _, registered := range registry.Names() {
			fmt.Fprintf(out, "  %s\n", registered)
		}
		return nil
	}

	gate, ok := registry.Lookup(name)
	if !ok {
		return fmt.Errorf("unknown gate: %s (registered gates: %s)",
			name, strings.Join(registry.Names(), ", "))
	}

	qubits, err := quantum.GateQubitCount(gate)
	if err != nil {
		return fmt.Errorf("gate %s: %w", name, err)
	}

	fmt.Fprintf(out, "\n=== Gate: %s ===\n", gate.Name())
	fmt.Fprintf(out, "Qubits: %d\n", qubits)
	fmt.Fprintln(out, "Matrix:")
	for _, row := range gate.Matrix() {
		fmt.Fprint(out, "  [")
		for j, v := range row {
			if j > 0 {
				fmt.Fprint(out, "  ")
			}
			fmt.Fprintf(out, "%6.3f%+.3fi", real(v), imag(v))
		}
		fmt.Fprintln(out, "]")
	}
	return nil
}

// runDemos dispatches a demo word. gateName carries the optional argument of
// the "gate" command and is empty for every other demo.
func runDemos(demoType, gateName string) error {
	switch demoType {
	case "hadamard":
		examples.RunAllHadamardDemos()
	case "tgate":
		examples.RunAllTGateDemos()
	case "bell":
		examples.RunAllBellDemos()
	case "algorithm":
		examples.RunAllAlgorithmDemos()
	case "visual":
		examples.RunAllVisualizationDemos()
	case "noise":
		examples.RunAllNoiseDemos()
	case "gates":
		examples.RunAllGatesDemos()
	case "gate":
		return runGate(os.Stdout, gateName)
	case "all":
		fmt.Println("\n========================================================")
		fmt.Println("             QUANTUM COMPUTING IN GO")
		fmt.Println("              ALL DEMONSTRATIONS")
		fmt.Println("========================================================")

		examples.RunAllHadamardDemos()
		fmt.Println("\nPress Enter to continue to T-gate demonstrations...")
		fmt.Scanln()

		examples.RunAllTGateDemos()
		fmt.Println("\nPress Enter to continue to Bell state demonstrations...")
		fmt.Scanln()

		examples.RunAllBellDemos()
		fmt.Println("\nPress Enter to continue to algorithm demonstrations...")
		fmt.Scanln()

		examples.RunAllAlgorithmDemos()
		fmt.Println("\nPress Enter to continue to visualization demonstrations...")
		fmt.Scanln()

		examples.RunAllVisualizationDemos()
		fmt.Println("\nPress Enter to continue to noise channel demonstrations...")
		fmt.Scanln()

		examples.RunAllNoiseDemos()
		fmt.Println("\nPress Enter to continue to gate catalog demonstrations...")
		fmt.Scanln()

		examples.RunAllGatesDemos()

		fmt.Println("\n========================================================")
		fmt.Println("             ALL DEMONSTRATIONS COMPLETED")
		fmt.Println("========================================================")
	default:
		return fmt.Errorf("unknown demo: %s", demoType)
	}
	return nil
}

func main() {
	demoFlag := flag.String("demo", "", "Demo to run (hadamard, tgate, bell, algorithm, visual, noise, gates, gate, all)")
	flag.Usage = usage
	flag.Parse()

	demoType := *demoFlag
	args := flag.Args()

	if demoType != "" && len(args) > 0 {
		fmt.Fprintln(os.Stderr, "demo provided both as flag and positional argument")
		flag.Usage()
		os.Exit(2)
	}

	var extraArgs []string
	if demoType == "" {
		if len(args) == 0 {
			flag.Usage()
			os.Exit(2)
		}
		demoType = args[0]
		extraArgs = args[1:]
	}

	// "gate" is the only demo word that takes a trailing argument (the gate
	// name to look up); every other one stands alone.
	if len(extraArgs) > 1 || (len(extraArgs) == 1 && demoType != "gate") {
		fmt.Fprintln(os.Stderr, "too many arguments")
		flag.Usage()
		os.Exit(2)
	}

	var gateName string
	if len(extraArgs) == 1 {
		gateName = extraArgs[0]
	}

	// Display header
	fmt.Println("\n********************************************************")
	fmt.Println("*              QUANTUM COMPUTING IN GO                 *")
	fmt.Println("*            Quantum Circuit Simulator                 *")
	fmt.Println("********************************************************")

	// Run the selected demonstrations
	if err := runDemos(demoType, gateName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(2)
	}
}
