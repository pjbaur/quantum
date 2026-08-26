package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestCLIHelpOutputDemos verifies that the help output includes all implemented demos.
// This test prevents drift between implementation and documentation.
func TestCLIHelpOutputDemos(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"hadamard demo", "hadamard"},
		{"tgate demo", "tgate"},
		{"bell demo", "bell"},
		{"algorithm demo", "algorithm"},
		{"visual demo", "visual"},
		{"noise demo", "noise"},
		{"gates demo", "gates"},
		{"all demo", "all"},
		{"gate command", "Look up one built-in gate by name"},
	}

	// Get help output
	cmd := exec.Command("go", "run", "./cmd/quantum", "-h")
	output, err := cmd.CombinedOutput()
	if err == nil {
		// -h may exit with 0, which is fine
	}

	helpText := string(output)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(helpText, tt.expected) {
				t.Errorf("help output missing %q demo.\nHelp output:\n%s", tt.expected, helpText)
			}
		})
	}
}

// TestCLIHelpOutputFlagDescription verifies the -demo flag description includes all demos.
func TestCLIHelpOutputFlagDescription(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum", "-h")
	output, err := cmd.CombinedOutput()
	// -h may not return an error, that's fine
	_ = err

	helpText := string(output)

	// The -demo flag should list all available demos
	expectedDemos := []string{"hadamard", "tgate", "bell", "algorithm", "visual", "noise", "gates", "all"}
	for _, demo := range expectedDemos {
		if !strings.Contains(helpText, demo) {
			t.Errorf("-demo flag description missing %q in help output.\nHelp output:\n%s", demo, helpText)
		}
	}
}

// TestCLIUnknownDemo verifies that unknown demos return an error.
func TestCLIUnknownDemo(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum", "unknown-demo-that-does-not-exist")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected error for unknown demo, but command succeeded.\nOutput:\n%s", string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "unknown demo") {
		t.Errorf("expected 'unknown demo' error message, got:\n%s", outputStr)
	}
}

// TestCLIMissingArgument verifies that running without arguments shows usage.
func TestCLIMissingArgument(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected error when no demo provided, but command succeeded.\nOutput:\n%s", string(output))
	}

	// Should exit with non-zero code
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Errorf("expected non-zero exit code, got %d", exitErr.ExitCode())
		}
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage:") {
		t.Errorf("expected usage output, got:\n%s", outputStr)
	}
}

// TestCLIDemoFlagAndPositionalConflict verifies error when both flag and positional are used.
func TestCLIDemoFlagAndPositionalConflict(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum", "-demo", "hadamard", "bell")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected error when demo provided both as flag and positional, but command succeeded.\nOutput:\n%s", string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "both as flag and positional") {
		t.Errorf("expected 'both as flag and positional' error message, got:\n%s", outputStr)
	}
}

// TestCLITooManyArguments verifies error for too many positional arguments.
func TestCLITooManyArguments(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum", "hadamard", "extra-arg")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected error for too many arguments, but command succeeded.\nOutput:\n%s", string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "too many arguments") {
		t.Errorf("expected 'too many arguments' error message, got:\n%s", outputStr)
	}
}

// TestCLIVisualDemoRuns verifies the visual demo executes successfully.
func TestCLIVisualDemoRuns(t *testing.T) {
	// Skip in short mode since this runs the actual demo
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cmd := exec.Command("go", "run", "./cmd/quantum", "visual")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		t.Fatalf("visual demo failed to run.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	// Verify some expected output content
	output := stdout.String()
	if !strings.Contains(output, "Bloch Vector") {
		t.Errorf("visual demo output missing 'Bloch Vector', got:\n%s", output)
	}
}

// TestCLINoiseDemoRuns verifies the noise demo executes successfully.
func TestCLINoiseDemoRuns(t *testing.T) {
	// Skip in short mode since this runs the actual demo
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cmd := exec.Command("go", "run", "./cmd/quantum", "noise")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		t.Fatalf("noise demo failed to run.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	// Verify some expected output content
	output := stdout.String()
	for _, want := range []string{"Dephasing", "Amplitude Damping", "Depolarizing", "purity"} {
		if !strings.Contains(output, want) {
			t.Errorf("noise demo output missing %q, got:\n%s", want, output)
		}
	}
}

// TestCLIGatesDemoRuns verifies the gates demo executes successfully.
func TestCLIGatesDemoRuns(t *testing.T) {
	// Skip in short mode since this runs the actual demo
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cmd := exec.Command("go", "run", "./cmd/quantum", "gates")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		t.Fatalf("gates demo failed to run.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	// Verify some expected output content
	output := stdout.String()
	for _, want := range []string{"Hadamard", "CNOT", "SWAP", "decomposition", "match"} {
		if !strings.Contains(output, want) {
			t.Errorf("gates demo output missing %q, got:\n%s", want, output)
		}
	}
}

// TestCLIAlgorithmDemoRuns verifies the algorithm demo executes successfully,
// and that its gate-built Grover diffusion still agrees with the algorithm
// package's fast path. The demo prints a verdict line per comparison and
// swallows its own errors, so both the verdict and the absence of an error
// report are asserted here rather than left as prose nobody reads.
func TestCLIAlgorithmDemoRuns(t *testing.T) {
	// Skip in short mode since this runs the actual demo
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cmd := exec.Command("go", "run", "./cmd/quantum", "algorithm")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		t.Fatalf("algorithm demo failed to run.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	output := stdout.String()
	want := []string{
		"Deutsch-Jozsa",
		"Grover Demonstration",
		"Grover Diffusion as a Gate Sequence",
		"algorithm.Grover",
		"Amplitudes match",
	}
	for _, expected := range want {
		if !strings.Contains(output, expected) {
			t.Errorf("algorithm demo output missing %q, got:\n%s", expected, output)
		}
	}
	if strings.Contains(output, "DO NOT match") {
		t.Errorf("gate circuit disagreed with the algorithm's fast path, got:\n%s", output)
	}
	if strings.Contains(output, "Error") {
		t.Errorf("algorithm demo reported an error, got:\n%s", output)
	}
}

// TestCLIGateLookup verifies that "quantum gate <name>" reports the gate found
// in the built-in registry, for both a one-qubit and a two-qubit gate.
func TestCLIGateLookup(t *testing.T) {
	tests := []struct {
		name     string
		gate     string
		expected []string
	}{
		{
			name:     "one-qubit gate",
			gate:     "Hadamard",
			expected: []string{"Gate: Hadamard", "Qubits: 1", "0.707+0.000i", "-0.707+0.000i"},
		},
		{
			name:     "two-qubit gate",
			gate:     "CNOT",
			expected: []string{"Gate: CNOT", "Qubits: 2", "1.000+0.000i"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", "./cmd/quantum", "gate", tt.gate)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				t.Fatalf("gate lookup failed.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
			}

			output := stdout.String()
			for _, want := range tt.expected {
				if !strings.Contains(output, want) {
					t.Errorf("gate output missing %q, got:\n%s", want, output)
				}
			}
		})
	}
}

// TestCLIGateUnknownName verifies that an unregistered gate name fails and that
// the error lists the names the registry does hold.
func TestCLIGateUnknownName(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum", "gate", "NotAGate")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected error for unknown gate, but command succeeded.\nOutput:\n%s", string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "unknown gate: NotAGate") {
		t.Errorf("expected 'unknown gate' error message, got:\n%s", outputStr)
	}
	for _, want := range []string{"Hadamard", "CNOT", "SWAP"} {
		if !strings.Contains(outputStr, want) {
			t.Errorf("error message missing registered gate %q, got:\n%s", want, outputStr)
		}
	}
}

// TestCLIGateListsNames verifies that "quantum gate" without a name lists every
// gate in the built-in registry.
func TestCLIGateListsNames(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum", "gate")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("gate listing failed.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{"CNOT", "Hadamard", "PauliX", "PauliY", "PauliZ", "S", "SWAP", "T", "Toffoli"} {
		if !strings.Contains(output, want) {
			t.Errorf("gate listing missing %q, got:\n%s", want, output)
		}
	}
}

// TestCLIGateTooManyArguments verifies that the gate command still accepts only
// a single gate name.
func TestCLIGateTooManyArguments(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum", "gate", "Hadamard", "PauliX")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected error for too many arguments, but command succeeded.\nOutput:\n%s", string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "too many arguments") {
		t.Errorf("expected 'too many arguments' error message, got:\n%s", outputStr)
	}
}

// TestCLIAllDemoPipedStdinDoesNotHang verifies that "quantum all" with a
// piped (non-terminal) stdin runs straight through to completion instead of
// blocking on the "Press Enter to continue..." pauses. This is the ADR-0006
// regression test for the piped-hang defect: previously, every unguarded
// fmt.Scanln() in the "all" case would wait forever for input that a pipe
// never provides.
func TestCLIAllDemoPipedStdinDoesNotHang(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cmd := exec.Command("go", "run", "./cmd/quantum", "all")
	cmd.Stdin = strings.NewReader("") // a pipe, not a terminal
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("quantum all with piped stdin failed.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"ALL DEMONSTRATIONS",
		"Hadamard Gate Demonstration",
		"T Gate vs. Hadamard Comparison",
		"Bell State Creation Demonstration",
		"Deutsch-Jozsa Demonstration",
		"Bloch Vector",
		"Dephasing Channel Demonstration",
		"Built-in Gate Catalog",
		"ALL DEMONSTRATIONS COMPLETED",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("quantum all output missing section banner %q, got:\n%s", want, output)
		}
	}
}

// TestCLIAllDemoNoPauseFlag verifies that -no-pause suppresses the
// interactive "Press Enter" prompts even though it doesn't affect whether
// stdin is a terminal.
func TestCLIAllDemoNoPauseFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cmd := exec.Command("go", "run", "./cmd/quantum", "-no-pause", "all")
	cmd.Stdin = strings.NewReader("")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("quantum -no-pause all failed.\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	output := stdout.String()
	if strings.Contains(output, "Press Enter") {
		t.Errorf("-no-pause should suppress \"Press Enter\" prompts, got:\n%s", output)
	}
	if !strings.Contains(output, "ALL DEMONSTRATIONS COMPLETED") {
		t.Errorf("quantum -no-pause all did not complete, got:\n%s", output)
	}
}

// TestCLINoPauseFlagDocumented verifies that -no-pause is documented in the
// help output, alongside the other flags.
func TestCLINoPauseFlagDocumented(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/quantum", "-h")
	output, err := cmd.CombinedOutput()
	_ = err // -h may not return an error, that's fine

	if !strings.Contains(string(output), "no-pause") {
		t.Errorf("help output missing -no-pause flag, got:\n%s", string(output))
	}
}

// TestPausePrompterTypedContentDoesNotDisablePauses verifies that typing
// something other than a bare Enter at a pause (e.g. "y" + Enter, a natural
// response to a "Press Enter to continue" prompt) does not silently disable
// the remaining pauses. This is the byte-identical-TTY-behavior regression
// this test guards against: the original fmt.Scanln() call discarded its
// result entirely, so stray input at one pause never affected later ones.
func TestPausePrompterTypedContentDoesNotDisablePauses(t *testing.T) {
	var out bytes.Buffer
	p := &pausePrompter{
		w: &out,
		r: bufio.NewReader(strings.NewReader("y\nanything else\n")),
	}

	p.pause("section A")
	if p.disabled {
		t.Fatalf("pause disabled itself after typed content \"y\\n\", want still enabled")
	}

	p.pause("section B")
	if p.disabled {
		t.Fatalf("pause disabled itself after typed content \"anything else\\n\", want still enabled")
	}

	want := "\nPress Enter to continue to section A...\n" + "\nPress Enter to continue to section B...\n"
	if got := out.String(); got != want {
		t.Errorf("prompt output = %q, want %q", got, want)
	}
}

// TestPausePrompterEOFDisablesRemainingPauses verifies that a genuine read
// failure (stdin ending before a newline arrives) disables all later pauses
// instead of repeating the prompt or aborting the demos: the section after
// the failed read still runs, just without any further pausing.
func TestPausePrompterEOFDisablesRemainingPauses(t *testing.T) {
	var out bytes.Buffer
	p := &pausePrompter{
		w: &out,
		r: bufio.NewReader(strings.NewReader("")), // EOF with no data at all
	}

	p.pause("first section")
	if !p.disabled {
		t.Fatalf("pause did not disable itself after a read failure, want disabled")
	}

	p.pause("second section")

	want := "\nPress Enter to continue to first section...\n"
	if got := out.String(); got != want {
		t.Errorf("prompt output = %q, want %q (second pause should have printed nothing)", got, want)
	}
}

// TestPausePrompterDisabledIsNoOp verifies that an already-disabled prompter
// neither prints nor reads.
func TestPausePrompterDisabledIsNoOp(t *testing.T) {
	var out bytes.Buffer
	p := &pausePrompter{w: &out, disabled: true}

	p.pause("some section")

	if out.Len() != 0 {
		t.Errorf("disabled pause() wrote output: %q", out.String())
	}
}

// TestMain runs setup/teardown for CLI tests.
func TestMain(m *testing.M) {
	// Change to repo root so "go run ./cmd/quantum" works
	// The tests run from the package directory, so we need to go up
	cwd, _ := os.Getwd()
	// If we're in cmd/quantum, go to repo root
	if strings.HasSuffix(cwd, "/cmd/quantum") || cwd == "cmd/quantum" {
		os.Chdir("../..")
	}

	os.Exit(m.Run())
}
