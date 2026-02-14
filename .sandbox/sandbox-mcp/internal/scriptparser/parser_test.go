package scriptparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListScripts(t *testing.T) {
	scriptsDir := "/workspace/.sandbox/scripts"
	if _, err := os.Stat(scriptsDir); err != nil {
		t.Skip("Scripts directory not found")
	}

	scripts, err := ListScripts(scriptsDir)
	if err != nil {
		t.Fatalf("ListScripts: %v", err)
	}

	if len(scripts) == 0 {
		t.Fatal("Expected at least one script")
	}

	// Verify known scripts exist
	found := map[string]bool{}
	for _, s := range scripts {
		found[s.Name] = true
	}

	expected := []string{"validate-secrets.sh", "check-secret-sync.sh"}
	for _, name := range expected {
		if !found[name] {
			t.Errorf("Expected script %s not found in list", name)
		}
	}

	// Verify excluded scripts
	excluded := []string{"_startup_common.sh", "help.sh"}
	for _, name := range excluded {
		if found[name] {
			t.Errorf("Script %s should be excluded", name)
		}
	}
}

func TestListScriptsEnvironment(t *testing.T) {
	scriptsDir := "/workspace/.sandbox/scripts"
	if _, err := os.Stat(scriptsDir); err != nil {
		t.Skip("Scripts directory not found")
	}

	scripts, err := ListScripts(scriptsDir)
	if err != nil {
		t.Fatalf("ListScripts: %v", err)
	}

	for _, s := range scripts {
		switch s.Name {
		case "init-host-env.sh":
			if s.Environment != "host" {
				t.Errorf("%s: environment = %q, want %q", s.Name, s.Environment, "host")
			}
		case "validate-secrets.sh", "sync-secrets.sh", "sync-compose-secrets.sh":
			if s.Environment != "container" {
				t.Errorf("%s: environment = %q, want %q", s.Name, s.Environment, "container")
			}
		}
	}
}

func TestListScriptsCategory(t *testing.T) {
	scriptsDir := "/workspace/.sandbox/scripts"
	if _, err := os.Stat(scriptsDir); err != nil {
		t.Skip("Scripts directory not found")
	}

	scripts, err := ListScripts(scriptsDir)
	if err != nil {
		t.Fatalf("ListScripts: %v", err)
	}

	for _, s := range scripts {
		if s.Name == "test-validate-secrets.sh" {
			if s.Category != "test" {
				t.Errorf("%s: category = %q, want %q", s.Name, s.Category, "test")
			}
		}
		if s.Name == "validate-secrets.sh" {
			if s.Category != "utility" {
				t.Errorf("%s: category = %q, want %q", s.Name, s.Category, "utility")
			}
		}
	}
}

func TestGetDetailedInfo(t *testing.T) {
	scriptsDir := "/workspace/.sandbox/scripts"
	if _, err := os.Stat(scriptsDir); err != nil {
		t.Skip("Scripts directory not found")
	}

	info, err := GetDetailedInfo(scriptsDir, "validate-secrets.sh")
	if err != nil {
		t.Fatalf("GetDetailedInfo: %v", err)
	}

	if info.Description == "" {
		t.Error("Expected non-empty description")
	}
	if info.Environment != "container" {
		t.Errorf("environment = %q, want %q", info.Environment, "container")
	}
}

func TestGetDetailedInfoPathTraversal(t *testing.T) {
	_, err := GetDetailedInfo("/workspace/.sandbox/scripts", "../../../etc/passwd")
	if err == nil {
		t.Error("Expected error for path traversal")
	}

	_, err = GetDetailedInfo("/workspace/.sandbox/scripts", "foo/bar.sh")
	if err == nil {
		t.Error("Expected error for path with slash")
	}
}

func TestIsHostOnly(t *testing.T) {
	if !IsHostOnly("init-host-env.sh") {
		t.Error("init-host-env.sh should be host-only")
	}
	if IsHostOnly("validate-secrets.sh") {
		t.Error("validate-secrets.sh should not be host-only")
	}
}

func TestParseHeaderFormat(t *testing.T) {
	// Test that merge-claude-settings.sh now has correct format
	scriptsDir := "/workspace/.sandbox/scripts"
	if _, err := os.Stat(scriptsDir); err != nil {
		t.Skip("Scripts directory not found")
	}

	info, err := parseHeader(filepath.Join(scriptsDir, "merge-claude-settings.sh"))
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}

	if info.Description == "" {
		t.Error("Expected non-empty description for merge-claude-settings.sh")
	}
	// Description should be English (before # --- separator)
	if info.Description == "サブプロジェクトの .claude/settings.json を workspace 直下にマージ" {
		t.Error("Description appears to be Japanese - should be English before # --- separator")
	}
}

func TestListScriptsNonexistentDir(t *testing.T) {
	_, err := ListScripts("/nonexistent/directory/xyz")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestGetDetailedInfoNonexistentScript(t *testing.T) {
	scriptsDir := "/workspace/.sandbox/scripts"
	if _, err := os.Stat(scriptsDir); err != nil {
		t.Skip("Scripts directory not found")
	}

	_, err := GetDetailedInfo(scriptsDir, "does-not-exist.sh")
	if err == nil {
		t.Error("Expected error for non-existent script")
	}
}

func TestParseHeaderMinimalFile(t *testing.T) {
	dir := t.TempDir()
	// File with only shebang line (less than 3 lines)
	script := filepath.Join(dir, "minimal.sh")
	os.WriteFile(script, []byte("#!/bin/bash\n"), 0755)

	info, err := parseHeader(script)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}
	if info.Name != "minimal.sh" {
		t.Errorf("Name = %q, want %q", info.Name, "minimal.sh")
	}
	// Description should be empty for minimal file
	if info.Description != "" {
		t.Errorf("Description = %q, want empty", info.Description)
	}
}

func TestParseHeaderEmptyFile(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "empty.sh")
	os.WriteFile(script, []byte(""), 0755)

	info, err := parseHeader(script)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}
	if info.Name != "empty.sh" {
		t.Errorf("Name = %q, want %q", info.Name, "empty.sh")
	}
}

func TestListScriptsSkipsNonSh(t *testing.T) {
	dir := t.TempDir()
	// Create a .sh file and a non-.sh file
	os.WriteFile(filepath.Join(dir, "good.sh"), []byte("#!/bin/bash\n# good.sh\n# English\n# Japanese\n"), 0755)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a script"), 0644)

	scripts, err := ListScripts(dir)
	if err != nil {
		t.Fatalf("ListScripts: %v", err)
	}
	for _, s := range scripts {
		if s.Name == "readme.txt" {
			t.Error("Non-.sh file should be excluded")
		}
	}
}

func TestListScriptsSkipsUnderscoreAndHelp(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "_common.sh"), []byte("#!/bin/bash\n# lib\n# EN\n# JA\n"), 0755)
	os.WriteFile(filepath.Join(dir, "help.sh"), []byte("#!/bin/bash\n# help\n# EN\n# JA\n"), 0755)
	os.WriteFile(filepath.Join(dir, "valid.sh"), []byte("#!/bin/bash\n# valid.sh\n# EN\n# JA\n"), 0755)

	scripts, err := ListScripts(dir)
	if err != nil {
		t.Fatalf("ListScripts: %v", err)
	}
	for _, s := range scripts {
		if s.Name == "_common.sh" || s.Name == "help.sh" {
			t.Errorf("Script %s should be excluded", s.Name)
		}
	}
}

// TestParseHeaderStopsAtSeparator verifies that parsing stops at # --- separator.
// This aligns with Go tools behavior where // --- marks the end of parsed content.
func TestParseHeaderStopsAtSeparator(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "test.sh")

	// Script with # --- separator followed by Japanese docs
	content := `#!/bin/bash
# test.sh
# Validates secret hiding mechanism
# ---
# シークレット隠蔽機構の検証
# この行も無視されるべき
`
	os.WriteFile(script, []byte(content), 0755)

	info, err := parseHeader(script)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}

	// Description should only contain the English part (before # ---)
	if info.Description != "Validates secret hiding mechanism" {
		t.Errorf("Description = %q, want %q", info.Description, "Validates secret hiding mechanism")
	}

	// Should NOT contain Japanese text after # ---
	if strings.Contains(info.Description, "シークレット") {
		t.Error("Description should not contain Japanese text after # --- separator")
	}
	if strings.Contains(info.Description, "この行も無視") {
		t.Error("Description should not contain text after # --- separator")
	}
}

// TestParseHeaderWithoutSeparator verifies backward compatibility.
// Scripts without # --- should still parse correctly.
func TestParseHeaderWithoutSeparator(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "legacy.sh")

	// Old format without separator
	content := `#!/bin/bash
# legacy.sh
# Legacy script description
`
	os.WriteFile(script, []byte(content), 0755)

	info, err := parseHeader(script)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}

	if info.Description != "Legacy script description" {
		t.Errorf("Description = %q, want %q", info.Description, "Legacy script description")
	}
}

// TestParseHeaderSeparatorVariations tests edge cases with # --- separator.
func TestParseHeaderSeparatorVariations(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantDesc    string
		wantNoMatch string // text that should NOT appear in description
	}{
		{
			name: "separator with extra text",
			content: `#!/bin/bash
# test.sh
# English description
# --- 日本語 ---
# 日本語の説明
`,
			wantDesc:    "English description",
			wantNoMatch: "日本語",
		},
		{
			name: "multiple separators",
			content: `#!/bin/bash
# test.sh
# First description
# ---
# This should be ignored
# ---
# This too
`,
			wantDesc:    "First description",
			wantNoMatch: "should be ignored",
		},
		{
			name: "separator on line 3",
			content: `#!/bin/bash
# test.sh
# ---
# Everything after first separator ignored
`,
			wantDesc:    "",
			wantNoMatch: "Everything",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			script := filepath.Join(dir, "test.sh")
			os.WriteFile(script, []byte(tt.content), 0755)

			info, err := parseHeader(script)
			if err != nil {
				t.Fatalf("parseHeader: %v", err)
			}

			if info.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", info.Description, tt.wantDesc)
			}

			if tt.wantNoMatch != "" && strings.Contains(info.Description, tt.wantNoMatch) {
				t.Errorf("Description should not contain %q, but got %q", tt.wantNoMatch, info.Description)
			}
		})
	}
}

// TestParseDetailedHeaderUsageBeforeSeparator verifies that Usage sections before # --- are parsed.
// This aligns with Go tools where Usage/Examples come before // --- separator.
func TestParseDetailedHeaderUsageBeforeSeparator(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "test.sh")

	// Script with Usage BEFORE # --- separator (should be parsed)
	content := `#!/bin/bash
# test.sh
# English description
#
# Usage:
#   test.sh [options]
#   test.sh --verbose
#
# ---
# 日本語の説明
#
# 使用法:
#   test.sh [オプション]
`
	os.WriteFile(script, []byte(content), 0755)

	info, err := parseDetailedHeader(script)
	if err != nil {
		t.Fatalf("parseDetailedHeader: %v", err)
	}

	// Usage should be parsed (it's before # ---)
	if info.Usage == "" {
		t.Error("Expected Usage to be parsed when it appears before # --- separator")
	}

	// Should contain English usage
	if !strings.Contains(info.Usage, "test.sh [options]") {
		t.Errorf("Usage should contain English usage, got: %q", info.Usage)
	}

	// Should NOT contain Japanese usage (after # ---)
	if strings.Contains(info.Usage, "使用法") {
		t.Errorf("Usage should not contain Japanese usage after # --- separator, got: %q", info.Usage)
	}
	if strings.Contains(info.Usage, "オプション") {
		t.Errorf("Usage should not contain Japanese text after # --- separator, got: %q", info.Usage)
	}
}

// TestParseDetailedHeaderUsageAfterSeparator verifies that Usage sections after # --- are NOT parsed.
func TestParseDetailedHeaderUsageAfterSeparator(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "test.sh")

	// Script with Usage AFTER # --- separator (should NOT be parsed)
	content := `#!/bin/bash
# test.sh
# English description
# ---
# 日本語の説明
#
# Usage:
#   test.sh [options]
`
	os.WriteFile(script, []byte(content), 0755)

	info, err := parseDetailedHeader(script)
	if err != nil {
		t.Fatalf("parseDetailedHeader: %v", err)
	}

	// Usage should NOT be parsed (it's after # ---)
	if info.Usage != "" {
		t.Errorf("Usage should not be parsed when it appears after # --- separator, got: %q", info.Usage)
	}
}
