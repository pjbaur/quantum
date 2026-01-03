package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/pjbaur/quantum/internal/examples"
)

func showUsage() {
	fmt.Println("Quantum Computing in Go")
	fmt.Println("======================")
	fmt.Println("Usage: go run main.go [demo]")
	fmt.Println("")
	fmt.Println("Available Demos:")
	fmt.Println("  all       - Run all demonstrations")
	fmt.Println("  hadamard  - Hadamard gate demonstrations")
	fmt.Println("  tgate     - T-gate demonstrations")
	fmt.Println("  bell      - Bell state demonstrations")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run main.go hadamard  - Run Hadamard gate examples")
	fmt.Println("  go run main.go all       - Run all examples sequentially")
}

func runDemos(demoType string) {
	switch demoType {
	case "hadamard":
		examples.RunAllHadamardDemos()
	case "tgate":
		examples.RunAllTGateDemos()
	case "bell":
		examples.RunAllBellDemos()
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

		fmt.Println("\n========================================================")
		fmt.Println("             ALL DEMONSTRATIONS COMPLETED")
		fmt.Println("========================================================")
	default:
		fmt.Printf("Unknown example: %s\n", demoType)
		showUsage()
	}
}

func main() {
	// Seed the random number generator with current time
	rand.Seed(time.Now().UnixNano())

	// Display header
	fmt.Println("\n********************************************************")
	fmt.Println("*              QUANTUM COMPUTING IN GO                 *")
	fmt.Println("*            Quantum Circuit Simulator                 *")
	fmt.Println("********************************************************")

	// Process command line arguments
	args := os.Args
	if len(args) < 2 {
		showUsage()
		return
	}

	// Run the specified demonstration
	demoType := args[1]

	// Check for optional parameters
	var param int = 0
	if len(args) >= 3 {
		var err error
		param, err = strconv.Atoi(args[2])
		if err != nil {
			fmt.Printf("Invalid parameter: %s\n", args[2])
			return
		}
	}

	// Placeholder for using param if needed
	_ = param

	// Run the selected demonstrations
	runDemos(demoType)
}
