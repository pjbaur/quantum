package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/pjbaur/quantum/internal/examples"
)

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintln(out, "Quantum Computing in Go")
	fmt.Fprintln(out, "======================")
	fmt.Fprintln(out, "Usage: quantum [options] <demo> [param]")
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
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  go run ./cmd/quantum hadamard       - Run Hadamard gate examples")
	fmt.Fprintln(out, "  go run ./cmd/quantum -demo bell     - Run Bell state examples")
	fmt.Fprintln(out, "  go run ./cmd/quantum visual         - Run visualization examples")
	fmt.Fprintln(out, "  go run ./cmd/quantum all 3          - Run all examples with param")
}

func runDemos(demoType string) error {
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

		fmt.Println("\n========================================================")
		fmt.Println("             ALL DEMONSTRATIONS COMPLETED")
		fmt.Println("========================================================")
	default:
		return fmt.Errorf("unknown demo: %s", demoType)
	}
	return nil
}

func main() {
	demoFlag := flag.String("demo", "", "Demo to run (hadamard, tgate, bell, algorithm, all)")
	paramFlag := flag.Int("param", 0, "Optional numeric parameter for demos")
	flag.Usage = usage
	flag.Parse()

	demoType := *demoFlag
	param := *paramFlag
	args := flag.Args()

	if demoType != "" && len(args) > 0 {
		fmt.Fprintln(os.Stderr, "demo provided both as flag and positional argument")
		flag.Usage()
		os.Exit(2)
	}

	if demoType == "" {
		if len(args) == 0 {
			flag.Usage()
			os.Exit(2)
		}
		demoType = args[0]
		args = args[1:]
	}

	if len(args) > 0 {
		if *paramFlag != 0 {
			fmt.Fprintln(os.Stderr, "param provided both as flag and positional argument")
			flag.Usage()
			os.Exit(2)
		}
		parsedParam, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid parameter: %s\n", args[0])
			os.Exit(2)
		}
		param = parsedParam
		args = args[1:]
	}

	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "too many arguments")
		flag.Usage()
		os.Exit(2)
	}

	// Seed the random number generator with current time
	rand.Seed(time.Now().UnixNano())

	// Display header
	fmt.Println("\n********************************************************")
	fmt.Println("*              QUANTUM COMPUTING IN GO                 *")
	fmt.Println("*            Quantum Circuit Simulator                 *")
	fmt.Println("********************************************************")

	// Placeholder for using param if needed
	_ = param

	// Run the selected demonstrations
	if err := runDemos(demoType); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(2)
	}
}
