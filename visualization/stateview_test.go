package visualization_test

import (
	"strings"
	"testing"

	"github.com/pjbaur/quantum/state"
	"github.com/pjbaur/quantum/visualization"
)

func TestFormatStateViewIncludesHeaderAndEntry(t *testing.T) {
	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	opts := visualization.StateViewOptions{
		MinProbability: 0.1,
		Precision:      2,
		IncludeHeader:  true,
	}

	view := visualization.FormatStateView(s, opts)
	if !strings.Contains(view, "idx") || !strings.Contains(view, "basis") {
		t.Fatalf("expected header in state view, got: %q", view)
	}
	if !strings.Contains(view, "|0>") {
		t.Fatalf("expected basis |0> in state view, got: %q", view)
	}
	if !strings.Contains(view, "1.00+0.00i") {
		t.Fatalf("expected formatted amplitude in state view, got: %q", view)
	}
}
