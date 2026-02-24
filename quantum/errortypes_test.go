package quantum_test

import (
	"strings"
	"testing"

	"github.com/pjbaur/quantum/quantum"
)

func TestNormalizationErrorFields(t *testing.T) {
	err := &quantum.NormalizationError{
		AttemptedSum: 1.5,
		CurrentSum:   1.0,
	}

	if err.AttemptedSum != 1.5 {
		t.Errorf("expected AttemptedSum 1.5, got %v", err.AttemptedSum)
	}
	if err.CurrentSum != 1.0 {
		t.Errorf("expected CurrentSum 1.0, got %v", err.CurrentSum)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "1.5") {
		t.Errorf("error message should contain attempted sum 1.5, got: %s", errStr)
	}
	if !strings.Contains(errStr, "1.0") {
		t.Errorf("error message should contain current sum 1.0, got: %s", errStr)
	}
}

func TestInvalidQubitCountError(t *testing.T) {
	err := &quantum.InvalidQubitCountError{
		Requested: 0,
		Reason:    "must be positive",
	}

	if err.Requested != 0 {
		t.Errorf("expected Requested 0, got %v", err.Requested)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "0") {
		t.Errorf("error message should contain requested count 0, got: %s", errStr)
	}
	if !strings.Contains(errStr, "must be positive") {
		t.Errorf("error message should contain reason, got: %s", errStr)
	}
}

func TestSharedStateError(t *testing.T) {
	err := &quantum.SharedStateError{
		DuplicateIndices: []int{0, 2},
	}

	if len(err.DuplicateIndices) != 2 {
		t.Errorf("expected 2 duplicate indices, got %d", len(err.DuplicateIndices))
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "0") || !strings.Contains(errStr, "2") {
		t.Errorf("error message should contain duplicate indices, got: %s", errStr)
	}
	if !strings.Contains(errStr, "independent") {
		t.Errorf("error message should mention independent states, got: %s", errStr)
	}
}

func TestUnsupportedOperationError(t *testing.T) {
	err := &quantum.UnsupportedOperationError{
		Operation:   "3-qubit gate application",
		Backend:     "sparse",
		Alternative: "dense state backend",
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "sparse") {
		t.Errorf("error message should contain backend name, got: %s", errStr)
	}
	if !strings.Contains(errStr, "3-qubit") {
		t.Errorf("error message should contain operation, got: %s", errStr)
	}
	if !strings.Contains(errStr, "dense") {
		t.Errorf("error message should contain alternative, got: %s", errStr)
	}
}
