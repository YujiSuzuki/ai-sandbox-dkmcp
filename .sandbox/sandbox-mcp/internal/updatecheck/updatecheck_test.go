package updatecheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadStateFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "valid state file",
			content:     "1705315800:v1.2.0",
			wantVersion: "v1.2.0",
			wantErr:     false,
		},
		{
			name:        "empty version",
			content:     "1705315800:",
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:        "no timestamp (invalid format)",
			content:     "v1.2.0",
			wantVersion: "",
			wantErr:     true,
		},
		{
			name:        "empty file",
			content:     "",
			wantVersion: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary state file
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "update-check")

			// Always create file (empty or with content)
			err := os.WriteFile(stateFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Test ReadStateFile
			version, err := ReadStateFile(stateFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadStateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && version != tt.wantVersion {
				t.Errorf("ReadStateFile() version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func TestReadStateFile_NotFound(t *testing.T) {
	// State file doesn't exist
	stateFile := "/tmp/nonexistent-update-check-file"

	version, err := ReadStateFile(stateFile)

	// Should return empty version (no error, first run case)
	if err != nil {
		t.Errorf("ReadStateFile() should not error on missing file, got: %v", err)
	}

	if version != "" {
		t.Errorf("ReadStateFile() version = %q, want empty string", version)
	}
}

func TestParseTemplateConfig(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantRepo     string
		wantChannel  string
		wantEnabled  bool
		wantInterval int
		wantErr      bool
	}{
		{
			name: "valid config",
			content: `TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"
CHECK_UPDATES="true"
CHECK_INTERVAL_HOURS="24"
`,
			wantRepo:     "YujiSuzuki/ai-sandbox-dkmcp",
			wantChannel:  "all",
			wantEnabled:  true,
			wantInterval: 24,
			wantErr:      false,
		},
		{
			name: "disabled updates",
			content: `TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="stable"
CHECK_UPDATES="false"
CHECK_INTERVAL_HOURS="48"
`,
			wantRepo:     "YujiSuzuki/ai-sandbox-dkmcp",
			wantChannel:  "stable",
			wantEnabled:  false,
			wantInterval: 48,
			wantErr:      false,
		},
		{
			name: "missing repo (invalid)",
			content: `CHECK_CHANNEL="all"
CHECK_UPDATES="true"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "template-source.conf")
			err := os.WriteFile(configFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Test ParseTemplateConfig
			cfg, err := ParseTemplateConfig(configFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTemplateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if cfg.Repo != tt.wantRepo {
				t.Errorf("ParseTemplateConfig() Repo = %q, want %q", cfg.Repo, tt.wantRepo)
			}

			if cfg.Channel != tt.wantChannel {
				t.Errorf("ParseTemplateConfig() Channel = %q, want %q", cfg.Channel, tt.wantChannel)
			}

			if cfg.Enabled != tt.wantEnabled {
				t.Errorf("ParseTemplateConfig() Enabled = %v, want %v", cfg.Enabled, tt.wantEnabled)
			}

			if cfg.IntervalHours != tt.wantInterval {
				t.Errorf("ParseTemplateConfig() IntervalHours = %d, want %d", cfg.IntervalHours, tt.wantInterval)
			}
		})
	}
}

func TestGetUpdateStatus(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "update-check")
	configFile := filepath.Join(tmpDir, "template-source.conf")

	// Write test state file
	err := os.WriteFile(stateFile, []byte("1705315800:v1.2.0"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Write test config file
	configContent := `TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"
CHECK_UPDATES="true"
CHECK_INTERVAL_HOURS="24"
`
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test GetUpdateStatus
	status, err := GetUpdateStatus(stateFile, configFile)
	if err != nil {
		t.Fatalf("GetUpdateStatus() error = %v", err)
	}

	if status.LatestVersion != "v1.2.0" {
		t.Errorf("LatestVersion = %q, want %q", status.LatestVersion, "v1.2.0")
	}

	if status.Repo != "YujiSuzuki/ai-sandbox-dkmcp" {
		t.Errorf("Repo = %q, want %q", status.Repo, "YujiSuzuki/ai-sandbox-dkmcp")
	}

	if status.ReleaseURL == "" {
		t.Error("ReleaseURL should not be empty")
	}
}
