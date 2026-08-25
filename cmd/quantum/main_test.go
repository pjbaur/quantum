package main

import (
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
		{"all demo", "all"},
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
	expectedDemos := []string{"hadamard", "tgate", "bell", "algorithm", "visual", "noise", "all"}
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
