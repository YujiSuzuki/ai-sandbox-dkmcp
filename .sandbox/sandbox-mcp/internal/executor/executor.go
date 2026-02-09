// Package executor runs scripts and tools with path validation.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultTimeout is the default execution timeout.
const DefaultTimeout = 30 * time.Second

// Result holds the output of a script/tool execution.
type Result struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// String formats the result for display.
func (r *Result) String() string {
	var b strings.Builder
	if r.Stdout != "" {
		b.WriteString(r.Stdout)
	}
	if r.Stderr != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[stderr]\n")
		b.WriteString(r.Stderr)
	}
	if r.ExitCode != 0 {
		fmt.Fprintf(&b, "\n[exit code: %d]", r.ExitCode)
	}
	return b.String()
}

// RunScript executes a shell script within the scripts directory.
func RunScript(dir, name string, args []string) (*Result, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, name)
	if !strings.HasSuffix(name, ".sh") {
		return nil, fmt.Errorf("not a shell script: %s", name)
	}

	return run(path, args)
}

// RunTool executes a Go tool via "go run" within the tools directory.
func RunTool(dir, name string, args []string) (*Result, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, name)
	if !strings.HasSuffix(name, ".go") {
		return nil, fmt.Errorf("not a Go file: %s", name)
	}

	// Run via "go run"
	goArgs := append([]string{"run", path}, args...)
	return run("go", goArgs)
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("empty name")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid name (path traversal): %s", name)
	}
	return nil
}

func run(cmdPath string, args []string) (*Result, error) {
	return runWithTimeout(cmdPath, args, DefaultTimeout)
}

func runWithTimeout(cmdPath string, args []string, timeout time.Duration) (*Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("execution timed out after %v", timeout)
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("execution error: %w", err)
		}
	}

	return result, nil
}
