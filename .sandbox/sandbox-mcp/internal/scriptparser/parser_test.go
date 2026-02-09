package scriptparser

import (
	"os"
	"path/filepath"
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

	expected := []string{"validate-secrets.sh", "copy-credentials.sh", "check-secret-sync.sh"}
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
		case "copy-credentials.sh", "init-host-env.sh":
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

	if info.DescriptionEN == "" {
		t.Error("Expected non-empty English description")
	}
	if info.DescriptionJA == "" {
		t.Error("Expected non-empty Japanese description")
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
	if !IsHostOnly("copy-credentials.sh") {
		t.Error("copy-credentials.sh should be host-only")
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

	if info.DescriptionEN == "" {
		t.Error("Expected non-empty English description for merge-claude-settings.sh")
	}
	// After Phase 0 fix, EN description should be English
	if info.DescriptionEN == "サブプロジェクトの .claude/settings.json を workspace 直下にマージ" {
		t.Error("EN description appears to be Japanese - header format may not be fixed")
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
	// File with only shebang line (less than 4 lines)
	script := filepath.Join(dir, "minimal.sh")
	os.WriteFile(script, []byte("#!/bin/bash\n"), 0755)

	info, err := parseHeader(script)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}
	if info.Name != "minimal.sh" {
		t.Errorf("Name = %q, want %q", info.Name, "minimal.sh")
	}
	// Descriptions should be empty for minimal file
	if info.DescriptionEN != "" {
		t.Errorf("DescriptionEN = %q, want empty", info.DescriptionEN)
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
