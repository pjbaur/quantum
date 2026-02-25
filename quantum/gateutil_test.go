package quantum

import (
	"testing"
)

// mockGate implements Gate for testing
type mockGate struct {
	name   string
	matrix [][]complex128
}

func (g *mockGate) Name() string           { return g.name }
func (g *mockGate) Matrix() [][]complex128 { return g.matrix }

// toffoliMatrix creates an 8x8 Toffoli gate matrix
func toffoliMatrix() [][]complex128 {
	m := make([][]complex128, 8)
	for i := range m {
		m[i] = make([]complex128, 8)
		if i < 6 {
			m[i][i] = 1
		}
	}
	m[6][7] = 1
	m[7][6] = 1
	return m
}

func TestGateQubitCount(t *testing.T) {
	tests := []struct {
		name        string
		gate        Gate
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name:      "1-qubit gate (2x2)",
			gate: &mockGate{
				name: "X",
				matrix: [][]complex128{
					{0, 1},
					{1, 0},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "2-qubit gate (4x4)",
			gate: &mockGate{
				name: "CNOT",
				matrix: [][]complex128{
					{1, 0, 0, 0},
					{0, 1, 0, 0},
					{0, 0, 0, 1},
					{0, 0, 1, 0},
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "3-qubit gate (8x8)",
			gate: &mockGate{
				name:   "Toffoli",
				matrix: toffoliMatrix(),
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "empty matrix",
			gate: &mockGate{
				name:   "Empty",
				matrix: [][]complex128{},
			},
			wantCount:   0,
			wantErr:     true,
			errContains: "empty matrix",
		},
		{
			name: "nil matrix",
			gate: &mockGate{
				name:   "Nil",
				matrix: nil,
			},
			wantCount:   0,
			wantErr:     true,
			errContains: "empty matrix",
		},
		{
			name: "non-square matrix (more rows)",
			gate: &mockGate{
				name: "NonSquare",
				matrix: [][]complex128{
					{1, 0},
					{0, 1},
					{1, 1},
				},
			},
			wantCount:   0,
			wantErr:     true,
			errContains: "must be square",
		},
		{
			name: "non-square matrix (jagged)",
			gate: &mockGate{
				name: "Jagged",
				matrix: [][]complex128{
					{1, 0, 0},
					{0, 1},
					{0, 0, 1},
				},
			},
			wantCount:   0,
			wantErr:     true,
			errContains: "must be square",
		},
		{
			name: "non-power-of-two size (3x3)",
			gate: &mockGate{
				name: "BadSize",
				matrix: [][]complex128{
					{1, 0, 0},
					{0, 1, 0},
					{0, 0, 1},
				},
			},
			wantCount:   0,
			wantErr:     true,
			errContains: "not a power of two",
		},
		{
			name: "non-power-of-two size (5x5)",
			gate: &mockGate{
				name: "BadSize5",
				matrix: [][]complex128{
					{1, 0, 0, 0, 0},
					{0, 1, 0, 0, 0},
					{0, 0, 1, 0, 0},
					{0, 0, 0, 1, 0},
					{0, 0, 0, 0, 1},
				},
			},
			wantCount:   0,
			wantErr:     true,
			errContains: "not a power of two",
		},
		{
			name: "non-power-of-two size (6x6)",
			gate: &mockGate{
				name: "BadSize6",
				matrix: [][]complex128{
					{1, 0, 0, 0, 0, 0},
					{0, 1, 0, 0, 0, 0},
					{0, 0, 1, 0, 0, 0},
					{0, 0, 0, 1, 0, 0},
					{0, 0, 0, 0, 1, 0},
					{0, 0, 0, 0, 0, 1},
				},
			},
			wantCount:   0,
			wantErr:     true,
			errContains: "not a power of two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCount, err := GateQubitCount(tt.gate)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GateQubitCount() expected error, got nil")
					return
				}
				if tt.errContains != "" {
					errMsg := err.Error()
					if !containsSubstring(errMsg, tt.errContains) {
						t.Errorf("GateQubitCount() error = %v, want error containing %q", err, tt.errContains)
					}
				}
				// Verify error type
				var invalidMatrixErr *InvalidGateMatrixError
				if !asInvalidGateMatrixError(err, &invalidMatrixErr) {
					t.Errorf("GateQubitCount() error type = %T, want *InvalidGateMatrixError", err)
				}
			} else {
				if err != nil {
					t.Errorf("GateQubitCount() unexpected error: %v", err)
					return
				}
				if gotCount != tt.wantCount {
					t.Errorf("GateQubitCount() = %v, want %v", gotCount, tt.wantCount)
				}
			}
		})
	}
}

func TestInvalidGateMatrixError(t *testing.T) {
	tests := []struct {
		name     string
		err      *InvalidGateMatrixError
		expected string
	}{
		{
			name: "error without size",
			err: &InvalidGateMatrixError{
				GateName: "TestGate",
				Reason:   "empty matrix",
			},
			expected: "gate TestGate has invalid matrix: empty matrix",
		},
		{
			name: "error with size",
			err: &InvalidGateMatrixError{
				GateName: "TestGate",
				Reason:   "matrix size is not a power of two",
				Size:     3,
			},
			expected: "gate TestGate has invalid matrix: matrix size is not a power of two (size 3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// asInvalidGateMatrixError is a helper to check error type without importing errors
func asInvalidGateMatrixError(err error, target **InvalidGateMatrixError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*InvalidGateMatrixError); ok {
		*target = e
		return true
	}
	return false
}
