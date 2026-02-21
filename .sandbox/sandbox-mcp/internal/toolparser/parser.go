// Package toolparser parses .sandbox/tools/ Go file headers.
package toolparser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ToolInfo holds parsed metadata about a tool.
type ToolInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Usage       string   `json:"usage,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// ListTools returns metadata for all tools in the directory.
func ListTools(dir string) ([]ToolInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var tools []ToolInfo
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		info, err := parseGoHeader(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		tools = append(tools, info)
	}
	return tools, nil
}

// GetDetailedInfo returns full info for a specific tool.
func GetDetailedInfo(dir, name string) (ToolInfo, error) {
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return ToolInfo{}, fmt.Errorf("invalid tool name: %s", name)
	}

	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		return ToolInfo{}, fmt.Errorf("tool not found: %s", name)
	}

	return parseGoHeader(path)
}

// parseGoHeader extracts metadata from Go file // comments.
// Expected format:
//
//	// description - short description
//	//
//	// Usage:
//	//   go run .sandbox/tools/file.go [options] <args>
//	//
//	// Examples:
//	//   go run .sandbox/tools/file.go "pattern"
func parseGoHeader(path string) (ToolInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return ToolInfo{}, err
	}
	defer f.Close()

	name := filepath.Base(path)
	info := ToolInfo{Name: name}

	scanner := bufio.NewScanner(f)
	var usageLines []string
	var examples []string
	section := "" // "", "usage", "examples"

	for scanner.Scan() {
		line := scanner.Text()

		// Stop at package declaration
		if strings.HasPrefix(line, "package ") {
			break
		}

		if !strings.HasPrefix(line, "//") {
			continue
		}

		content := strings.TrimSpace(strings.TrimPrefix(line, "//"))

		// Stop at separator line (e.g. "// ---" marks start of localized/non-parsed section)
		if strings.HasPrefix(content, "---") {
			break
		}

		// First non-empty comment line is the description
		if info.Description == "" && content != "" {
			info.Description = content
			continue
		}

		// Detect sections
		if strings.HasPrefix(content, "Usage:") {
			section = "usage"
			continue
		}
		if strings.HasPrefix(content, "Examples:") {
			section = "examples"
			continue
		}

		// Empty comment line resets section
		if content == "" {
			if section != "" && section != "usage" && section != "examples" {
				section = ""
			}
			// Allow blank lines within sections
			continue
		}

		switch section {
		case "usage":
			usageLines = append(usageLines, content)
		case "examples":
			examples = append(examples, content)
		}
	}

	if len(usageLines) > 0 {
		info.Usage = strings.Join(usageLines, "\n")
	}
	info.Examples = examples

	return info, nil
}
