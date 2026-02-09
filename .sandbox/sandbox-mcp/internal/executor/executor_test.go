package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"valid-script.sh", false},
		{"", true},
		{"../etc/passwd", true},
		{"foo/bar.sh", true},
		{"normal.sh", false},
	}

	for _, tt := range tests {
		err := validateName(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestRunScriptRejectsNonSh(t *testing.T) {
	_, err := RunScript("/tmp", "notashell.py", nil)
	if err == nil {
		t.Error("Expected error for non-.sh file")
	}
}

func TestRunToolRejectsNonGo(t *testing.T) {
	_, err := RunTool("/tmp", "notago.py", nil)
	if err == nil {
		t.Error("Expected error for non-.go file")
	}
}

func TestResultString(t *testing.T) {
	r := &Result{Stdout: "hello", ExitCode: 0}
	if r.String() != "hello" {
		t.Errorf("String() = %q, want %q", r.String(), "hello")
	}

	r = &Result{Stdout: "out", Stderr: "err", ExitCode: 1}
	s := r.String()
	if s == "" {
		t.Error("Expected non-empty string")
	}
}

func TestRunTimeoutReturnsTimeoutError(t *testing.T) {
	// Invoke sleep directly (not via bash) so SIGKILL terminates the process cleanly
	_, err := runWithTimeout("sleep", []string{"60"}, 100*time.Millisecond)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error message, got: %v", err)
	}
}

func TestRunNonZeroExitCode(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fail.sh")
	os.WriteFile(script, []byte("#!/bin/bash\nexit 42\n"), 0755)

	result, err := runWithTimeout(script, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Expected no error for non-zero exit, got: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
}

// Smoke test: ensures the happy path of runWithTimeout is not broken.
// Kept to cover all 3 paths alongside the timeout and non-zero exit tests.
// Smoke test: runWithTimeout の正常系が壊れていないことを保証する。
// タイムアウト・異常終了のテストと合わせて3パスを網羅する目的で残している。
func TestRunSuccess(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "ok.sh")
	os.WriteFile(script, []byte("#!/bin/bash\necho hello\n"), 0755)

	result, err := runWithTimeout(script, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if strings.TrimSpace(result.Stdout) != "hello" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "hello\n")
	}
}

func TestRunScriptHappyPath(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "greet.sh")
	os.WriteFile(script, []byte("#!/bin/bash\necho \"hello $1\"\n"), 0755)

	result, err := RunScript(dir, "greet.sh", []string{"world"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("Stdout = %q, want to contain %q", result.Stdout, "hello world")
	}
}

func TestRunToolHappyPath(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "hello.go")
	os.WriteFile(tool, []byte(`package main

import "fmt"

func main() { fmt.Println("hello from tool") }
`), 0644)

	result, err := RunTool(dir, "hello.go", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello from tool") {
		t.Errorf("Stdout = %q, want to contain %q", result.Stdout, "hello from tool")
	}
}

func TestRunCommandNotFound(t *testing.T) {
	_, err := runWithTimeout("/nonexistent/command/xyz", nil, 5*time.Second)
	if err == nil {
		t.Fatal("Expected error for non-existent command")
	}
	if !strings.Contains(err.Error(), "execution error") {
		t.Errorf("Error = %q, expected to contain %q", err.Error(), "execution error")
	}
}

func TestResultStringStderrOnly(t *testing.T) {
	r := &Result{Stderr: "warning message", ExitCode: 0}
	s := r.String()
	if !strings.Contains(s, "[stderr]") {
		t.Errorf("String() = %q, expected to contain [stderr]", s)
	}
	if !strings.Contains(s, "warning message") {
		t.Errorf("String() = %q, expected to contain stderr content", s)
	}
}

func TestResultStringEmpty(t *testing.T) {
	r := &Result{}
	if r.String() != "" {
		t.Errorf("String() = %q, want empty string", r.String())
	}
}

func TestResultStringExitCodeOnly(t *testing.T) {
	r := &Result{ExitCode: 1}
	s := r.String()
	if !strings.Contains(s, "[exit code: 1]") {
		t.Errorf("String() = %q, expected to contain exit code", s)
	}
}

func TestResultStringAllFields(t *testing.T) {
	r := &Result{Stdout: "output", Stderr: "error", ExitCode: 2}
	s := r.String()
	if !strings.Contains(s, "output") {
		t.Errorf("String() missing stdout")
	}
	if !strings.Contains(s, "[stderr]") {
		t.Errorf("String() missing stderr header")
	}
	if !strings.Contains(s, "error") {
		t.Errorf("String() missing stderr content")
	}
	if !strings.Contains(s, "[exit code: 2]") {
		t.Errorf("String() missing exit code")
	}
}

func TestRunScriptPathTraversal(t *testing.T) {
	_, err := RunScript("/tmp", "../etc/passwd.sh", nil)
	if err == nil {
		t.Error("Expected error for path traversal")
	}
}

func TestRunToolPathTraversal(t *testing.T) {
	_, err := RunTool("/tmp", "../etc/passwd.go", nil)
	if err == nil {
		t.Error("Expected error for path traversal")
	}
}
