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
	"errors"
	"testing"

	"github.com/pjbaur/quantum/parameterized"
)

// Backlog item 21: copying a Template by value after a declaration
// desyncs ParamNames from ParamStepCounts.
//
// Template copied by value after AddParamGate × shared seen map, private
// paramOrder slice → a declaration added to one copy is counted in the
// other copy's ParamStepCounts (seen is shared) but absent from its
// ParamNames (paramOrder is not), so Bind on the stale copy accepts
// values that omit the name it still counts.
func TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent(t *testing.T) {
	a := *parameterized.NewTemplate(2)
	if err := a.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	b := a
	if err := b.AddParamGate("phi", parameterized.Rx, 1); err != nil {
		t.Fatal(err)
	}
	if err := a.AddParamGate("phi", parameterized.Rx, 1); err != nil {
		t.Fatal(err)
	}
	names := a.ParamNames()
	counts := a.ParamStepCounts()
	for name := range counts {
		found := false
		for _, n := range names {
			if n == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("a.ParamStepCounts() = %v lists %q but a.ParamNames() = %q lacks it", counts, name, names)
		}
	}
	var missing *parameterized.MissingParameterError
	if _, err := a.Bind(parameterized.Params{"theta": 0.1}); !errors.As(err, &missing) {
		t.Fatalf("a.Bind without phi: err = %v, want MissingParameterError", err)
	}
}
