//go:build redtests

package parameterized_test

// Red tests for known open defects. Each test below documents a defect
// tracked as a numbered item in docs/enhancement-backlog-2026-08-27.md and
// is expected to fail until that item is closed. The redtests build
// constraint keeps them out of default builds so the regular suite stays
// green; run them with
//
//	go test -tags redtests ./parameterized -run '^TestRed'
//
// When an item is fixed, its test turns green: move it into the regular
// suite and drop it from here.

import (
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
)

// Backlog item 18: zero-value Template panics in AddParamGate.
//
// Template zero value × nil seen map in AddParamGate → panic instead of an
// error (or success).
func TestRedZeroValueTemplateAddParamGateDoesNotPanic(t *testing.T) {
	var tmpl parameterized.Template
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AddParamGate on a zero-value Template panicked: %v; want an error or success, never a panic (NewTemplate is not documented as required)", r)
		}
	}()
	// A zero-value template has no qubits, so any target is out of range;
	// the no-target call is the one that reaches the parameter bookkeeping.
	err := tmpl.AddParamGate("", parameterized.Ry)
	if err == nil {
		if names := tmpl.ParamNames(); len(names) != 1 || names[0] != "" {
			t.Fatalf("after an accepted AddParamGate(%q): ParamNames() = %q, want [%q]", "", names, "")
		}
		if counts := tmpl.ParamStepCounts(); counts[""] != 1 {
			t.Fatalf("after an accepted AddParamGate(%q): ParamStepCounts() = %v, want map[\"\":1]", "", counts)
		}
	}
}

// Backlog item 19: AddParamGate and AddGate accept an empty target list.
//
// AddParamGate("") with no targets × missing guard → accepted at build time
// although circuit.AddGate rejects a gate with no targets, so the step is
// declared and counted and only Bind fails.
func TestRedAddParamGateRejectsNoTargets(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	err := tmpl.AddParamGate("", parameterized.Ry)
	if err == nil {
		_, bindErr := tmpl.Bind(parameterized.Params{"": 0.1})
		t.Fatalf("AddParamGate(%q, Ry) with no targets accepted: ParamNames() = %q, ParamStepCounts() = %v; Bind then fails with %v; want the step rejected when added", "", tmpl.ParamNames(), tmpl.ParamStepCounts(), bindErr)
	}
	fixed := parameterized.NewTemplate(2)
	if err := fixed.AddGate(gates.NewHadamard()); err == nil {
		_, bindErr := fixed.Bind(parameterized.Params{})
		t.Fatalf("AddGate(H) with no targets accepted; Bind then fails with %v; want the step rejected when added", bindErr)
	}
}
