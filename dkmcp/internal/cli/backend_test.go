// backend_test.go contains unit tests for backend.go functions.
//
// backend_test.goはbackend.goの関数の単体テストを含みます。
package cli

import (
	"testing"
)

// TestParseExitCode tests the parseExitCode function.
// It verifies that exit codes are correctly extracted from MCP response text.
//
// TestParseExitCodeはparseExitCode関数をテストします。
// MCPレスポンステキストから終了コードが正しく抽出されることを確認します。
func TestParseExitCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "standard format exit code 0",
			input:    "Command: pwd\nExit Code: 0\n\nOutput:\n/app",
			expected: 0,
		},
		{
			name:     "standard format exit code 1",
			input:    "Command: cat /nonexistent\nExit Code: 1\n\nOutput:\ncat: /nonexistent: No such file or directory",
			expected: 1,
		},
		{
			name:     "exit code 127 (command not found)",
			input:    "Command: nonexistent\nExit Code: 127\n\nOutput:\nbash: nonexistent: command not found",
			expected: 127,
		},
		{
			name:     "exit code 255",
			input:    "Exit Code: 255\n\nOutput:\nerror",
			expected: 255,
		},
		{
			name:     "no exit code in response",
			input:    "Some output without exit code",
			expected: 0, // Default to success
		},
		{
			name:     "empty response",
			input:    "",
			expected: 0, // Default to success
		},
		{
			name:     "malformed exit code",
			input:    "Exit Code: abc\n\nOutput:\n",
			expected: 0, // Default to success on parse error
		},
		{
			name:     "exit code with extra whitespace",
			input:    "Exit Code:  42 \n\nOutput:\n",
			expected: 0, // Regex expects "Exit Code: N" format exactly
		},
		{
			name:     "multiple exit codes (takes first)",
			input:    "Exit Code: 1\nExit Code: 2\n",
			expected: 1,
		},
		{
			name:     "exit code in middle of text",
			input:    "Some text\nExit Code: 5\nMore text",
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseExitCode(tt.input)
			if got != tt.expected {
				t.Errorf("parseExitCode(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}
