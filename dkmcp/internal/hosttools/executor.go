package hosttools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Result holds the output of a tool/command execution.
// Resultはツール/コマンド実行の出力を保持します。
type Result struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// String formats the result for display.
// Stringは表示用に結果をフォーマットします。
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

// RunTool executes a tool file with the given arguments and timeout.
// The tool is dispatched based on file extension:
//   - .go  → go run <path> [args...]
//   - .sh  → bash <path> [args...]
//   - .py  → python3 <path> [args...]
//
// The working directory is set to workDir. If workDir is empty, the tool's
// directory is used as a fallback.
//
// RunToolは指定された引数とタイムアウトでツールファイルを実行します。
// 作業ディレクトリはworkDirに設定されます。workDirが空の場合、
// ツールのディレクトリがフォールバックとして使用されます。
func RunTool(dir, name string, args []string, timeout time.Duration, workDir string) (*Result, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	ext := getExtension(name)
	path := dir + "/" + name

	var cmdPath string
	var cmdArgs []string

	switch ext {
	case ".go":
		cmdPath = "go"
		cmdArgs = append([]string{"run", path}, args...)
	case ".sh":
		cmdPath = "bash"
		cmdArgs = append([]string{path}, args...)
	case ".py":
		cmdPath = "python3"
		cmdArgs = append([]string{path}, args...)
	default:
		return nil, fmt.Errorf("unsupported extension: %s", ext)
	}

	if workDir == "" {
		workDir = dir
	}
	return runWithTimeout(cmdPath, cmdArgs, workDir, timeout)
}

// ExecHostCommand executes a host CLI command string with the given
// working directory and timeout.
//
// ExecHostCommandは指定された作業ディレクトリとタイムアウトで
// ホストCLIコマンド文字列を実行します。
func ExecHostCommand(command string, workspaceRoot string, timeout time.Duration) (*Result, error) {
	args, err := parseCommandArgs(command)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return runWithTimeout(args[0], args[1:], workspaceRoot, timeout)
}

// runWithTimeout runs a command with the specified timeout and working directory.
// runWithTimeoutは指定されたタイムアウトと作業ディレクトリでコマンドを実行します。
func runWithTimeout(cmdPath string, args []string, workDir string, timeout time.Duration) (*Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
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

// getExtension returns the file extension including the dot.
func getExtension(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i:]
		}
	}
	return ""
}

// parseCommandArgs splits a command string into arguments.
// Handles quoted strings (single and double quotes).
//
// parseCommandArgsはコマンド文字列を引数に分割します。
// クォート文字列（シングルおよびダブルクォート）を処理します。
func parseCommandArgs(command string) ([]string, error) {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, ch := range command {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' && !inSingle {
			escaped = true
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}

		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if ch == ' ' && !inSingle && !inDouble {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("unclosed quote in command: %s", command)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}
