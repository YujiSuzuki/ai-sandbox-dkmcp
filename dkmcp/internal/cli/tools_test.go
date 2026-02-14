// tools_test.go contains unit and integration tests for the tools command group.
// These tests verify command registration, flag configuration, and the actual
// execution logic of runToolsList and runToolsSync functions.
//
// tools_test.goはtoolsコマンドグループのユニットテストと統合テストを含みます。
// コマンド登録、フラグ設定、runToolsListおよびrunToolsSync関数の
// 実際の実行ロジックを検証します。
package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestToolsCommand verifies that the tools command is properly configured.
// Checks command name, short description, and that it is not nil.
//
// TestToolsCommandはtoolsコマンドが適切に設定されていることを確認します。
// コマンド名、短い説明文、nilでないことを確認します。
func TestToolsCommand(t *testing.T) {
	if toolsCmd == nil {
		t.Fatal("toolsCmd is nil")
	}

	if toolsCmd.Use != "tools" {
		t.Errorf("toolsCmd.Use = %q, want %q", toolsCmd.Use, "tools")
	}

	if toolsCmd.Short == "" {
		t.Error("toolsCmd.Short should not be empty")
	}
}

// TestToolsSubcommands verifies that sync and list subcommands are registered.
// Ensures no subcommand was accidentally removed or renamed.
//
// TestToolsSubcommandsはsyncとlistサブコマンドが登録されていることを確認します。
// サブコマンドが誤って削除・名前変更されていないことを確認します。
func TestToolsSubcommands(t *testing.T) {
	expectedSubcommands := []string{"sync", "list"}

	commands := toolsCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Use] = true
	}

	for _, name := range expectedSubcommands {
		if !commandNames[name] {
			t.Errorf("expected subcommand %q not found under tools", name)
		}
	}
}

// TestToolsWorkspaceFlag verifies that the --workspace persistent flag is configured.
// This flag allows overriding the workspace root directory for tools commands.
//
// TestToolsWorkspaceFlagは--workspace永続フラグが設定されていることを確認します。
// このフラグはtoolsコマンドのワークスペースルートディレクトリを上書きできます。
func TestToolsWorkspaceFlag(t *testing.T) {
	flag := toolsCmd.PersistentFlags().Lookup("workspace")
	if flag == nil {
		t.Fatal("--workspace flag not found on toolsCmd")
	}

	if flag.DefValue != "" {
		t.Errorf("--workspace default = %q, want empty string", flag.DefValue)
	}
}

// TestToolsSyncHasRunE verifies that the sync subcommand has a RunE function.
//
// TestToolsSyncHasRunEはsyncサブコマンドにRunE関数があることを確認します。
func TestToolsSyncHasRunE(t *testing.T) {
	if toolsSyncCmd.RunE == nil {
		t.Error("toolsSyncCmd.RunE should not be nil")
	}
}

// TestToolsListHasRunE verifies that the list subcommand has a RunE function.
//
// TestToolsListHasRunEはlistサブコマンドにRunE関数があることを確認します。
func TestToolsListHasRunE(t *testing.T) {
	if toolsListCmd.RunE == nil {
		t.Error("toolsListCmd.RunE should not be nil")
	}
}

// --- Integration tests using temp directories and config files ---

// writeTestConfig creates a temporary YAML config file and returns its path.
// The caller should use t.TempDir() for the parent directory.
//
// writeTestConfigは一時的なYAML設定ファイルを作成し、そのパスを返します。
// 呼び出し元はt.TempDir()を親ディレクトリに使用してください。
func writeTestConfig(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "dkmcp.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}

// createTestTool creates a minimal shell script in the given directory.
//
// createTestToolは指定ディレクトリにシンプルなシェルスクリプトを作成します。
func createTestTool(t *testing.T, dir, name, description string) {
	t.Helper()
	content := fmt.Sprintf("#!/bin/bash\n# %s\necho hello\n", description)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write test tool: %v", err)
	}
}

// TestRunToolsList_Disabled verifies that runToolsList returns an error
// when host_tools is not enabled in the configuration.
//
// TestRunToolsList_Disabledはhost_toolsが設定で無効な場合に
// runToolsListがエラーを返すことを確認します。
func TestRunToolsList_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir, `
server:
  port: 8080
security:
  mode: permissive
host_access:
  host_tools:
    enabled: false
`)

	oldCfgFile := cfgFile
	cfgFile = configPath
	defer func() { cfgFile = oldCfgFile }()

	err := runToolsList(toolsListCmd, nil)
	if err == nil {
		t.Fatal("expected error when host_tools is disabled")
	}
	if !strings.Contains(err.Error(), "not enabled") {
		t.Errorf("error = %q, want it to contain 'not enabled'", err.Error())
	}
}

// TestRunToolsList_LegacyMode verifies that runToolsList shows legacy mode output
// when approved_dir is not configured and tools are listed from directories.
//
// TestRunToolsList_LegacyModeはapproved_dirが未設定の場合に
// レガシーモード出力とディレクトリからのツール一覧表示を確認します。
func TestRunToolsList_LegacyMode(t *testing.T) {
	tmpDir := t.TempDir()
	toolsDir := filepath.Join(tmpDir, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		t.Fatalf("failed to create tools dir: %v", err)
	}
	createTestTool(t, toolsDir, "my-tool.sh", "A test tool")

	configPath := writeTestConfig(t, tmpDir, fmt.Sprintf(`
server:
  port: 8080
security:
  mode: permissive
host_access:
  workspace_root: %s
  host_tools:
    enabled: true
    directories:
      - %s
`, tmpDir, toolsDir))

	oldCfgFile := cfgFile
	cfgFile = configPath
	defer func() { cfgFile = oldCfgFile }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runToolsList(toolsListCmd, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Mode:         legacy") {
		t.Errorf("output should contain 'Mode:         legacy', got:\n%s", output)
	}
	if !strings.Contains(output, "my-tool.sh") {
		t.Errorf("output should list 'my-tool.sh', got:\n%s", output)
	}
	if !strings.Contains(output, "Tools (1)") {
		t.Errorf("output should contain 'Tools (1)', got:\n%s", output)
	}
}

// TestRunToolsList_SecureMode verifies that runToolsList shows secure mode output
// with project ID and approved directory when approved_dir is configured.
//
// TestRunToolsList_SecureModeはapproved_dirが設定されている場合に
// プロジェクトIDと承認済みディレクトリを含むセキュアモード出力を確認します。
func TestRunToolsList_SecureMode(t *testing.T) {
	tmpDir := t.TempDir()
	approvedDir := filepath.Join(tmpDir, "approved")
	projectDir := filepath.Join(approvedDir, "projects")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create approved dir: %v", err)
	}

	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	configPath := writeTestConfig(t, tmpDir, fmt.Sprintf(`
server:
  port: 8080
security:
  mode: permissive
host_access:
  workspace_root: %s
  host_tools:
    enabled: true
    approved_dir: %s
`, workspaceDir, approvedDir))

	oldCfgFile := cfgFile
	cfgFile = configPath
	defer func() { cfgFile = oldCfgFile }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runToolsList(toolsListCmd, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Mode:         secure") {
		t.Errorf("output should contain 'Mode:         secure', got:\n%s", output)
	}
	if !strings.Contains(output, "Project ID:") {
		t.Errorf("output should contain 'Project ID:', got:\n%s", output)
	}
	if !strings.Contains(output, "Approved dir:") {
		t.Errorf("output should contain 'Approved dir:', got:\n%s", output)
	}
}

// TestRunToolsList_NoTools verifies that runToolsList outputs "No tools found."
// when the tools directory is empty.
//
// TestRunToolsList_NoToolsはツールディレクトリが空の場合に
// "No tools found."が出力されることを確認します。
func TestRunToolsList_NoTools(t *testing.T) {
	tmpDir := t.TempDir()
	emptyToolsDir := filepath.Join(tmpDir, "empty-tools")
	if err := os.MkdirAll(emptyToolsDir, 0755); err != nil {
		t.Fatalf("failed to create empty tools dir: %v", err)
	}

	configPath := writeTestConfig(t, tmpDir, fmt.Sprintf(`
server:
  port: 8080
security:
  mode: permissive
host_access:
  workspace_root: %s
  host_tools:
    enabled: true
    directories:
      - %s
`, tmpDir, emptyToolsDir))

	oldCfgFile := cfgFile
	cfgFile = configPath
	defer func() { cfgFile = oldCfgFile }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runToolsList(toolsListCmd, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "No tools found") {
		t.Errorf("output should contain 'No tools found', got:\n%s", output)
	}
}

// TestRunToolsSync_NotEnabled verifies that runToolsSync returns an error
// when host_tools is not enabled in configuration.
//
// TestRunToolsSync_NotEnabledはhost_toolsが設定で無効な場合に
// runToolsSyncがエラーを返すことを確認します。
func TestRunToolsSync_NotEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir, `
server:
  port: 8080
security:
  mode: permissive
host_access:
  host_tools:
    enabled: false
`)

	oldCfgFile := cfgFile
	cfgFile = configPath
	defer func() { cfgFile = oldCfgFile }()

	err := runToolsSync(toolsSyncCmd, nil)
	if err == nil {
		t.Fatal("expected error when host_tools is disabled")
	}
	if !strings.Contains(err.Error(), "not enabled") {
		t.Errorf("error = %q, want it to contain 'not enabled'", err.Error())
	}
}

// TestRunToolsSync_NotSecureMode verifies that runToolsSync returns an error
// when approved_dir is not configured (legacy mode doesn't support sync).
//
// TestRunToolsSync_NotSecureModeはapproved_dirが未設定の場合に
// runToolsSyncがエラーを返すことを確認します（レガシーモードはsync非対応）。
func TestRunToolsSync_NotSecureMode(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir, fmt.Sprintf(`
server:
  port: 8080
security:
  mode: permissive
host_access:
  workspace_root: %s
  host_tools:
    enabled: true
    directories:
      - /some/dir
`, tmpDir))

	oldCfgFile := cfgFile
	cfgFile = configPath
	defer func() { cfgFile = oldCfgFile }()

	err := runToolsSync(toolsSyncCmd, nil)
	if err == nil {
		t.Fatal("expected error when approved_dir is not configured")
	}
	if !strings.Contains(err.Error(), "approved_dir") {
		t.Errorf("error = %q, want it to contain 'approved_dir'", err.Error())
	}
}

// TestRunToolsList_WorkspaceFlag verifies that the --workspace flag overrides
// the workspace_root from the configuration file.
//
// TestRunToolsList_WorkspaceFlagは--workspaceフラグが設定ファイルの
// workspace_rootを上書きすることを確認します。
func TestRunToolsList_WorkspaceFlag(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "custom-workspace")
	toolsDir := filepath.Join(tmpDir, "tools")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		t.Fatalf("failed to create tools dir: %v", err)
	}

	configPath := writeTestConfig(t, tmpDir, fmt.Sprintf(`
server:
  port: 8080
security:
  mode: permissive
host_access:
  workspace_root: /nonexistent/original
  host_tools:
    enabled: true
    directories:
      - %s
`, toolsDir))

	oldCfgFile := cfgFile
	cfgFile = configPath
	defer func() { cfgFile = oldCfgFile }()

	// Override workspace via package variable (simulating --workspace flag)
	oldWorkspace := flagToolsWorkspace
	flagToolsWorkspace = workspaceDir
	defer func() { flagToolsWorkspace = oldWorkspace }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runToolsList(toolsListCmd, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, workspaceDir) {
		t.Errorf("output should contain workspace path %q, got:\n%s", workspaceDir, output)
	}
}
