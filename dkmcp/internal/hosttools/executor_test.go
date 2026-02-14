package hosttools

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

func TestRunTool_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	_, err := RunTool(dir, "../etc/passwd", nil, 10*time.Second, "")
	if err == nil {
		t.Error("RunTool should reject path traversal")
	}
}

func TestRunTool_UnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tool.rb"), []byte("# ruby\n"), 0755)

	_, err := RunTool(dir, "tool.rb", nil, 10*time.Second, "")
	if err == nil {
		t.Error("RunTool should reject unsupported extension")
	}
}

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

func TestExecHostCommand_EmptyCommand(t *testing.T) {
	_, err := ExecHostCommand("", "/tmp", 10*time.Second)
	if err == nil {
		t.Error("ExecHostCommand should error for empty command")
	}
}

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
