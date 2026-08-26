package visualization_test

import (
	"strings"
	"testing"

	"github.com/pjbaur/quantum/state"
	"github.com/pjbaur/quantum/visualization"
)

func TestDefaultStateViewOptions(t *testing.T) {
	opts := visualization.DefaultStateViewOptions()

	if opts.MinProbability != 0 {
		t.Errorf("MinProbability = %v, want 0", opts.MinProbability)
	}
	if opts.Precision != 4 {
		t.Errorf("Precision = %v, want 4", opts.Precision)
	}
	if !opts.IncludeHeader {
		t.Errorf("IncludeHeader = false, want true")
	}
}

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
