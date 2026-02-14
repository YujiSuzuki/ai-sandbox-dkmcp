package hosttools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseGoHeader verifies that Go header comments are parsed correctly,
// including description, usage, and examples sections, and stops at the "---" separator.
//
// TestParseGoHeaderは、Go headerのコメントが正しくパースされることを確認します。
// description、usage、examplesの各セクション、および "---" 区切り文字での停止を検証します。
func TestParseGoHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-tool.go")
	content := `// Short description of the tool
//
// Usage:
//   go run test-tool.go [options]
//
// Examples:
//   go run test-tool.go "hello"
//   go run test-tool.go -v "world"
//
// ---
// Japanese description (not parsed)
package main
`
	os.WriteFile(path, []byte(content), 0644)

	info, err := parseGoHeader(path)
	if err != nil {
		t.Fatalf("parseGoHeader error: %v", err)
	}

	if info.Name != "test-tool.go" {
		t.Errorf("Name = %q, want test-tool.go", info.Name)
	}
	if info.Description != "Short description of the tool" {
		t.Errorf("Description = %q, want 'Short description of the tool'", info.Description)
	}
	if info.Usage != "go run test-tool.go [options]" {
		t.Errorf("Usage = %q, want 'go run test-tool.go [options]'", info.Usage)
	}
	if len(info.Examples) != 2 {
		t.Errorf("Examples length = %d, want 2", len(info.Examples))
	}
	if info.Extension != ".go" {
		t.Errorf("Extension = %q, want .go", info.Extension)
	}
}

// TestParseShellHeader verifies that shell script header comments are parsed correctly,
// extracting the description and stopping at the "---" separator.
//
// TestParseShellHeaderは、シェルスクリプトのheaderコメントが正しくパースされることを確認します。
// descriptionの抽出と "---" 区切り文字での停止を検証します。
func TestParseShellHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-tool.sh")
	content := `#!/bin/bash
# my-tool.sh
# Tool that does something useful
# ---
set -e
echo "hello"
`
	os.WriteFile(path, []byte(content), 0644)

	info, err := parseShellHeader(path)
	if err != nil {
		t.Fatalf("parseShellHeader error: %v", err)
	}

	if info.Name != "my-tool.sh" {
		t.Errorf("Name = %q, want my-tool.sh", info.Name)
	}
	if info.Description != "Tool that does something useful" {
		t.Errorf("Description = %q, want 'Tool that does something useful'", info.Description)
	}
	if info.Extension != ".sh" {
		t.Errorf("Extension = %q, want .sh", info.Extension)
	}
}

// TestParsePythonHeader verifies that Python header comments are parsed correctly,
// extracting the description while ignoring shebang and encoding declarations.
//
// TestParsePythonHeaderは、Pythonのheaderコメントが正しくパースされることを確認します。
// shebangやエンコーディング宣言を無視しつつ、descriptionを抽出できることを検証します。
func TestParsePythonHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool.py")
	content := `#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# Python tool for data processing
import sys
`
	os.WriteFile(path, []byte(content), 0644)

	info, err := parsePythonHeader(path)
	if err != nil {
		t.Fatalf("parsePythonHeader error: %v", err)
	}

	if info.Description != "Python tool for data processing" {
		t.Errorf("Description = %q, want 'Python tool for data processing'", info.Description)
	}
	if info.Extension != ".py" {
		t.Errorf("Extension = %q, want .py", info.Extension)
	}
}

// TestListTools_FiltersExtensions verifies that ListTools only includes files
// with allowed extensions and excludes underscore-prefixed files.
//
// TestListTools_FiltersExtensionsは、ListToolsが許可された拡張子のファイルのみを含み、
// アンダースコアで始まるファイルを除外することを確認します。
func TestListTools_FiltersExtensions(t *testing.T) {
	dir := t.TempDir()

	// Create files with different extensions
	files := map[string]string{
		"tool1.go": "// Go tool\npackage main\n",
		"tool2.sh": "#!/bin/bash\n# tool2.sh\n# Shell tool\n",
		"tool3.py": "# Python tool\n",
		"tool4.rb": "# Ruby tool\n",                   // not allowed
		"_lib.go":  "// Library helper\npackage lib\n", // underscore prefix
	}
	for name, content := range files {
		os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	}

	// Only .go and .sh allowed
	tools, err := ListTools(dir, []string{".go", ".sh"})
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("ListTools returned %d tools, want 2 (tool1.go and tool2.sh)", len(tools))
	}

	// Verify .rb and _lib.go are filtered out
	for _, tool := range tools {
		if tool.Extension == ".rb" {
			t.Error("ListTools should not include .rb files")
		}
		if tool.Name == "_lib.go" {
			t.Error("ListTools should not include underscore-prefixed files")
		}
	}
}

// TestGetToolInfo_PathTraversal verifies that GetToolInfo rejects path traversal
// attempts and paths containing slashes.
//
// TestGetToolInfo_PathTraversalは、GetToolInfoがパストラバーサル攻撃や
// スラッシュを含むパスを拒否することを確認します。
func TestGetToolInfo_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	_, err := GetToolInfo(dir, "../../../etc/passwd", []string{".sh"})
	if err == nil {
		t.Error("GetToolInfo should reject path traversal")
	}

	_, err = GetToolInfo(dir, "sub/tool.go", []string{".go"})
	if err == nil {
		t.Error("GetToolInfo should reject paths with /")
	}
}

// TestGetToolInfo_ExtensionNotAllowed verifies that GetToolInfo rejects files
// with extensions not in the allowed list.
//
// TestGetToolInfo_ExtensionNotAllowedは、GetToolInfoが許可リストに含まれない
// 拡張子のファイルを拒否することを確認します。
func TestGetToolInfo_ExtensionNotAllowed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tool.rb"), []byte("# Ruby\n"), 0644)

	_, err := GetToolInfo(dir, "tool.rb", []string{".go", ".sh"})
	if err == nil {
		t.Error("GetToolInfo should reject extensions not in allowedExtensions")
	}
}

// TestValidateName verifies that validateName correctly accepts valid filenames
// and rejects empty names, path traversal attempts, and paths with slashes.
//
// TestValidateNameは、validateNameが正しいファイル名を受け入れ、
// 空の名前、パストラバーサル、スラッシュを含むパスを拒否することを確認します。
func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "tool.go", false},
		{"empty name", "", true},
		{"path traversal", "../tool.go", true},
		{"slash in name", "sub/tool.go", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
