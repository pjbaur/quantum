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
