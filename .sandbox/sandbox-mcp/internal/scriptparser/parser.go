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
	Name          string `json:"name"`
	DescriptionEN string `json:"description_en"`
	DescriptionJA string `json:"description_ja"`
	Environment   string `json:"environment"` // "host", "container", "any"
	Category      string `json:"category"`    // "utility", "test"
	Usage         string `json:"usage,omitempty"`
	Options       string `json:"options,omitempty"`
}

// Scripts that must run on host OS (from help.sh L40-41).
var hostOnly = map[string]bool{
	"copy-credentials.sh": true,
	"init-host-env.sh":    true,
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
//	Line 3: # English description
//	Line 4: # Japanese description
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
	for scanner.Scan() && lineNum < 4 {
		lineNum++
		line := scanner.Text()
		switch lineNum {
		case 3:
			info.DescriptionEN = stripComment(line)
		case 4:
			info.DescriptionJA = stripComment(line)
		}
	}

	return info, nil
}

// parseDetailedHeader reads more of the file to extract usage and options.
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

		// Detect usage section
		stripped := stripComment(line)
		if strings.HasPrefix(strings.ToLower(stripped), "usage:") || strings.HasPrefix(stripped, "使用法:") {
			inUsage = true
			usageLines = append(usageLines, stripped)
			continue
		}

		if inUsage {
			// End of usage section: non-comment line or empty comment
			if !strings.HasPrefix(line, "#") {
				inUsage = false
				continue
			}
			content := stripComment(line)
			if content == "" {
				inUsage = false
				continue
			}
			usageLines = append(usageLines, content)
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
