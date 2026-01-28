// Package security tests verify the output masking functionality.
// These tests ensure that sensitive data is properly masked in tool output.
//
// securityパッケージのテストは出力マスキング機能を検証します。
// これらのテストはツール出力内の機密データが適切にマスクされることを確認します。
package security

import (
	"strings"
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// TestNewOutputMasker verifies that NewOutputMasker creates a properly initialized masker.
// TestNewOutputMaskerはNewOutputMaskerが適切に初期化されたマスカーを作成することを検証します。
func TestNewOutputMasker(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "[MASKED]",
		Patterns: []string{
			`(?i)password\s*=\s*\S+`,
		},
		ApplyTo: config.OutputMaskingTargets{
			Logs:    true,
			Exec:    true,
			Inspect: true,
		},
	}

	masker, err := NewOutputMasker(cfg)
	if err != nil {
		t.Fatalf("NewOutputMasker failed: %v", err)
	}

	if masker == nil {
		t.Fatal("NewOutputMasker returned nil")
	}

	if !masker.IsEnabled() {
		t.Error("Masker should be enabled")
	}

	if masker.PatternCount() != 1 {
		t.Errorf("PatternCount = %d, want 1", masker.PatternCount())
	}
}

// TestNewOutputMasker_NilConfig verifies that nil config creates a disabled masker.
// TestNewOutputMasker_NilConfigはnil設定が無効なマスカーを作成することを検証します。
func TestNewOutputMasker_NilConfig(t *testing.T) {
	masker, err := NewOutputMasker(nil)
	if err != nil {
		t.Fatalf("NewOutputMasker(nil) failed: %v", err)
	}

	if masker == nil {
		t.Fatal("NewOutputMasker(nil) returned nil")
	}

	if masker.IsEnabled() {
		t.Error("Masker with nil config should be disabled")
	}
}

// TestMaskOutput_Passwords verifies password patterns are masked correctly.
// TestMaskOutput_Passwordsはパスワードパターンが正しくマスクされることを検証します。
func TestMaskOutput_Passwords(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "[MASKED]",
		Patterns: []string{
			`(?i)(password|passwd|pwd)\s*[=:]\s*["']?[^\s"'\n]+["']?`,
		},
		ApplyTo: config.OutputMaskingTargets{
			Logs: true,
			Exec: true,
		},
	}

	masker, _ := NewOutputMasker(cfg)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "password=value",
			input: "Connecting with password=secret123",
			want:  "Connecting with [MASKED]",
		},
		{
			name:  "PASSWORD=value uppercase",
			input: "PASSWORD=MySecretPass",
			want:  "[MASKED]",
		},
		{
			name:  "passwd: value with colon",
			input: "passwd: admin123",
			want:  "[MASKED]",
		},
		{
			name:  "pwd='quoted'",
			input: "Config: pwd='secret'",
			want:  "Config: [MASKED]",
		},
		{
			name:  "no password",
			input: "Normal log message",
			want:  "Normal log message",
		},
		{
			name:  "multiple passwords",
			input: "password=abc passwd=def pwd=ghi",
			want:  "[MASKED] [MASKED] [MASKED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := masker.MaskOutput(tt.input)
			if got != tt.want {
				t.Errorf("MaskOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMaskOutput_APIKeys verifies API key patterns are masked correctly.
// TestMaskOutput_APIKeysはAPIキーパターンが正しくマスクされることを検証します。
func TestMaskOutput_APIKeys(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "[MASKED]",
		Patterns: []string{
			`sk-[a-zA-Z0-9]{20,}`,
			`(?i)bearer\s+[a-zA-Z0-9._-]+`,
		},
		ApplyTo: config.OutputMaskingTargets{
			Logs: true,
		},
	}

	masker, _ := NewOutputMasker(cfg)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "OpenAI API key",
			input: "Using API key: sk-abcdefghijklmnopqrstuvwxyz123456",
			want:  "Using API key: [MASKED]",
		},
		{
			name:  "Bearer token",
			input: "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xxxxx",
			want:  "Authorization: [MASKED]",
		},
		{
			name:  "no API key",
			input: "Normal log message without secrets",
			want:  "Normal log message without secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := masker.MaskOutput(tt.input)
			if got != tt.want {
				t.Errorf("MaskOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMaskOutput_DatabaseURLs verifies database connection strings are masked.
// TestMaskOutput_DatabaseURLsはデータベース接続文字列がマスクされることを検証します。
func TestMaskOutput_DatabaseURLs(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "[MASKED]",
		Patterns: []string{
			`(?i)(postgres|mysql|mongodb|redis)://[^:]+:[^@]+@`,
		},
		ApplyTo: config.OutputMaskingTargets{
			Logs: true,
		},
	}

	masker, _ := NewOutputMasker(cfg)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "postgres URL",
			input: "DATABASE_URL=postgres://user:password123@db.example.com:5432/mydb",
			want:  "DATABASE_URL=[MASKED]db.example.com:5432/mydb",
		},
		{
			name:  "mysql URL",
			input: "Connecting to mysql://admin:secret@localhost:3306/app",
			want:  "Connecting to [MASKED]localhost:3306/app",
		},
		{
			name:  "mongodb URL",
			input: "mongodb://root:rootpass@mongo:27017/testdb",
			want:  "[MASKED]mongo:27017/testdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := masker.MaskOutput(tt.input)
			if got != tt.want {
				t.Errorf("MaskOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMaskOutput_Disabled verifies no masking when disabled.
// TestMaskOutput_Disabledは無効時にマスキングが行われないことを検証します。
func TestMaskOutput_Disabled(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     false,
		Replacement: "[MASKED]",
		Patterns: []string{
			`(?i)password\s*=\s*\S+`,
		},
	}

	masker, _ := NewOutputMasker(cfg)

	input := "password=secret123"
	got := masker.MaskOutput(input)

	if got != input {
		t.Errorf("MaskOutput when disabled = %q, want %q (unchanged)", got, input)
	}
}

// TestMaskLogs_ApplyTo verifies masking is applied based on ApplyTo settings.
// TestMaskLogs_ApplyToはApplyTo設定に基づいてマスキングが適用されることを検証します。
func TestMaskLogs_ApplyTo(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "[MASKED]",
		Patterns: []string{
			`(?i)password\s*=\s*\S+`,
		},
		ApplyTo: config.OutputMaskingTargets{
			Logs:    true,
			Exec:    false,
			Inspect: true,
		},
	}

	masker, _ := NewOutputMasker(cfg)
	input := "password=secret123"

	// Logs should be masked
	// ログはマスクされるべき
	if got := masker.MaskLogs(input); got == input {
		t.Error("MaskLogs should mask when apply_to.logs is true")
	}

	// Exec should NOT be masked
	// Execはマスクされるべきではない
	if got := masker.MaskExec(input); got != input {
		t.Error("MaskExec should not mask when apply_to.exec is false")
	}

	// Inspect should be masked
	// Inspectはマスクされるべき
	if got := masker.MaskInspect(input); got == input {
		t.Error("MaskInspect should mask when apply_to.inspect is true")
	}
}

// TestAddPattern verifies runtime pattern addition.
// TestAddPatternは実行時のパターン追加を検証します。
func TestAddPattern(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "[MASKED]",
		Patterns:    []string{},
		ApplyTo: config.OutputMaskingTargets{
			Logs: true,
		},
	}

	masker, _ := NewOutputMasker(cfg)

	// Initially no patterns
	// 初期状態ではパターンなし
	if masker.PatternCount() != 0 {
		t.Errorf("Initial PatternCount = %d, want 0", masker.PatternCount())
	}

	// Add a valid pattern
	// 有効なパターンを追加
	err := masker.AddPattern(`(?i)secret\s*=\s*\S+`)
	if err != nil {
		t.Errorf("AddPattern failed: %v", err)
	}

	if masker.PatternCount() != 1 {
		t.Errorf("PatternCount after add = %d, want 1", masker.PatternCount())
	}

	// Verify the pattern works
	// パターンが動作することを確認
	input := "secret=mysecret"
	got := masker.MaskLogs(input)
	if got != "[MASKED]" {
		t.Errorf("MaskLogs after AddPattern = %q, want [MASKED]", got)
	}

	// Add invalid pattern should return error
	// 無効なパターンの追加はエラーを返すべき
	err = masker.AddPattern(`[invalid`)
	if err == nil {
		t.Error("AddPattern with invalid regex should return error")
	}
}

// TestDefaultReplacement verifies default replacement string is used when empty.
// TestDefaultReplacementは空の場合にデフォルト置換文字列が使用されることを検証します。
func TestDefaultReplacement(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "", // Empty replacement
		Patterns: []string{
			`(?i)password\s*=\s*\S+`,
		},
		ApplyTo: config.OutputMaskingTargets{
			Logs: true,
		},
	}

	masker, _ := NewOutputMasker(cfg)

	input := "password=secret"
	got := masker.MaskLogs(input)

	// Should use default "[MASKED]"
	// デフォルトの"[MASKED]"が使用されるべき
	if got != "[MASKED]" {
		t.Errorf("MaskLogs with empty replacement = %q, want [MASKED]", got)
	}
}

// TestMultilineOutput verifies masking works across multiple lines.
// TestMultilineOutputはマスキングが複数行で動作することを検証します。
func TestMultilineOutput(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "[MASKED]",
		Patterns: []string{
			`(?i)password\s*[=:]\s*["']?[^\s"'\n]+["']?`,
			`sk-[a-zA-Z0-9]{20,}`,
		},
		ApplyTo: config.OutputMaskingTargets{
			Logs: true,
		},
	}

	masker, _ := NewOutputMasker(cfg)

	input := `Starting application...
Config loaded:
  password=secret123
  api_key=sk-abcdefghijklmnopqrstuvwxyz
  host=localhost
Server started on port 8080`

	got := masker.MaskLogs(input)

	// Verify passwords and API keys are masked
	// パスワードとAPIキーがマスクされていることを確認
	if strings.Contains(got, "secret123") {
		t.Error("Password should be masked in output")
	}
	if strings.Contains(got, "sk-abcdefghijklmnopqrstuvwxyz") {
		t.Error("API key should be masked in output")
	}

	// Verify non-sensitive data is preserved
	// 機密でないデータが保持されていることを確認
	if !strings.Contains(got, "Starting application") {
		t.Error("Non-sensitive data should be preserved")
	}
	if !strings.Contains(got, "localhost") {
		t.Error("Host should be preserved")
	}
}

// TestNewOutputMasker_InvalidPatternIgnored verifies that invalid regex patterns are silently skipped.
// This ensures the masker still works even if one pattern is invalid.
//
// TestNewOutputMasker_InvalidPatternIgnoredは無効な正規表現パターンが無視されることを検証します。
// 1つのパターンが無効でもマスカーが動作することを確認します。
func TestNewOutputMasker_InvalidPatternIgnored(t *testing.T) {
	cfg := &config.OutputMaskingConfig{
		Enabled:     true,
		Replacement: "[MASKED]",
		Patterns: []string{
			`(?i)password\s*=\s*\S+`, // valid
			`[invalid`,               // invalid regex - unclosed bracket
			`sk-[a-zA-Z0-9]{20,}`,    // valid
		},
		ApplyTo: config.OutputMaskingTargets{
			Logs: true,
		},
	}

	masker, err := NewOutputMasker(cfg)
	if err != nil {
		t.Fatalf("NewOutputMasker failed: %v", err)
	}

	// Should have 2 valid patterns (invalid one is skipped)
	// 2つの有効なパターンがあるはず（無効なものはスキップ）
	if masker.PatternCount() != 2 {
		t.Errorf("PatternCount = %d, want 2 (invalid pattern should be skipped)", masker.PatternCount())
	}

	// Verify both valid patterns work
	// 両方の有効なパターンが動作することを確認
	input1 := "password=secret"
	if got := masker.MaskLogs(input1); got != "[MASKED]" {
		t.Errorf("First valid pattern not working: got %q", got)
	}

	input2 := "key: sk-abcdefghijklmnopqrstuvwxyz"
	if !strings.Contains(masker.MaskLogs(input2), "[MASKED]") {
		t.Error("Second valid pattern not working")
	}
}
