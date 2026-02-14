package hosttools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

func TestManager_Disabled(t *testing.T) {
	cfg := &config.HostToolsConfig{Enabled: false}
	m := NewManager(cfg, "/tmp")

	if m.IsEnabled() {
		t.Error("Manager.IsEnabled() should be false when disabled")
	}

	_, err := m.ListTools()
	if err == nil {
		t.Error("ListTools should error when disabled")
	}

	_, err = m.GetToolInfo("tool.go")
	if err == nil {
		t.Error("GetToolInfo should error when disabled")
	}

	_, err = m.RunTool("tool.go", nil)
	if err == nil {
		t.Error("RunTool should error when disabled")
	}
}

func TestManager_NilConfig(t *testing.T) {
	m := NewManager(nil, "/tmp")

	if m.IsEnabled() {
		t.Error("Manager.IsEnabled() should be false with nil config")
	}
}

func TestManager_ListTools(t *testing.T) {
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	os.MkdirAll(toolsDir, 0755)

	// Create test tools
	os.WriteFile(filepath.Join(toolsDir, "tool1.sh"), []byte("#!/bin/bash\n# tool1.sh\n# First tool\n"), 0755)
	os.WriteFile(filepath.Join(toolsDir, "tool2.go"), []byte("// Second tool\npackage main\n"), 0644)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		AllowedExtensions: []string{".sh", ".go"},
		Timeout:           30,
	}
	m := NewManager(cfg, dir)

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("ListTools returned %d tools, want 2", len(tools))
	}
}

func TestManager_GetToolInfo(t *testing.T) {
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	os.MkdirAll(toolsDir, 0755)

	os.WriteFile(filepath.Join(toolsDir, "mytool.sh"), []byte("#!/bin/bash\n# mytool.sh\n# My useful tool\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, dir)

	info, err := m.GetToolInfo("mytool.sh")
	if err != nil {
		t.Fatalf("GetToolInfo error: %v", err)
	}

	if info.Name != "mytool.sh" {
		t.Errorf("Name = %q, want mytool.sh", info.Name)
	}
	if info.Description != "My useful tool" {
		t.Errorf("Description = %q, want 'My useful tool'", info.Description)
	}
}

func TestManager_GetToolInfo_NotFound(t *testing.T) {
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	os.MkdirAll(toolsDir, 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, dir)

	_, err := m.GetToolInfo("nonexistent.sh")
	if err == nil {
		t.Error("GetToolInfo should error for nonexistent tool")
	}
}

func TestManager_RunTool(t *testing.T) {
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	os.MkdirAll(toolsDir, 0755)

	os.WriteFile(filepath.Join(toolsDir, "greet.sh"), []byte("#!/bin/bash\n# greet.sh\n# Greet tool\necho \"Hello $1\"\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, dir)

	result, err := m.RunTool("greet.sh", []string{"World"})
	if err != nil {
		t.Fatalf("RunTool error: %v", err)
	}

	if result.Stdout != "Hello World\n" {
		t.Errorf("Stdout = %q, want 'Hello World\\n'", result.Stdout)
	}
}

func TestManager_RunTool_NotFound(t *testing.T) {
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	os.MkdirAll(toolsDir, 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, dir)

	_, err := m.RunTool("nonexistent.sh", nil)
	if err == nil {
		t.Error("RunTool should error for nonexistent tool")
	}
}

func TestManager_MultipleDirectories(t *testing.T) {
	dir := t.TempDir()
	dir1 := filepath.Join(dir, "tools1")
	dir2 := filepath.Join(dir, "tools2")
	os.MkdirAll(dir1, 0755)
	os.MkdirAll(dir2, 0755)

	os.WriteFile(filepath.Join(dir1, "a.sh"), []byte("#!/bin/bash\n# a.sh\n# Tool A\n"), 0755)
	os.WriteFile(filepath.Join(dir2, "b.sh"), []byte("#!/bin/bash\n# b.sh\n# Tool B\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools1", "tools2"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, dir)

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("ListTools returned %d tools, want 2 (from two directories)", len(tools))
	}
}

func TestManager_NonexistentDirectory(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"nonexistent-dir"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, dir)

	// Should return empty list, not error (directory is skipped)
	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("ListTools should return empty list for nonexistent directory, got %d", len(tools))
	}
}

// --- Secure mode tests ---

func TestManager_SecureMode_ListTools(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create tools in project-specific approved directory
	projectID := ProjectID(workspaceDir)
	approvedDir := filepath.Join(approvedBaseDir, projectID)
	os.MkdirAll(approvedDir, 0755)
	os.WriteFile(filepath.Join(approvedDir, "approved.sh"),
		[]byte("#!/bin/bash\n# approved.sh\n# An approved tool\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"staging"},
		Common:            false,
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, workspaceDir)

	if !m.IsSecureMode() {
		t.Error("Manager should be in secure mode when ApprovedDir is set")
	}

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools) != 1 {
		t.Errorf("ListTools returned %d tools, want 1", len(tools))
	}
	if len(tools) > 0 && tools[0].Name != "approved.sh" {
		t.Errorf("tool name = %q, want approved.sh", tools[0].Name)
	}
}

func TestManager_SecureMode_WithCommon(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create project-specific tool
	projectID := ProjectID(workspaceDir)
	projectDir := filepath.Join(approvedBaseDir, projectID)
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, "project-tool.sh"),
		[]byte("#!/bin/bash\n# project-tool.sh\n# Project tool\n"), 0755)

	// Create common tool
	commonDir := filepath.Join(approvedBaseDir, "_common")
	os.MkdirAll(commonDir, 0755)
	os.WriteFile(filepath.Join(commonDir, "common-tool.sh"),
		[]byte("#!/bin/bash\n# common-tool.sh\n# Common tool\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		Common:            true,
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, workspaceDir)

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("ListTools returned %d tools, want 2 (project + common)", len(tools))
	}
}

func TestManager_SecureMode_ProjectOverridesCommon(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create same-named tool in both project and common
	projectID := ProjectID(workspaceDir)
	projectDir := filepath.Join(approvedBaseDir, projectID)
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, "tool.sh"),
		[]byte("#!/bin/bash\n# tool.sh\n# Project version\n"), 0755)

	commonDir := filepath.Join(approvedBaseDir, "_common")
	os.MkdirAll(commonDir, 0755)
	os.WriteFile(filepath.Join(commonDir, "tool.sh"),
		[]byte("#!/bin/bash\n# tool.sh\n# Common version\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		Common:            true,
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, workspaceDir)

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	// Should only return 1 tool (project takes priority, deduplicates)
	if len(tools) != 1 {
		t.Errorf("ListTools returned %d tools, want 1 (project overrides common)", len(tools))
	}
	if len(tools) > 0 && tools[0].Description != "Project version" {
		t.Errorf("tool description = %q, want 'Project version' (project should override common)", tools[0].Description)
	}
}

func TestManager_SecureMode_StagingNotExecuted(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create tool ONLY in staging (not approved)
	stagingDir := filepath.Join(workspaceDir, "host-tools")
	os.MkdirAll(stagingDir, 0755)
	os.WriteFile(filepath.Join(stagingDir, "unapproved.sh"),
		[]byte("#!/bin/bash\n# unapproved.sh\n# Unapproved tool\necho dangerous\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"host-tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, workspaceDir)

	// ListTools should not show staging tools
	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("ListTools returned %d tools, want 0 (staging tools should not be listed)", len(tools))
	}

	// RunTool should not find staging tools
	_, err = m.RunTool("unapproved.sh", nil)
	if err == nil {
		t.Error("RunTool should fail for unapproved tool in staging")
	}
}

// --- Dev mode tests ---

func TestManager_DevMode_StagingOverridesApproved(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create tool in approved with old content
	projectID := ProjectID(workspaceDir)
	approvedDir := filepath.Join(approvedBaseDir, projectID)
	os.MkdirAll(approvedDir, 0755)
	os.WriteFile(filepath.Join(approvedDir, "tool.sh"),
		[]byte("#!/bin/bash\n# tool.sh\n# Approved version\necho approved\n"), 0755)

	// Create same tool in staging with new content
	stagingDir := filepath.Join(workspaceDir, "host-tools")
	os.MkdirAll(stagingDir, 0755)
	os.WriteFile(filepath.Join(stagingDir, "tool.sh"),
		[]byte("#!/bin/bash\n# tool.sh\n# Staging version\necho staging\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"host-tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, workspaceDir)
	m.SetDevMode(true)

	if !m.IsDevMode() {
		t.Error("Manager should be in dev mode after SetDevMode(true)")
	}

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools) != 1 {
		t.Errorf("ListTools returned %d tools, want 1 (staging overrides approved)", len(tools))
	}
	// Staging version should win (highest priority)
	if len(tools) > 0 && tools[0].Description != "Staging version" {
		t.Errorf("tool description = %q, want 'Staging version' (staging should override approved)", tools[0].Description)
	}
}

func TestManager_DevMode_StagingNewTool(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create approved tool
	projectID := ProjectID(workspaceDir)
	approvedDir := filepath.Join(approvedBaseDir, projectID)
	os.MkdirAll(approvedDir, 0755)
	os.WriteFile(filepath.Join(approvedDir, "approved.sh"),
		[]byte("#!/bin/bash\n# approved.sh\n# Approved tool\n"), 0755)

	// Create new tool only in staging
	stagingDir := filepath.Join(workspaceDir, "host-tools")
	os.MkdirAll(stagingDir, 0755)
	os.WriteFile(filepath.Join(stagingDir, "new-tool.sh"),
		[]byte("#!/bin/bash\n# new-tool.sh\n# New staging tool\necho new\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"host-tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, workspaceDir)
	m.SetDevMode(true)

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	// Should see both: new-tool from staging + approved from approved dir
	if len(tools) != 2 {
		t.Errorf("ListTools returned %d tools, want 2 (staging new + approved)", len(tools))
	}
}

func TestManager_DevMode_Disabled_StagingNotIncluded(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create tool only in staging
	stagingDir := filepath.Join(workspaceDir, "host-tools")
	os.MkdirAll(stagingDir, 0755)
	os.WriteFile(filepath.Join(stagingDir, "staging-only.sh"),
		[]byte("#!/bin/bash\n# staging-only.sh\n# Staging only tool\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"host-tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, workspaceDir)
	// devMode NOT set

	if m.IsDevMode() {
		t.Error("Manager should NOT be in dev mode by default")
	}

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	// Staging tool should NOT be visible without dev mode
	if len(tools) != 0 {
		t.Errorf("ListTools returned %d tools, want 0 (staging not included without dev mode)", len(tools))
	}
}

func TestManager_LegacyMode(t *testing.T) {
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	os.MkdirAll(toolsDir, 0755)
	os.WriteFile(filepath.Join(toolsDir, "tool.sh"),
		[]byte("#!/bin/bash\n# tool.sh\n# Legacy tool\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		ApprovedDir:       "", // empty = legacy mode
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	m := NewManager(cfg, dir)

	if m.IsSecureMode() {
		t.Error("Manager should NOT be in secure mode when ApprovedDir is empty")
	}

	tools, err := m.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools) != 1 {
		t.Errorf("ListTools returned %d tools, want 1", len(tools))
	}
}
