package toolparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListTools(t *testing.T) {
	toolsDir := "/workspace/.sandbox/tools"
	if _, err := os.Stat(toolsDir); err != nil {
		t.Skip("Tools directory not found")
	}

	tools, err := ListTools(toolsDir)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	if len(tools) == 0 {
		t.Fatal("Expected at least one tool")
	}

	// search-history.go should be found
	found := false
	for _, tool := range tools {
		if tool.Name == "search-history.go" {
			found = true
			if tool.Description == "" {
				t.Error("Expected non-empty description for search-history.go")
			}
			if tool.Usage == "" {
				t.Error("Expected non-empty usage for search-history.go")
			}
			if len(tool.Examples) == 0 {
				t.Error("Expected at least one example for search-history.go")
			}
		}
	}
	if !found {
		t.Error("search-history.go not found in tools list")
	}
}

func TestGetDetailedInfoPathTraversal(t *testing.T) {
	_, err := GetDetailedInfo("/workspace/.sandbox/tools", "../etc/passwd")
	if err == nil {
		t.Error("Expected error for path traversal")
	}
}

func TestListToolsNonexistentDir(t *testing.T) {
	_, err := ListTools("/nonexistent/directory/xyz")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestGetDetailedInfoNonexistentTool(t *testing.T) {
	toolsDir := "/workspace/.sandbox/tools"
	if _, err := os.Stat(toolsDir); err != nil {
		t.Skip("Tools directory not found")
	}

	_, err := GetDetailedInfo(toolsDir, "does-not-exist.go")
	if err == nil {
		t.Error("Expected error for non-existent tool")
	}
}

func TestParseGoHeaderMinimalFile(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "minimal.go")
	os.WriteFile(tool, []byte("package main\n"), 0644)

	info, err := parseGoHeader(tool)
	if err != nil {
		t.Fatalf("parseGoHeader: %v", err)
	}
	if info.Name != "minimal.go" {
		t.Errorf("Name = %q, want %q", info.Name, "minimal.go")
	}
	// No comment header → empty description
	if info.Description != "" {
		t.Errorf("Description = %q, want empty", info.Description)
	}
}

func TestParseGoHeaderFullFormat(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "full.go")
	os.WriteFile(tool, []byte(`// A tool that does something
//
// Usage:
//   go run full.go [options]
//
// Examples:
//   go run full.go --verbose
//   go run full.go "search term"
package main
`), 0644)

	info, err := parseGoHeader(tool)
	if err != nil {
		t.Fatalf("parseGoHeader: %v", err)
	}
	if info.Description != "A tool that does something" {
		t.Errorf("Description = %q, want %q", info.Description, "A tool that does something")
	}
	if info.Usage == "" {
		t.Error("Expected non-empty usage")
	}
	if len(info.Examples) != 2 {
		t.Errorf("Examples count = %d, want 2", len(info.Examples))
	}
}

func TestListToolsSkipsNonGo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tool.go"), []byte("// a tool\npackage main\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0644)

	tools, err := ListTools(dir)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name == "readme.md" {
			t.Error("Non-.go file should be excluded")
		}
	}
}

func TestGetDetailedInfoSlash(t *testing.T) {
	_, err := GetDetailedInfo("/workspace/.sandbox/tools", "subdir/tool.go")
	if err == nil {
		t.Error("Expected error for path with slash")
	}
}
