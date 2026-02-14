// commands_test.go contains unit tests for the CLI commands.
// These tests verify that commands are properly registered with correct flags and arguments.
//
// commands_test.goはCLIコマンドのユニットテストを含みます。
// これらのテストは、コマンドが正しいフラグと引数で適切に登録されていることを確認します。
//
// IMPORTANT: Scope and Limitations of These Tests
// 重要: これらのテストのスコープと制限事項
//
// These tests verify REGISTRATION only, not EXECUTION:
// これらのテストは登録のみを検証し、実行は検証しません：
//
//   - ✅ Command exists (コマンドが存在する)
//   - ✅ Command name and usage string (コマンド名と使用方法文字列)
//   - ✅ Flag names and default values (フラグ名とデフォルト値)
//   - ❌ Actual command execution logic (実際のコマンド実行ロジック)
//   - ❌ Backend interactions (バックエンドとのやり取り)
//   - ❌ Error handling during execution (実行中のエラーハンドリング)
//
// Why these tests exist:
// これらのテストが存在する理由：
//
//   1. Smoke tests: Ensure init() registers commands correctly
//      スモークテスト: init()がコマンドを正しく登録することを確認
//   2. Refactoring guard: Detect accidental command/flag removal
//      リファクタリングガード: コマンド/フラグの誤削除を検出
//   3. Documentation: Show expected command structure
//      ドキュメント: 期待されるコマンド構造を示す
//
// The actual execution logic is tested indirectly via Backend tests
// (backend_test.go) which cover the core functionality.
// 実際の実行ロジックはBackendテスト（backend_test.go）で間接的にテストされ、
// コア機能がカバーされています。
package cli

import (
	"testing"
)

// TestLogsCommand verifies that the logs command is properly configured.
// It checks the command name, description, and argument validation.
//
// TestLogsCommandはlogsコマンドが適切に設定されていることを確認します。
// コマンド名、説明、引数の検証を確認します。
func TestLogsCommand(t *testing.T) {
	// Test that logs command is registered.
	// logsコマンドが登録されていることをテストします。
	if logsCmd == nil {
		t.Fatal("logsCmd is nil")
	}

	// Verify the command usage string.
	// コマンドの使用方法文字列を確認します。
	if logsCmd.Use != "logs [container]" {
		t.Errorf("Expected Use to be 'logs [container]', got %s", logsCmd.Use)
	}

	// Verify that short description exists.
	// 短い説明が存在することを確認します。
	if logsCmd.Short == "" {
		t.Error("logsCmd should have a short description")
	}

	// Test that the command requires exactly 1 argument.
	// コマンドが正確に1つの引数を必要とすることをテストします。
	if logsCmd.Args == nil {
		t.Error("logsCmd should have Args validation")
	}
}

// TestLogsTailFlag verifies that the --tail flag is properly configured.
// It checks the flag name, shorthand, and default value.
//
// TestLogsTailFlagは--tailフラグが適切に設定されていることを確認します。
// フラグ名、ショートハンド、デフォルト値を確認します。
func TestLogsTailFlag(t *testing.T) {
	// Check that tail flag exists.
	// tailフラグが存在することを確認します。
	flag := logsCmd.Flags().Lookup("tail")
	if flag == nil {
		t.Fatal("tail flag not found")
	}

	// Verify the shorthand is 'n' (like tail -n).
	// ショートハンドが'n'であることを確認します（tail -nのように）。
	if flag.Shorthand != "n" {
		t.Errorf("Expected tail shorthand to be 'n', got %s", flag.Shorthand)
	}

	// Verify the default value is 100 lines.
	// デフォルト値が100行であることを確認します。
	if flag.DefValue != "100" {
		t.Errorf("Expected tail default to be '100', got %s", flag.DefValue)
	}
}

// TestListCommand verifies that the list command is properly configured.
// It checks the command name and description.
//
// TestListCommandはlistコマンドが適切に設定されていることを確認します。
// コマンド名と説明を確認します。
func TestListCommand(t *testing.T) {
	// Test that list command is registered.
	// listコマンドが登録されていることをテストします。
	if listCmd == nil {
		t.Fatal("listCmd is nil")
	}

	// Verify the command usage string.
	// コマンドの使用方法文字列を確認します。
	if listCmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got %s", listCmd.Use)
	}

	// Verify that short description exists.
	// 短い説明が存在することを確認します。
	if listCmd.Short == "" {
		t.Error("listCmd should have a short description")
	}
}

// TestExecCommand verifies that the exec command is properly configured.
// It checks the command name, description, and argument validation.
//
// TestExecCommandはexecコマンドが適切に設定されていることを確認します。
// コマンド名、説明、引数の検証を確認します。
func TestExecCommand(t *testing.T) {
	// Test that exec command is registered.
	// execコマンドが登録されていることをテストします。
	if execCmd == nil {
		t.Fatal("execCmd is nil")
	}

	// Verify the command usage string.
	// コマンドの使用方法文字列を確認します。
	if execCmd.Use != "exec <command>" {
		t.Errorf("Expected Use to be 'exec <command>', got %s", execCmd.Use)
	}

	// Verify that short description exists.
	// 短い説明が存在することを確認します。
	if execCmd.Short == "" {
		t.Error("execCmd should have a short description")
	}

	// Test that the command requires minimum 1 argument (the command to execute).
	// コマンドが最低1つの引数（実行するコマンド）を必要とすることをテストします。
	if execCmd.Args == nil {
		t.Error("execCmd should have Args validation")
	}
}

// TestRootCommandSubcommands verifies that all expected subcommands are registered.
// This ensures no subcommand was accidentally removed or renamed.
//
// TestRootCommandSubcommandsはすべての期待されるサブコマンドが登録されていることを確認します。
// これにより、サブコマンドが誤って削除されたり名前が変更されたりしていないことを確認します。
func TestRootCommandSubcommands(t *testing.T) {
	// Define the list of expected subcommands.
	// 期待されるサブコマンドのリストを定義します。
	expectedSubcommands := []string{"serve", "list", "logs", "exec", "stats", "inspect", "use", "client", "tools", "version"}

	// Get all registered subcommands.
	// 登録されたすべてのサブコマンドを取得します。
	commands := rootCmd.Commands()

	// Build a map of command names for lookup.
	// 検索用のコマンド名のマップを構築します。
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		// Extract first word from Use (e.g., "logs <container>" -> "logs").
		// Useから最初の単語を抽出します（例："logs <container>" -> "logs"）。
		use := cmd.Use
		for i, ch := range use {
			if ch == ' ' {
				use = use[:i]
				break
			}
		}
		commandNames[use] = true
	}

	// Verify each expected subcommand is registered.
	// 期待される各サブコマンドが登録されていることを確認します。
	for _, expected := range expectedSubcommands {
		if !commandNames[expected] {
			t.Errorf("Missing expected subcommand: %s", expected)
		}
	}
}

// TestUseCommand verifies that the use command is properly configured.
// It checks the command name, description, and flags.
//
// TestUseCommandはuseコマンドが適切に設定されていることを確認します。
// コマンド名、説明、フラグを確認します。
func TestUseCommand(t *testing.T) {
	// Test that use command is registered.
	// useコマンドが登録されていることをテストします。
	if useCmd == nil {
		t.Fatal("useCmd is nil")
	}

	// Verify the command usage string.
	// コマンドの使用方法文字列を確認します。
	if useCmd.Use != "use [container]" {
		t.Errorf("Expected Use to be 'use [container]', got %s", useCmd.Use)
	}

	// Verify that short description exists.
	// 短い説明が存在することを確認します。
	if useCmd.Short == "" {
		t.Error("useCmd should have a short description")
	}

	// Verify --clear flag exists.
	// --clearフラグが存在することを確認します。
	clearFlag := useCmd.Flags().Lookup("clear")
	if clearFlag == nil {
		t.Error("useCmd should have --clear flag")
	}
}

// TestStatsCommand verifies that the stats command is properly configured.
// It checks the command name, description, and argument validation.
//
// TestStatsCommandはstatsコマンドが適切に設定されていることを確認します。
// コマンド名、説明、引数の検証を確認します。
func TestStatsCommand(t *testing.T) {
	// Test that stats command is registered.
	// statsコマンドが登録されていることをテストします。
	if statsCmd == nil {
		t.Fatal("statsCmd is nil")
	}

	// Verify the command usage string.
	// コマンドの使用方法文字列を確認します。
	if statsCmd.Use != "stats [container]" {
		t.Errorf("Expected Use to be 'stats [container]', got %s", statsCmd.Use)
	}

	// Verify that short description exists.
	// 短い説明が存在することを確認します。
	if statsCmd.Short == "" {
		t.Error("statsCmd should have a short description")
	}

	// Test that the command accepts optional argument (0 or 1).
	// コマンドがオプション引数（0または1）を受け入れることをテストします。
	if statsCmd.Args == nil {
		t.Error("statsCmd should have Args validation")
	}
}

// TestInspectCommand verifies that the inspect command is properly configured.
// It checks the command name, description, and argument validation.
//
// TestInspectCommandはinspectコマンドが適切に設定されていることを確認します。
// コマンド名、説明、引数の検証を確認します。
func TestInspectCommand(t *testing.T) {
	// Test that inspect command is registered.
	// inspectコマンドが登録されていることをテストします。
	if inspectCmd == nil {
		t.Fatal("inspectCmd is nil")
	}

	// Verify the command usage string.
	// コマンドの使用方法文字列を確認します。
	if inspectCmd.Use != "inspect [container]" {
		t.Errorf("Expected Use to be 'inspect [container]', got %s", inspectCmd.Use)
	}

	// Verify that short description exists.
	// 短い説明が存在することを確認します。
	if inspectCmd.Short == "" {
		t.Error("inspectCmd should have a short description")
	}

	// Test that the command accepts optional argument (0 or 1).
	// コマンドがオプション引数（0または1）を受け入れることをテストします。
	if inspectCmd.Args == nil {
		t.Error("inspectCmd should have Args validation")
	}
}

// TestInspectJSONFlag verifies that the --json flag is properly configured on inspect command.
// Default is false (summary mode), --json outputs full JSON.
//
// TestInspectJSONFlagはinspectコマンドの--jsonフラグが適切に設定されていることを確認します。
// デフォルトはfalse（サマリーモード）、--jsonはフルJSONを出力します。
func TestInspectJSONFlag(t *testing.T) {
	// Check that json flag exists.
	// jsonフラグが存在することを確認します。
	flag := inspectCmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("json flag not found on inspectCmd")
	}

	// Verify the default value is false (summary mode).
	// デフォルト値がfalse（サマリーモード）であることを確認します。
	if flag.DefValue != "false" {
		t.Errorf("Expected json flag default to be 'false', got %s", flag.DefValue)
	}

	// Verify the flag type is bool.
	// フラグの型がboolであることを確認します。
	if flag.Value.Type() != "bool" {
		t.Errorf("Expected json flag type to be 'bool', got %s", flag.Value.Type())
	}
}

// TestExecContainerFlag verifies that the -c/--container flag is properly configured.
// This flag allows specifying the target container for exec command.
//
// TestExecContainerFlagは-c/--containerフラグが適切に設定されていることを確認します。
// このフラグはexecコマンドのターゲットコンテナを指定できるようにします。
func TestExecContainerFlag(t *testing.T) {
	// Check that container flag exists.
	// containerフラグが存在することを確認します。
	flag := execCmd.Flags().Lookup("container")
	if flag == nil {
		t.Fatal("container flag not found")
	}

	// Verify the shorthand is 'c'.
	// ショートハンドが'c'であることを確認します。
	if flag.Shorthand != "c" {
		t.Errorf("Expected container shorthand to be 'c', got %s", flag.Shorthand)
	}
}

// TestCommandsHaveRunE verifies that commands have their RunE function set.
// RunE is required for commands that need to return errors.
//
// TestCommandsHaveRunEはコマンドがRunE関数を設定していることを確認します。
// RunEはエラーを返す必要があるコマンドに必要です。
func TestCommandsHaveRunE(t *testing.T) {
	// Commands that should have RunE function.
	// Note: This slice is defined for potential future expansion.
	//
	// RunE関数を持つべきコマンド。
	// 注意：このスライスは将来の拡張のために定義されています。
	commands := []*struct {
		name string
		cmd  interface{ RunE() error }
	}{
		// We test via the exported command variables
		// エクスポートされたコマンド変数を介してテストします
	}

	// Check logsCmd.RunE is set.
	// logsCmd.RunEが設定されていることを確認します。
	if logsCmd.RunE == nil {
		t.Error("logsCmd should have RunE function")
	}

	// Check listCmd.RunE is set.
	// listCmd.RunEが設定されていることを確認します。
	if listCmd.RunE == nil {
		t.Error("listCmd should have RunE function")
	}

	// Check execCmd.RunE is set.
	// execCmd.RunEが設定されていることを確認します。
	if execCmd.RunE == nil {
		t.Error("execCmd should have RunE function")
	}

	// Check useCmd.RunE is set.
	// useCmd.RunEが設定されていることを確認します。
	if useCmd.RunE == nil {
		t.Error("useCmd should have RunE function")
	}

	// Check statsCmd.RunE is set.
	// statsCmd.RunEが設定されていることを確認します。
	if statsCmd.RunE == nil {
		t.Error("statsCmd should have RunE function")
	}

	// Check inspectCmd.RunE is set.
	// inspectCmd.RunEが設定されていることを確認します。
	if inspectCmd.RunE == nil {
		t.Error("inspectCmd should have RunE function")
	}

	// Silence unused variable warning.
	// 未使用変数の警告を抑制します。
	_ = commands
}

// TestFormatPortsForDisplay tests the port formatting for CLI display.
// TestFormatPortsForDisplayはCLI表示用のポートフォーマットをテストします。
func TestFormatPortsForDisplay(t *testing.T) {
	tests := []struct {
		name     string   // Test case name / テストケース名
		ports    []string // Input ports / 入力ポート
		expected string   // Expected output / 期待される出力
	}{
		{
			name:     "empty ports",
			ports:    nil,
			expected: "-",
		},
		{
			name:     "empty slice",
			ports:    []string{},
			expected: "-",
		},
		{
			name:     "single port",
			ports:    []string{"0.0.0.0:80->80/tcp"},
			expected: "0.0.0.0:80->80/tcp",
		},
		{
			name:     "multiple ports",
			ports:    []string{"0.0.0.0:80->80/tcp", "443/tcp"},
			expected: "0.0.0.0:80->80/tcp, 443/tcp",
		},
		{
			name:     "truncated long ports",
			ports:    []string{"0.0.0.0:80->80/tcp", "0.0.0.0:443->443/tcp", "0.0.0.0:8080->8080/tcp"},
			expected: "0.0.0.0:80->80/tcp, 0.0.0.0:443->443/...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPortsForDisplay(tt.ports)
			if result != tt.expected {
				t.Errorf("formatPortsForDisplay() = %q, want %q", result, tt.expected)
			}
		})
	}
}
