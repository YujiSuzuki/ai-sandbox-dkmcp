// client_test.go contains unit tests for the client command and its subcommands.
// These tests verify that client commands are properly registered with correct configuration.
//
// client_test.goはclientコマンドとそのサブコマンドのユニットテストを含みます。
// これらのテストはクライアントコマンドが正しい設定で適切に登録されていることを確認します。
//
// IMPORTANT: Scope and Limitations of These Tests
// 重要: これらのテストのスコープと制限事項
//
// Tests like TestClientCommand, TestClientListCommand, etc. verify REGISTRATION only:
// TestClientCommand、TestClientListCommand等のテストは登録のみを検証します：
//
//   - ✅ Subcommand exists (サブコマンドが存在する)
//   - ✅ Flag configuration (フラグ設定)
//   - ❌ Actual HTTP/SSE communication (実際のHTTP/SSE通信)
//   - ❌ RunE execution logic (RunE実行ロジック)
//
// However, this file also contains meaningful tests:
// ただし、このファイルには意味のあるテストも含まれています：
//
//   - TestServerURLEnvVar: Tests environment variable priority logic
//     環境変数の優先順位ロジックをテスト
//   - TestExtractJSONFromMarkdown: Tests JSON extraction from markdown
//     MarkdownからのJSON抽出をテスト
//   - TestParseExitCode: Tests exit code parsing
//     終了コード解析をテスト
//
// The actual client communication is tested in internal/client/client_test.go
// which covers SSE connections, tool calls, and error handling.
// 実際のクライアント通信はinternal/client/client_test.goでテストされ、
// SSE接続、ツール呼び出し、エラーハンドリングがカバーされています。
package cli

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// TestClientCommand verifies that the client command is properly configured.
// It checks the command name and the default URL flag value.
//
// TestClientCommandはclientコマンドが適切に設定されていることを確認します。
// コマンド名とデフォルトのURLフラグ値を確認します。
func TestClientCommand(t *testing.T) {
	// Test that client command is registered.
	// clientコマンドが登録されていることをテストします。
	if clientCmd == nil {
		t.Fatal("clientCmd is nil")
	}

	// Verify the command usage string.
	// コマンドの使用方法文字列を確認します。
	if clientCmd.Use != "client" {
		t.Errorf("Expected Use to be 'client', got %s", clientCmd.Use)
	}

	// Test default URL flag has a value.
	// The actual default is "http://host.docker.internal:8080".
	//
	// デフォルトのURLフラグが値を持っていることをテストします。
	// 実際のデフォルトは"http://host.docker.internal:8080"です。
	if serverURL == "" {
		t.Error("serverURL should have a default value")
	}
}

// TestClientURLFlag verifies that the --url flag is properly configured on client command.
// This is a persistent flag that applies to all subcommands.
//
// TestClientURLFlagはclientコマンドの--urlフラグが適切に設定されていることを確認します。
// これはすべてのサブコマンドに適用される永続フラグです。
func TestClientURLFlag(t *testing.T) {
	// Check that url flag exists as persistent flag.
	// urlフラグが永続フラグとして存在することを確認します。
	flag := clientCmd.PersistentFlags().Lookup("url")
	if flag == nil {
		t.Fatal("url flag not found on clientCmd")
	}

	// Verify the default value is the expected DockMCP server URL.
	// デフォルト値が期待されるDockMCPサーバーURLであることを確認します。
	expectedDefault := "http://host.docker.internal:8080"
	if flag.DefValue != expectedDefault {
		t.Errorf("Expected url flag default to be '%s', got %s", expectedDefault, flag.DefValue)
	}

	// Verify the flag type is string.
	// フラグの型がstringであることを確認します。
	if flag.Value.Type() != "string" {
		t.Errorf("Expected url flag type to be 'string', got %s", flag.Value.Type())
	}
}

// TestClientSubcommands verifies that all expected subcommands are registered.
// This ensures the client command group is complete.
//
// TestClientSubcommandsはすべての期待されるサブコマンドが登録されていることを確認します。
// これによりclientコマンドグループが完全であることを確認します。
func TestClientSubcommands(t *testing.T) {
	// Define the list of expected subcommands.
	// 期待されるサブコマンドのリストを定義します。
	expectedSubcommands := []string{"list", "logs", "exec", "stats", "inspect", "restart", "stop", "start", "host-tools", "host-exec"}

	// Get all registered subcommands under client.
	// client配下のすべての登録されたサブコマンドを取得します。
	commands := clientCmd.Commands()

	// Build a map of command names for lookup.
	// 検索用のコマンド名のマップを構築します。
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		// Extract first word from Use (e.g., "logs CONTAINER" -> "logs").
		// Useから最初の単語を抽出します（例："logs CONTAINER" -> "logs"）。
		use := cmd.Use
		if len(use) > 0 {
			for i, ch := range use {
				if ch == ' ' {
					use = use[:i]
					break
				}
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

// TestClientListCommand verifies that the client list subcommand is properly configured.
//
// TestClientListCommandはclient listサブコマンドが適切に設定されていることを確認します。
func TestClientListCommand(t *testing.T) {
	// Test that client list command is registered.
	// client listコマンドが登録されていることをテストします。
	if clientListCmd == nil {
		t.Fatal("clientListCmd is nil")
	}

	// Verify the command usage string.
	// コマンドの使用方法文字列を確認します。
	if clientListCmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got %s", clientListCmd.Use)
	}
}

// TestClientLogsCommand verifies that the client logs subcommand is properly configured.
//
// TestClientLogsCommandはclient logsサブコマンドが適切に設定されていることを確認します。
func TestClientLogsCommand(t *testing.T) {
	// Test that client logs command is registered.
	// client logsコマンドが登録されていることをテストします。
	if clientLogsCmd == nil {
		t.Fatal("clientLogsCmd is nil")
	}

	// Verify the command usage string includes CONTAINER argument.
	// コマンドの使用方法文字列がCONTAINER引数を含むことを確認します。
	if clientLogsCmd.Use != "logs CONTAINER" {
		t.Errorf("Expected Use to be 'logs CONTAINER', got %s", clientLogsCmd.Use)
	}
}

// TestClientLogsTailFlag verifies that the --tail flag is properly configured on client logs command.
//
// TestClientLogsTailFlagはclient logsコマンドの--tailフラグが適切に設定されていることを確認します。
func TestClientLogsTailFlag(t *testing.T) {
	// Check that tail flag exists.
	// tailフラグが存在することを確認します。
	flag := clientLogsCmd.Flags().Lookup("tail")
	if flag == nil {
		t.Fatal("tail flag not found on clientLogsCmd")
	}

	// Verify the default value is 100.
	// デフォルト値が100であることを確認します。
	if flag.DefValue != "100" {
		t.Errorf("Expected tail flag default to be '100', got %s", flag.DefValue)
	}

	// Verify the flag type is int.
	// フラグの型がintであることを確認します。
	if flag.Value.Type() != "int" {
		t.Errorf("Expected tail flag type to be 'int', got %s", flag.Value.Type())
	}
}

// TestClientLogsSinceFlag verifies that the --since flag is properly configured on client logs command.
//
// TestClientLogsSinceFlagはclient logsコマンドの--sinceフラグが適切に設定されていることを確認します。
func TestClientLogsSinceFlag(t *testing.T) {
	// Check that since flag exists.
	// sinceフラグが存在することを確認します。
	flag := clientLogsCmd.Flags().Lookup("since")
	if flag == nil {
		t.Fatal("since flag not found on clientLogsCmd")
	}

	// Verify the default value is empty string.
	// デフォルト値が空文字列であることを確認します。
	if flag.DefValue != "" {
		t.Errorf("Expected since flag default to be empty, got %s", flag.DefValue)
	}

	// Verify the flag type is string.
	// フラグの型がstringであることを確認します。
	if flag.Value.Type() != "string" {
		t.Errorf("Expected since flag type to be 'string', got %s", flag.Value.Type())
	}
}

// TestClientLogsFollowFlag verifies that the --follow flag is properly configured on client logs command.
//
// TestClientLogsFollowFlagはclient logsコマンドの--followフラグが適切に設定されていることを確認します。
func TestClientLogsFollowFlag(t *testing.T) {
	// Check that follow flag exists.
	// followフラグが存在することを確認します。
	flag := clientLogsCmd.Flags().Lookup("follow")
	if flag == nil {
		t.Fatal("follow flag not found on clientLogsCmd")
	}

	// Verify the default value is false.
	// デフォルト値がfalseであることを確認します。
	if flag.DefValue != "false" {
		t.Errorf("Expected follow flag default to be 'false', got %s", flag.DefValue)
	}

	// Verify the flag type is bool.
	// フラグの型がboolであることを確認します。
	if flag.Value.Type() != "bool" {
		t.Errorf("Expected follow flag type to be 'bool', got %s", flag.Value.Type())
	}
}

// TestClientExecCommand verifies that the client exec subcommand is properly configured.
//
// TestClientExecCommandはclient execサブコマンドが適切に設定されていることを確認します。
func TestClientExecCommand(t *testing.T) {
	// Test that client exec command is registered.
	// client execコマンドが登録されていることをテストします。
	if clientExecCmd == nil {
		t.Fatal("clientExecCmd is nil")
	}

	// Verify the command usage string includes CONTAINER and COMMAND arguments.
	// コマンドの使用方法文字列がCONTAINERとCOMMAND引数を含むことを確認します。
	if clientExecCmd.Use != "exec CONTAINER COMMAND" {
		t.Errorf("Expected Use to be 'exec CONTAINER COMMAND', got %s", clientExecCmd.Use)
	}
}

// TestClientStatsCommand verifies that the client stats subcommand is properly configured.
//
// TestClientStatsCommandはclient statsサブコマンドが適切に設定されていることを確認します。
func TestClientStatsCommand(t *testing.T) {
	// Test that client stats command is registered.
	// client statsコマンドが登録されていることをテストします。
	if clientStatsCmd == nil {
		t.Fatal("clientStatsCmd is nil")
	}

	// Verify the command usage string includes CONTAINER argument.
	// コマンドの使用方法文字列がCONTAINER引数を含むことを確認します。
	if clientStatsCmd.Use != "stats CONTAINER" {
		t.Errorf("Expected Use to be 'stats CONTAINER', got %s", clientStatsCmd.Use)
	}
}

// TestClientInspectCommand verifies that the client inspect subcommand is properly configured.
//
// TestClientInspectCommandはclient inspectサブコマンドが適切に設定されていることを確認します。
func TestClientInspectCommand(t *testing.T) {
	// Test that client inspect command is registered.
	// client inspectコマンドが登録されていることをテストします。
	if clientInspectCmd == nil {
		t.Fatal("clientInspectCmd is nil")
	}

	// Verify the command usage string includes CONTAINER argument.
	// コマンドの使用方法文字列がCONTAINER引数を含むことを確認します。
	if clientInspectCmd.Use != "inspect CONTAINER" {
		t.Errorf("Expected Use to be 'inspect CONTAINER', got %s", clientInspectCmd.Use)
	}
}

// TestClientInspectJSONFlag verifies that the --json flag is properly configured on client inspect command.
// Default is false (summary mode), --json outputs full JSON.
//
// TestClientInspectJSONFlagはclient inspectコマンドの--jsonフラグが適切に設定されていることを確認します。
// デフォルトはfalse（サマリーモード）、--jsonはフルJSONを出力します。
func TestClientInspectJSONFlag(t *testing.T) {
	// Check that json flag exists.
	// jsonフラグが存在することを確認します。
	flag := clientInspectCmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("json flag not found on clientInspectCmd")
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

// TestClientSuffixEnvVar verifies that DOCKMCP_CLIENT_SUFFIX environment variable is used
// as a fallback when --client-suffix flag is not explicitly set.
//
// DOCKMCP_CLIENT_SUFFIX環境変数が、--client-suffixフラグが明示的に設定されていない場合に
// フォールバックとして使用されることを確認します。
func TestClientSuffixEnvVar(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string // DOCKMCP_CLIENT_SUFFIX env var value
		flagArgs       []string
		expectedSuffix string
	}{
		{
			name:           "env var used when flag not set",
			envValue:       "from-env",
			flagArgs:       nil,
			expectedSuffix: "from-env",
		},
		{
			name:           "flag takes precedence over env var",
			envValue:       "from-env",
			flagArgs:       []string{"--client-suffix", "from-flag"},
			expectedSuffix: "from-flag",
		},
		{
			name:           "no suffix when neither set",
			envValue:       "",
			flagArgs:       nil,
			expectedSuffix: "",
		},
		{
			name:           "empty flag overrides env var",
			envValue:       "from-env",
			flagArgs:       []string{"--client-suffix", ""},
			expectedSuffix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original state.
			// 元の状態を保存して復元します。
			originalSuffix := clientSuffix
			originalEnv := os.Getenv("DOCKMCP_CLIENT_SUFFIX")
			defer func() {
				clientSuffix = originalSuffix
				os.Setenv("DOCKMCP_CLIENT_SUFFIX", originalEnv)
			}()

			// Reset clientSuffix to default (empty).
			// clientSuffixをデフォルト（空）にリセットします。
			clientSuffix = ""

			// Set environment variable.
			// 環境変数を設定します。
			if tt.envValue != "" {
				os.Setenv("DOCKMCP_CLIENT_SUFFIX", tt.envValue)
			} else {
				os.Unsetenv("DOCKMCP_CLIENT_SUFFIX")
			}

			// Create a fresh command to avoid flag state leaking between tests.
			// テスト間でフラグの状態が漏れないように新しいコマンドを作成します。
			cmd := &cobra.Command{Use: "test"}
			cmd.PersistentFlags().StringVarP(&clientSuffix, "client-suffix", "s", "", "")
			if tt.flagArgs != nil {
				cmd.SetArgs(tt.flagArgs)
				cmd.ParseFlags(tt.flagArgs)
			}

			// Execute PersistentPreRunE directly.
			// PersistentPreRunEを直接実行します。
			err := clientCmd.PersistentPreRunE(cmd, []string{})
			if err != nil {
				t.Fatalf("PersistentPreRunE returned error: %v", err)
			}

			if clientSuffix != tt.expectedSuffix {
				t.Errorf("clientSuffix = %q, want %q", clientSuffix, tt.expectedSuffix)
			}
		})
	}
}

// TestServerURLEnvVar verifies that DOCKMCP_SERVER_URL environment variable is used
// as a fallback when --url flag is not explicitly set.
//
// DOCKMCP_SERVER_URL環境変数が、--urlフラグが明示的に設定されていない場合に
// フォールバックとして使用されることを確認します。
func TestServerURLEnvVar(t *testing.T) {
	const defaultURL = "http://host.docker.internal:8080"

	tests := []struct {
		name        string
		envValue    string // DOCKMCP_SERVER_URL env var value
		flagArgs    []string
		expectedURL string
	}{
		{
			name:        "env var used when flag not set",
			envValue:    "http://localhost:9090",
			flagArgs:    nil,
			expectedURL: "http://localhost:9090",
		},
		{
			name:        "flag takes precedence over env var",
			envValue:    "http://localhost:9090",
			flagArgs:    []string{"--url", "http://custom:7070"},
			expectedURL: "http://custom:7070",
		},
		{
			name:        "default used when neither set",
			envValue:    "",
			flagArgs:    nil,
			expectedURL: defaultURL,
		},
		{
			name:        "explicit flag value overrides env var even if same as default",
			envValue:    "http://localhost:9090",
			flagArgs:    []string{"--url", defaultURL},
			expectedURL: defaultURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original state.
			// 元の状態を保存して復元します。
			originalURL := serverURL
			originalEnv := os.Getenv("DOCKMCP_SERVER_URL")
			defer func() {
				serverURL = originalURL
				os.Setenv("DOCKMCP_SERVER_URL", originalEnv)
			}()

			// Reset serverURL to default.
			// serverURLをデフォルトにリセットします。
			serverURL = defaultURL

			// Set environment variable.
			// 環境変数を設定します。
			if tt.envValue != "" {
				os.Setenv("DOCKMCP_SERVER_URL", tt.envValue)
			} else {
				os.Unsetenv("DOCKMCP_SERVER_URL")
			}

			// Create a fresh command to avoid flag state leaking between tests.
			// テスト間でフラグの状態が漏れないように新しいコマンドを作成します。
			cmd := &cobra.Command{Use: "test"}
			cmd.PersistentFlags().StringVar(&serverURL, "url", defaultURL, "")
			if tt.flagArgs != nil {
				cmd.SetArgs(tt.flagArgs)
				cmd.ParseFlags(tt.flagArgs)
			}

			// Execute PersistentPreRunE directly.
			// PersistentPreRunEを直接実行します。
			err := clientCmd.PersistentPreRunE(cmd, []string{})
			if err != nil {
				t.Fatalf("PersistentPreRunE returned error: %v", err)
			}

			if serverURL != tt.expectedURL {
				t.Errorf("serverURL = %q, want %q", serverURL, tt.expectedURL)
			}
		})
	}
}

// TestExtractJSONFromMarkdown tests the extraction of JSON from Markdown code blocks.
// The MCP server wraps JSON responses in Markdown format for AI assistants.
//
// TestExtractJSONFromMarkdownはMarkdownコードブロックからのJSON抽出をテストします。
// MCPサーバーはAIアシスタント向けにJSONレスポンスをMarkdown形式でラップします。
func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string // Test case name / テストケース名
		input    string // Input content / 入力コンテンツ
		expected string // Expected output / 期待される出力
	}{
		{
			name: "standard markdown format",
			input: `Container 'test' details:

` + "```json\n" + `{"Id": "abc123", "Name": "test"}` + "\n```",
			expected: `{"Id": "abc123", "Name": "test"}`,
		},
		{
			name: "multiline JSON in markdown",
			input: `Container 'test' details:

` + "```json\n" + `{
  "Id": "abc123",
  "Name": "test"
}` + "\n```",
			expected: `{
  "Id": "abc123",
  "Name": "test"
}`,
		},
		{
			name:     "plain JSON without markdown",
			input:    `{"Id": "abc123", "Name": "test"}`,
			expected: `{"Id": "abc123", "Name": "test"}`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name: "markdown without closing marker",
			input: `Container 'test' details:

` + "```json\n" + `{"Id": "abc123"}`,
			expected: `{"Id": "abc123"}`,
		},
		{
			name: "text before and after code block",
			input: `Some header text

` + "```json\n" + `{"key": "value"}` + "\n```" + `

Some footer text`,
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSONFromMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestClientLifecycleCommands verifies that lifecycle subcommands are registered with correct flags.
// TestClientLifecycleCommandsはlifecycleサブコマンドが正しいフラグで登録されていることを確認します。
func TestClientLifecycleCommands(t *testing.T) {
	// Restart command
	if clientRestartCmd == nil {
		t.Fatal("clientRestartCmd is nil")
	}
	if clientRestartCmd.Use != "restart CONTAINER" {
		t.Errorf("Expected Use 'restart CONTAINER', got %s", clientRestartCmd.Use)
	}
	flag := clientRestartCmd.Flags().Lookup("timeout")
	if flag == nil {
		t.Fatal("timeout flag not found on clientRestartCmd")
	}
	if flag.Value.Type() != "int" {
		t.Errorf("Expected timeout flag type 'int', got %s", flag.Value.Type())
	}

	// Stop command
	if clientStopCmd == nil {
		t.Fatal("clientStopCmd is nil")
	}
	if clientStopCmd.Use != "stop CONTAINER" {
		t.Errorf("Expected Use 'stop CONTAINER', got %s", clientStopCmd.Use)
	}
	flag = clientStopCmd.Flags().Lookup("timeout")
	if flag == nil {
		t.Fatal("timeout flag not found on clientStopCmd")
	}

	// Start command
	if clientStartCmd == nil {
		t.Fatal("clientStartCmd is nil")
	}
	if clientStartCmd.Use != "start CONTAINER" {
		t.Errorf("Expected Use 'start CONTAINER', got %s", clientStartCmd.Use)
	}
}

// TestClientHostToolsCommands verifies that host-tools subcommands are registered.
// TestClientHostToolsCommandsはhost-toolsサブコマンドが登録されていることを確認します。
func TestClientHostToolsCommands(t *testing.T) {
	if clientHostToolsCmd == nil {
		t.Fatal("clientHostToolsCmd is nil")
	}
	if clientHostToolsCmd.Use != "host-tools" {
		t.Errorf("Expected Use 'host-tools', got %s", clientHostToolsCmd.Use)
	}

	// Verify sub-subcommands
	// サブサブコマンドを確認
	subCmds := clientHostToolsCmd.Commands()
	subNames := make(map[string]bool)
	for _, cmd := range subCmds {
		use := cmd.Use
		for i, ch := range use {
			if ch == ' ' {
				use = use[:i]
				break
			}
		}
		subNames[use] = true
	}

	for _, expected := range []string{"list", "info", "run"} {
		if !subNames[expected] {
			t.Errorf("Missing host-tools subcommand: %s", expected)
		}
	}
}

// TestClientHostExecCommand verifies that host-exec command is registered with correct flags.
// TestClientHostExecCommandはhost-execコマンドが正しいフラグで登録されていることを確認します。
func TestClientHostExecCommand(t *testing.T) {
	if clientHostExecCmd == nil {
		t.Fatal("clientHostExecCmd is nil")
	}
	if clientHostExecCmd.Use != "host-exec COMMAND" {
		t.Errorf("Expected Use 'host-exec COMMAND', got %s", clientHostExecCmd.Use)
	}

	flag := clientHostExecCmd.Flags().Lookup("dangerously")
	if flag == nil {
		t.Fatal("dangerously flag not found on clientHostExecCmd")
	}
	if flag.DefValue != "false" {
		t.Errorf("Expected dangerously default 'false', got %s", flag.DefValue)
	}
}
