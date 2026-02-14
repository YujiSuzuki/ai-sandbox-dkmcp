package hosttools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

func TestProjectID(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantPrefix string
		wantLen   int // minimum length
	}{
		{
			name:       "simple path",
			path:       "/home/user/my-project",
			wantPrefix: "my-project-",
			wantLen:    19, // "my-project-" + 8 hex chars
		},
		{
			name:       "path with special chars",
			path:       "/home/user/my project (2)",
			wantPrefix: "myproject2-",
			wantLen:    19,
		},
		{
			name:       "root path",
			path:       "/",
			wantPrefix: "", // sanitized "/" base is empty, falls back to "project"
			wantLen:    16, // "project-" + 8 hex chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := ProjectID(tt.path)
			if tt.wantPrefix != "" && !strings.HasPrefix(id, tt.wantPrefix) {
				t.Errorf("ProjectID(%q) = %q, want prefix %q", tt.path, id, tt.wantPrefix)
			}
			if len(id) < tt.wantLen {
				t.Errorf("ProjectID(%q) = %q (len %d), want len >= %d", tt.path, id, len(id), tt.wantLen)
			}
		})
	}
}

func TestProjectID_Deterministic(t *testing.T) {
	id1 := ProjectID("/home/user/workspace")
	id2 := ProjectID("/home/user/workspace")
	if id1 != id2 {
		t.Errorf("ProjectID should be deterministic: %q != %q", id1, id2)
	}
}

func TestProjectID_DifferentPaths(t *testing.T) {
	id1 := ProjectID("/home/user/project-a")
	id2 := ProjectID("/home/user/project-b")
	if id1 == id2 {
		t.Errorf("Different paths should produce different IDs: both = %q", id1)
	}
}

func TestResolveApprovedDir(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "tilde expansion",
			input:   "~/.dkmcp/host-tools",
			wantErr: false,
		},
		{
			name:    "absolute path",
			input:   "/tmp/approved",
			wantErr: false,
		},
		{
			name:    "relative path",
			input:   "approved-tools",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveApprovedDir(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveApprovedDir(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !filepath.IsAbs(result) {
					t.Errorf("ResolveApprovedDir(%q) = %q, want absolute path", tt.input, result)
				}
				if tt.input == "~/.dkmcp/host-tools" {
					home, _ := os.UserHomeDir()
					expected := filepath.Join(home, ".dkmcp/host-tools")
					if result != expected {
						t.Errorf("ResolveApprovedDir(%q) = %q, want %q", tt.input, result, expected)
					}
				}
			}
		})
	}
}

func TestCompareFiles(t *testing.T) {
	dir := t.TempDir()

	// Create staging file
	stagingPath := filepath.Join(dir, "staging.sh")
	os.WriteFile(stagingPath, []byte("#!/bin/bash\necho hello\n"), 0755)

	t.Run("new file (approved doesn't exist)", func(t *testing.T) {
		approvedPath := filepath.Join(dir, "nonexistent.sh")
		status, err := compareFiles(stagingPath, approvedPath)
		if err != nil {
			t.Fatalf("compareFiles error: %v", err)
		}
		if status != SyncNew {
			t.Errorf("status = %d, want SyncNew (%d)", status, SyncNew)
		}
	})

	t.Run("unchanged (identical content)", func(t *testing.T) {
		approvedPath := filepath.Join(dir, "identical.sh")
		os.WriteFile(approvedPath, []byte("#!/bin/bash\necho hello\n"), 0755)
		status, err := compareFiles(stagingPath, approvedPath)
		if err != nil {
			t.Fatalf("compareFiles error: %v", err)
		}
		if status != SyncUnchanged {
			t.Errorf("status = %d, want SyncUnchanged (%d)", status, SyncUnchanged)
		}
	})

	t.Run("updated (different content)", func(t *testing.T) {
		approvedPath := filepath.Join(dir, "different.sh")
		os.WriteFile(approvedPath, []byte("#!/bin/bash\necho world\n"), 0755)
		status, err := compareFiles(stagingPath, approvedPath)
		if err != nil {
			t.Fatalf("compareFiles error: %v", err)
		}
		if status != SyncUpdated {
			t.Errorf("status = %d, want SyncUpdated (%d)", status, SyncUpdated)
		}
	})

	t.Run("updated (different size)", func(t *testing.T) {
		approvedPath := filepath.Join(dir, "diffsize.sh")
		os.WriteFile(approvedPath, []byte("#!/bin/bash\n"), 0755)
		status, err := compareFiles(stagingPath, approvedPath)
		if err != nil {
			t.Fatalf("compareFiles error: %v", err)
		}
		if status != SyncUpdated {
			t.Errorf("status = %d, want SyncUpdated (%d)", status, SyncUpdated)
		}
	})
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "src.sh")
	content := []byte("#!/bin/bash\necho test\n")
	os.WriteFile(src, content, 0755)

	dst := filepath.Join(dir, "subdir", "dst.sh")

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	// Verify content
	dstContent, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dst: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Errorf("copied content = %q, want %q", string(dstContent), string(content))
	}

	// Verify permissions
	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("copied mode = %v, want %v", dstInfo.Mode(), srcInfo.Mode())
	}
}

func TestSyncManager_DetectChanges(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create staging directory with tools
	stagingDir := filepath.Join(workspaceDir, "host-tools")
	os.MkdirAll(stagingDir, 0755)

	os.WriteFile(filepath.Join(stagingDir, "new-tool.sh"),
		[]byte("#!/bin/bash\n# new-tool.sh\n# A new tool\necho new\n"), 0755)
	os.WriteFile(filepath.Join(stagingDir, "existing-tool.sh"),
		[]byte("#!/bin/bash\n# existing-tool.sh\n# Updated content\necho updated\n"), 0755)
	os.WriteFile(filepath.Join(stagingDir, "unchanged-tool.sh"),
		[]byte("#!/bin/bash\n# unchanged-tool.sh\n# Same content\necho same\n"), 0755)

	// Create approved directory with some existing tools
	projectID := ProjectID(workspaceDir)
	approvedDir := filepath.Join(approvedBaseDir, projectID)
	os.MkdirAll(approvedDir, 0755)

	os.WriteFile(filepath.Join(approvedDir, "existing-tool.sh"),
		[]byte("#!/bin/bash\n# existing-tool.sh\n# Old content\necho old\n"), 0755)
	os.WriteFile(filepath.Join(approvedDir, "unchanged-tool.sh"),
		[]byte("#!/bin/bash\n# unchanged-tool.sh\n# Same content\necho same\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"host-tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}

	syncMgr := NewSyncManager(cfg, workspaceDir)
	items, err := syncMgr.DetectChanges()
	if err != nil {
		t.Fatalf("DetectChanges error: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("DetectChanges returned %d items, want 3", len(items))
	}

	// Build a map for easier assertions
	statusMap := make(map[string]SyncStatus)
	for _, item := range items {
		statusMap[item.Name] = item.Status
	}

	if statusMap["new-tool.sh"] != SyncNew {
		t.Errorf("new-tool.sh status = %d, want SyncNew (%d)", statusMap["new-tool.sh"], SyncNew)
	}
	if statusMap["existing-tool.sh"] != SyncUpdated {
		t.Errorf("existing-tool.sh status = %d, want SyncUpdated (%d)", statusMap["existing-tool.sh"], SyncUpdated)
	}
	if statusMap["unchanged-tool.sh"] != SyncUnchanged {
		t.Errorf("unchanged-tool.sh status = %d, want SyncUnchanged (%d)", statusMap["unchanged-tool.sh"], SyncUnchanged)
	}
}

func TestSyncManager_RunInteractiveSync_ApproveAll(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create staging directory with a new tool
	stagingDir := filepath.Join(workspaceDir, "host-tools")
	os.MkdirAll(stagingDir, 0755)
	os.WriteFile(filepath.Join(stagingDir, "tool.sh"),
		[]byte("#!/bin/bash\n# tool.sh\n# Test tool\necho test\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"host-tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}

	syncMgr := NewSyncManager(cfg, workspaceDir)
	// Simulate user typing "y"
	syncMgr.SetReader(strings.NewReader("y\n"))

	synced, err := syncMgr.RunInteractiveSync()
	if err != nil {
		t.Fatalf("RunInteractiveSync error: %v", err)
	}
	if synced != 1 {
		t.Errorf("synced = %d, want 1", synced)
	}

	// Verify tool was copied
	projectID := ProjectID(workspaceDir)
	approvedPath := filepath.Join(approvedBaseDir, projectID, "tool.sh")
	if _, err := os.Stat(approvedPath); os.IsNotExist(err) {
		t.Errorf("tool.sh not copied to approved directory: %s", approvedPath)
	}
}

func TestSyncManager_RunInteractiveSync_DenyAll(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	stagingDir := filepath.Join(workspaceDir, "host-tools")
	os.MkdirAll(stagingDir, 0755)
	os.WriteFile(filepath.Join(stagingDir, "tool.sh"),
		[]byte("#!/bin/bash\n# tool.sh\n# Test tool\necho test\n"), 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"host-tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}

	syncMgr := NewSyncManager(cfg, workspaceDir)
	// Simulate user typing "n"
	syncMgr.SetReader(strings.NewReader("n\n"))

	synced, err := syncMgr.RunInteractiveSync()
	if err != nil {
		t.Fatalf("RunInteractiveSync error: %v", err)
	}
	if synced != 0 {
		t.Errorf("synced = %d, want 0", synced)
	}

	// Verify tool was NOT copied
	projectID := ProjectID(workspaceDir)
	approvedPath := filepath.Join(approvedBaseDir, projectID, "tool.sh")
	if _, err := os.Stat(approvedPath); !os.IsNotExist(err) {
		t.Errorf("tool.sh should NOT have been copied to %s", approvedPath)
	}
}

func TestSyncManager_RunInteractiveSync_NoChanges(t *testing.T) {
	workspaceDir := t.TempDir()
	approvedBaseDir := t.TempDir()

	// Create same tool in both staging and approved
	stagingDir := filepath.Join(workspaceDir, "host-tools")
	os.MkdirAll(stagingDir, 0755)
	content := []byte("#!/bin/bash\n# tool.sh\n# Same tool\necho same\n")
	os.WriteFile(filepath.Join(stagingDir, "tool.sh"), content, 0755)

	projectID := ProjectID(workspaceDir)
	approvedDir := filepath.Join(approvedBaseDir, projectID)
	os.MkdirAll(approvedDir, 0755)
	os.WriteFile(filepath.Join(approvedDir, "tool.sh"), content, 0755)

	cfg := &config.HostToolsConfig{
		Enabled:           true,
		ApprovedDir:       approvedBaseDir,
		StagingDirs:       []string{"host-tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}

	syncMgr := NewSyncManager(cfg, workspaceDir)

	synced, err := syncMgr.RunInteractiveSync()
	if err != nil {
		t.Fatalf("RunInteractiveSync error: %v", err)
	}
	if synced != 0 {
		t.Errorf("synced = %d, want 0 (no changes)", synced)
	}
}

func TestWriteProjectMeta(t *testing.T) {
	dir := t.TempDir()

	err := writeProjectMeta(dir, "/home/user/my-project")
	if err != nil {
		t.Fatalf("writeProjectMeta error: %v", err)
	}

	metaPath := filepath.Join(dir, projectMetaFile)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("reading metadata: %v", err)
	}

	if !strings.Contains(string(data), "/home/user/my-project") {
		t.Errorf("metadata does not contain workspace path: %s", string(data))
	}
}
