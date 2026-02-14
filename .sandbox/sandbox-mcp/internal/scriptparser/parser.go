// Package scriptparser parses .sandbox/scripts/ shell script headers.
package scriptparser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScriptInfo holds parsed metadata about a script.
type ScriptInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Environment string `json:"environment"` // "host", "container", "any"
	Category    string `json:"category"`    // "utility", "test"
	Usage       string `json:"usage,omitempty"`
	Options     string `json:"options,omitempty"`
}

// Scripts that must run on host OS (from help.sh L40-41).
// Note: copy-credentials.sh moved to .sandbox/host-tools/ (DockMCP host tools)
var hostOnly = map[string]bool{
	"init-host-env.sh": true,
}

// Scripts that must run in container (from help.sh L42-43).
var containerOnly = map[string]bool{
	"sync-secrets.sh":         true,
	"validate-secrets.sh":     true,
	"sync-compose-secrets.sh": true,
}

// IsHostOnly returns true if the script can only run on the host OS.
func IsHostOnly(name string) bool {
	return hostOnly[name]
}

// ListScripts returns metadata for all scripts in the directory.
func ListScripts(dir string) ([]ScriptInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var scripts []ScriptInfo
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sh") {
			continue
		}
		// Skip libraries (underscore prefix) and help.sh
		if strings.HasPrefix(name, "_") || name == "help.sh" {
			continue
		}

		info, err := parseHeader(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		scripts = append(scripts, info)
	}
	return scripts, nil
}

// GetDetailedInfo returns full info including usage and options.
func GetDetailedInfo(dir, name string) (ScriptInfo, error) {
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return ScriptInfo{}, fmt.Errorf("invalid script name: %s", name)
	}

	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		return ScriptInfo{}, fmt.Errorf("script not found: %s", name)
	}

	return parseDetailedHeader(path)
}

// parseHeader extracts basic info from script header lines.
// Expected format:
//
//	Line 1: #!/bin/bash
//	Line 2: # filename.sh
//	Line 3+: # Description text (until # --- separator or end of comments)
//
// Parsing stops at:
//   - # --- separator (content after this is ignored, similar to Go tools' // ---)
//   - Non-comment line
//   - End of file
func parseHeader(path string) (ScriptInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return ScriptInfo{}, err
	}
	defer f.Close()

	name := filepath.Base(path)
	info := ScriptInfo{
		Name:        name,
		Environment: classifyEnvironment(name),
		Category:    classifyCategory(name),
	}

	scanner := bufio.NewScanner(f)
	lineNum := 0
	var descLines []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip shebang and filename lines
		if lineNum <= 2 {
			continue
		}

		// Stop at non-comment lines
		if !strings.HasPrefix(line, "#") {
			break
		}

		content := stripComment(line)

		// Stop at # --- separator
		if strings.HasPrefix(content, "---") {
			break
		}

		// Collect description lines (skip empty lines)
		if content != "" {
			descLines = append(descLines, content)
		}
	}

	// Join description lines with space
	if len(descLines) > 0 {
		info.Description = strings.Join(descLines, " ")
	}

	return info, nil
}

// parseDetailedHeader reads more of the file to extract usage and options.
// Parsing stops at # --- separator, aligning with Go tools' // --- pattern.
func parseDetailedHeader(path string) (ScriptInfo, error) {
	info, err := parseHeader(path)
	if err != nil {
		return info, err
	}

	f, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var usageLines []string
	inUsage := false

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum > 50 { // Only scan first 50 lines for header
			break
		}
		line := scanner.Text()

		// Stop at non-comment lines
		if !strings.HasPrefix(line, "#") {
			break
		}

		stripped := stripComment(line)

		// Stop at # --- separator (aligns with Go tools' // --- pattern)
		if strings.HasPrefix(stripped, "---") {
			break
		}

		// Detect usage section
		if strings.HasPrefix(strings.ToLower(stripped), "usage:") || strings.HasPrefix(stripped, "使用法:") {
			inUsage = true
			usageLines = append(usageLines, stripped)
			continue
		}

		if inUsage {
			// End of usage section: empty comment
			if stripped == "" {
				inUsage = false
				continue
			}
			usageLines = append(usageLines, stripped)
		}
	}

	if len(usageLines) > 0 {
		info.Usage = strings.Join(usageLines, "\n")
	}

	return info, nil
}

func stripComment(line string) string {
	if strings.HasPrefix(line, "#") {
		return strings.TrimSpace(strings.TrimPrefix(line, "#"))
	}
	return strings.TrimSpace(line)
}

func classifyEnvironment(name string) string {
	if hostOnly[name] {
		return "host"
	}
	if containerOnly[name] {
		return "container"
	}
	return "any"
}

func classifyCategory(name string) string {
	if strings.HasPrefix(name, "test-") {
		return "test"
	}
	return "utility"
}
