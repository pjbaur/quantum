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

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)

// Backlog item 22: declared targets alias the caller's variadic slice.
//
// AddParamGate and AddGate both append `step{... targets: targets}`
// directly, storing the caller's variadic slice by reference rather than
// a copy; mutating the slice after a successful declaration changes what
// Bind later builds, although the call already returned success and the
// declaration looked complete. Pre-dates item 19: the append line is
// unchanged by it, and the same defect is reachable through AddGate too
// (both share the storage, only AddParamGate is exercised here).
func TestRedDeclaredTargetsAreNotAliasedToCallerSlice(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	targets := []int{0}
	if err := tmpl.AddParamGate("theta", parameterized.Ry, targets...); err != nil {
		t.Fatal(err)
	}
	targets[0] = 1
	bound, err := tmpl.Bind(parameterized.Params{"theta": 0.2})
	if err != nil {
		t.Fatal(err)
	}
	// As declared: Ry(0.2) on qubit 0, nothing on qubit 1.
	manual, _ := circuit.New(2)
	_ = manual.AddGate(gates.NewRy(0.2), 0)
	sa, _ := state.New(2)
	sb, _ := state.New(2)
	if err := bound.Execute(sa); err != nil {
		t.Fatal(err)
	}
	if err := manual.Execute(sb); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if sa.Amplitude(i) != sb.Amplitude(i) {
			t.Fatalf("amplitude %d after mutating the caller's slice: got %v, want %v (Ry on qubit 0 as declared)", i, sa.Amplitude(i), sb.Amplitude(i))
		}
	}
}
