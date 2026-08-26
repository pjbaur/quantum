package examples

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// exactly what fn wrote. Capture is sequential (no concurrent writers), so a
// simple os.Pipe swap is race-free without further synchronization.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer failed: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured output failed: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("closing pipe reader failed: %v", err)
	}
	return buf.String()
}

func TestPrintRule(t *testing.T) {
	tests := []struct {
		name  string
		width int
		want  string
	}{
		{
			name:  "width_45",
			width: 45,
			want:  strings.Repeat("=", 45) + "\n",
		},
		{
			name:  "width_56",
			width: 56,
			want:  strings.Repeat("=", 56) + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStdout(t, func() { PrintRule(tt.width) })
			if got != tt.want {
				t.Errorf("PrintRule(%d) output = %q, want %q", tt.width, got, tt.want)
			}
		})
	}
}

func TestPrintBanner(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		titles []string
		want   string
	}{
		{
			name:  "single_title_width_45",
			width: 45,
			titles: []string{
				"    BELL STATE & ENTANGLEMENT DEMONSTRATIONS",
			},
			want: strings.Repeat("=", 45) + "\n" +
				"    BELL STATE & ENTANGLEMENT DEMONSTRATIONS\n" +
				strings.Repeat("=", 45) + "\n",
		},
		{
			name:  "two_titles_width_56",
			width: 56,
			titles: []string{
				"             QUANTUM COMPUTING IN GO",
				"              ALL DEMONSTRATIONS",
			},
			want: strings.Repeat("=", 56) + "\n" +
				"             QUANTUM COMPUTING IN GO\n" +
				"              ALL DEMONSTRATIONS\n" +
				strings.Repeat("=", 56) + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStdout(t, func() { PrintBanner(tt.width, tt.titles...) })
			if got != tt.want {
				t.Errorf("PrintBanner(%d, %v) output = %q, want %q", tt.width, tt.titles, got, tt.want)
			}
		})
	}
}
