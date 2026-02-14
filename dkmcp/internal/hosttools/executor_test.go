// Package hosttools tests verify host tool execution, argument parsing,
// timeout handling, and working directory management.
//
// hosttoolsパッケージのテストはホストツールの実行、引数の解析、
// タイムアウト処理、作業ディレクトリ管理を検証します。
package hosttools

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRunTool_ShellScript verifies basic shell script execution.
// Tests that a simple shell script runs successfully and produces expected output.
//
// TestRunTool_ShellScriptは基本的なシェルスクリプトの実行を検証します。
// シンプルなシェルスクリプトが正常に実行され、期待される出力を生成することをテストします。
func TestRunTool_ShellScript(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hello.sh")
	os.WriteFile(script, []byte("#!/bin/bash\n# hello.sh\n# Hello tool\necho hello world\n"), 0755)

	result, err := RunTool(dir, "hello.sh", nil, 10*time.Second, "")
	if err != nil {
		t.Fatalf("RunTool error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "hello world\n" {
		t.Errorf("Stdout = %q, want 'hello world\\n'", result.Stdout)
	}
}

// TestRunTool_WithArgs verifies that arguments are passed correctly to tools.
// Tests that tools receive and process command-line arguments as expected.
//
// TestRunTool_WithArgsは引数がツールに正しく渡されることを検証します。
// ツールがコマンドライン引数を期待通りに受け取り処理することをテストします。
func TestRunTool_WithArgs(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "echo-args.sh")
	os.WriteFile(script, []byte("#!/bin/bash\n# echo-args.sh\n# Echo args\necho \"$@\"\n"), 0755)

	result, err := RunTool(dir, "echo-args.sh", []string{"foo", "bar"}, 10*time.Second, "")
	if err != nil {
		t.Fatalf("RunTool error: %v", err)
	}

	if result.Stdout != "foo bar\n" {
		t.Errorf("Stdout = %q, want 'foo bar\\n'", result.Stdout)
	}
}

// TestRunTool_PathTraversal verifies that path traversal attempts are rejected.
// Tests that tools cannot access files outside the tool directory using ".." paths.
//
// TestRunTool_PathTraversalはパストラバーサル攻撃が拒否されることを検証します。
// ツールが".."パスを使ってツールディレクトリ外のファイルにアクセスできないことをテストします。
func TestRunTool_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	_, err := RunTool(dir, "../etc/passwd", nil, 10*time.Second, "")
	if err == nil {
		t.Error("RunTool should reject path traversal")
	}
}

// TestRunTool_UnsupportedExtension verifies that unsupported file types are rejected.
// Tests that only whitelisted file extensions (.sh, .py, etc.) can be executed.
//
// TestRunTool_UnsupportedExtensionは非対応のファイルタイプが拒否されることを検証します。
// ホワイトリストに登録された拡張子(.sh、.pyなど)のみ実行可能であることをテストします。
func TestRunTool_UnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tool.rb"), []byte("# ruby\n"), 0755)

	_, err := RunTool(dir, "tool.rb", nil, 10*time.Second, "")
	if err == nil {
		t.Error("RunTool should reject unsupported extension")
	}
}

// TestRunTool_Timeout verifies that long-running tools are terminated.
// Tests that tools exceeding the timeout limit are killed and return an error.
//
// TestRunTool_Timeoutは長時間実行されるツールが終了されることを検証します。
// タイムアウト制限を超えたツールが強制終了され、エラーが返されることをテストします。
func TestRunTool_Timeout(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "slow.sh")
	os.WriteFile(script, []byte("#!/bin/bash\nsleep 5\n"), 0755)

	_, err := RunTool(dir, "slow.sh", nil, 200*time.Millisecond, "")
	if err == nil {
		t.Error("RunTool should return timeout error")
	}
	if err != nil && !containsTimeout(err.Error()) {
		t.Errorf("error should mention timeout, got: %v", err)
	}
}

// TestRunTool_NonZeroExitCode verifies that non-zero exit codes are captured.
// Tests that tools exiting with non-zero codes return the correct ExitCode without error.
//
// TestRunTool_NonZeroExitCodeは非ゼロ終了コードがキャプチャされることを検証します。
// 非ゼロコードで終了したツールがエラーではなく正しいExitCodeを返すことをテストします。
func TestRunTool_NonZeroExitCode(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fail.sh")
	os.WriteFile(script, []byte("#!/bin/bash\nexit 42\n"), 0755)

	result, err := RunTool(dir, "fail.sh", nil, 10*time.Second, "")
	if err != nil {
		t.Fatalf("RunTool should not error for non-zero exit code, got: %v", err)
	}

	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
}

// TestRunTool_WorkDir verifies that tools run in the specified working directory.
// Tests that the workDir parameter correctly sets the tool's execution directory.
//
// TestRunTool_WorkDirは指定された作業ディレクトリでツールが実行されることを検証します。
// workDirパラメータがツールの実行ディレクトリを正しく設定することをテストします。
func TestRunTool_WorkDir(t *testing.T) {
	toolDir := t.TempDir()
	workDir := t.TempDir()
	script := filepath.Join(toolDir, "pwd.sh")
	os.WriteFile(script, []byte("#!/bin/bash\npwd\n"), 0755)

	result, err := RunTool(toolDir, "pwd.sh", nil, 10*time.Second, workDir)
	if err != nil {
		t.Fatalf("RunTool error: %v", err)
	}

	got := result.Stdout[:len(result.Stdout)-1] // trim newline
	if got != workDir {
		t.Errorf("working dir = %q, want %q", got, workDir)
	}
}

// TestRunTool_WorkDir_Empty_FallsBackToToolDir verifies workDir fallback behavior.
// Tests that when workDir is empty, the tool runs in the tool directory instead.
//
// TestRunTool_WorkDir_Empty_FallsBackToToolDirは作業ディレクトリのフォールバック動作を検証します。
// workDirが空の場合、ツールディレクトリで実行されることをテストします。
func TestRunTool_WorkDir_Empty_FallsBackToToolDir(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "pwd.sh")
	os.WriteFile(script, []byte("#!/bin/bash\npwd\n"), 0755)

	result, err := RunTool(dir, "pwd.sh", nil, 10*time.Second, "")
	if err != nil {
		t.Fatalf("RunTool error: %v", err)
	}

	got := result.Stdout[:len(result.Stdout)-1]
	if got != dir {
		t.Errorf("working dir = %q, want %q (tool dir fallback)", got, dir)
	}
}

// TestExecHostCommand verifies basic host command execution.
// Tests that simple shell commands run successfully and produce expected output.
//
// TestExecHostCommandは基本的なホストコマンドの実行を検証します。
// シンプルなシェルコマンドが正常に実行され、期待される出力を生成することをテストします。
func TestExecHostCommand(t *testing.T) {
	dir := t.TempDir()

	result, err := ExecHostCommand("echo hello world", dir, 10*time.Second)
	if err != nil {
		t.Fatalf("ExecHostCommand error: %v", err)
	}

	if result.Stdout != "hello world\n" {
		t.Errorf("Stdout = %q, want 'hello world\\n'", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

// TestExecHostCommand_WorkingDirectory verifies working directory handling.
// Tests that host commands execute in the specified working directory.
//
// TestExecHostCommand_WorkingDirectoryは作業ディレクトリの処理を検証します。
// ホストコマンドが指定された作業ディレクトリで実行されることをテストします。
func TestExecHostCommand_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := ExecHostCommand("pwd", dir, 10*time.Second)
	if err != nil {
		t.Fatalf("ExecHostCommand error: %v", err)
	}

	// pwd output should match the directory
	got := result.Stdout
	// Trim trailing newline
	got = got[:len(got)-1]
	if got != dir {
		t.Errorf("pwd output = %q, want %q", got, dir)
	}
}

// TestExecHostCommand_EmptyCommand verifies empty command rejection.
// Tests that empty command strings are rejected with an error.
//
// TestExecHostCommand_EmptyCommandは空のコマンドの拒否を検証します。
// 空のコマンド文字列がエラーとして拒否されることをテストします。
func TestExecHostCommand_EmptyCommand(t *testing.T) {
	_, err := ExecHostCommand("", "/tmp", 10*time.Second)
	if err == nil {
		t.Error("ExecHostCommand should error for empty command")
	}
}

// TestParseCommandArgs verifies command string parsing into argument arrays.
// Tests various quoting styles, escaping, and error cases for shell command parsing.
//
// TestParseCommandArgsはコマンド文字列の引数配列への解析を検証します。
// シェルコマンド解析における様々なクォート形式、エスケープ、エラーケースをテストします。
func TestParseCommandArgs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"simple", "echo hello", []string{"echo", "hello"}, false},
		{"quoted", `echo "hello world"`, []string{"echo", "hello world"}, false},
		{"single quoted", `echo 'hello world'`, []string{"echo", "hello world"}, false},
		{"escaped space", `echo hello\ world`, []string{"echo", "hello world"}, false},
		{"multiple args", "docker ps -a", []string{"docker", "ps", "-a"}, false},
		{"unclosed quote", `echo "hello`, nil, true},
		{"empty", "", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCommandArgs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCommandArgs(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("parseCommandArgs(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseCommandArgs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestResultString verifies that Result.String() produces output.
// Tests that the string representation of execution results is non-empty.
//
// TestResultStringはResult.String()が出力を生成することを検証します。
// 実行結果の文字列表現が空でないことをテストします。
func TestResultString(t *testing.T) {
	r := &Result{Stdout: "output", Stderr: "error", ExitCode: 1}
	s := r.String()
	if s == "" {
		t.Error("Result.String() should not be empty")
	}
}

func containsTimeout(s string) bool {
	return len(s) > 0 && (contains(s, "timed out") || contains(s, "timeout"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
