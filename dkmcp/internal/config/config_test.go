// Package config tests verify the configuration loading and validation logic.
// configパッケージのテストは設定の読み込みと検証ロジックを検証します。
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoad_ValidConfig tests loading a complete, valid configuration file.
// It verifies that all sections (server, security, logging) are correctly parsed.
//
// TestLoad_ValidConfigは完全で有効な設定ファイルの読み込みをテストします。
// すべてのセクション（server、security、logging）が正しく解析されることを検証します。
func TestLoad_ValidConfig(t *testing.T) {
	// Create a temporary config file for testing
	// テスト用の一時設定ファイルを作成
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "dkmcp.yaml")

	// Define a complete configuration with all supported options
	// すべてのサポートされたオプションを含む完全な設定を定義
	configContent := `
server:
  port: 8080
  host: "0.0.0.0"

security:
  mode: "moderate"
  allowed_containers:
    - "test-*"
    - "demo-app"
  exec_whitelist:
    "demo-app":
      - "npm test"
      - "npm run lint"
  permissions:
    logs: true
    inspect: true
    stats: true
    exec: true

logging:
  level: "info"
  format: "json"
  output: "stdout"
`

	// Write the test configuration to the temporary file
	// テスト設定を一時ファイルに書き込み
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Load the configuration and verify no errors occur
	// 設定を読み込み、エラーが発生しないことを確認
	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify server config was correctly parsed
	// サーバー設定が正しく解析されたことを確認
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %s, want 0.0.0.0", cfg.Server.Host)
	}

	// Verify security mode was correctly parsed
	// セキュリティモードが正しく解析されたことを確認
	if cfg.Security.Mode != "moderate" {
		t.Errorf("Security.Mode = %s, want moderate", cfg.Security.Mode)
	}

	// Verify allowed containers list was correctly parsed
	// 許可されたコンテナリストが正しく解析されたことを確認
	expectedContainers := []string{"test-*", "demo-app"}
	if len(cfg.Security.AllowedContainers) != len(expectedContainers) {
		t.Errorf("AllowedContainers length = %d, want %d",
			len(cfg.Security.AllowedContainers), len(expectedContainers))
	}

	// Verify exec whitelist was correctly parsed
	// 実行ホワイトリストが正しく解析されたことを確認
	if len(cfg.Security.ExecWhitelist) != 1 {
		t.Errorf("ExecWhitelist length = %d, want 1", len(cfg.Security.ExecWhitelist))
	}

	// Verify specific container's whitelisted commands
	// 特定のコンテナのホワイトリストコマンドを確認
	if commands, ok := cfg.Security.ExecWhitelist["demo-app"]; ok {
		if len(commands) != 2 {
			t.Errorf("demo-app whitelist length = %d, want 2", len(commands))
		}
	} else {
		t.Error("demo-app not found in exec whitelist")
	}

	// Verify permissions were correctly parsed
	// パーミッションが正しく解析されたことを確認
	if !cfg.Security.Permissions.Logs {
		t.Error("Logs permission should be enabled")
	}
	if !cfg.Security.Permissions.Exec {
		t.Error("Exec permission should be enabled")
	}

	// Verify logging config was correctly parsed
	// ロギング設定が正しく解析されたことを確認
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %s, want info", cfg.Logging.Level)
	}
}

// TestLoad_FileNotFound tests that Load returns an error for non-existent files.
// This ensures proper error handling when config files are missing.
//
// TestLoad_FileNotFoundは存在しないファイルに対してLoadがエラーを返すことをテストします。
// 設定ファイルが見つからない場合の適切なエラーハンドリングを確認します。
func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

// TestLoad_InvalidYAML tests that Load returns an error for malformed YAML.
// This verifies the parser correctly rejects syntactically invalid files.
//
// TestLoad_InvalidYAMLは不正な形式のYAMLに対してLoadがエラーを返すことをテストします。
// パーサーが構文的に無効なファイルを正しく拒否することを確認します。
func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	// Create an intentionally malformed YAML file
	// 意図的に不正な形式のYAMLファイルを作成
	invalidContent := `
server:
  port: "not a number"
  invalid yaml content
    bad indentation
`

	err := os.WriteFile(configFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Load should fail for invalid YAML
	// 無効なYAMLに対してLoadは失敗するべき
	_, err = Load(configFile)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// TestValidate_ValidConfig tests that Validate accepts a properly configured Config.
// This is the happy path test for validation.
//
// TestValidate_ValidConfigは適切に設定されたConfigをValidateが受け入れることをテストします。
// これは検証のハッピーパステストです。
func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Security: SecurityConfig{
			Mode:              "moderate",
			AllowedContainers: []string{"test-*"},
			Permissions: SecurityPermissions{
				Logs:    true,
				Inspect: true,
				Stats:   true,
				Exec:    true,
			},
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}

	// Validation should pass for valid config
	// 有効な設定に対して検証は成功するべき
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestValidate_InvalidPort tests that Validate rejects invalid port numbers.
// Uses table-driven tests to check multiple invalid port scenarios.
//
// TestValidate_InvalidPortは無効なポート番号をValidateが拒否することをテストします。
// テーブル駆動テストを使用して複数の無効なポートシナリオをチェックします。
func TestValidate_InvalidPort(t *testing.T) {
	// Define test cases for invalid ports
	// 無効なポートのテストケースを定義
	tests := []struct {
		name string // Test case name / テストケース名
		port int    // Port to test / テストするポート
	}{
		{"port too low", 0},      // Zero is invalid / ゼロは無効
		{"port negative", -1},    // Negative is invalid / 負数は無効
		{"port too high", 70000}, // Above 65535 is invalid / 65535より大きい値は無効
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: tt.port,
					Host: "0.0.0.0",
				},
				Security: SecurityConfig{
					Mode: "moderate",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			}

			// Validation should fail for invalid ports
			// 無効なポートに対して検証は失敗するべき
			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error for port %d", tt.port)
			}
		})
	}
}

// TestValidate_InvalidSecurityMode tests that Validate rejects unknown security modes.
// Only "strict", "moderate", and "permissive" are valid.
//
// TestValidate_InvalidSecurityModeは不明なセキュリティモードをValidateが拒否することをテストします。
// "strict"、"moderate"、"permissive"のみが有効です。
func TestValidate_InvalidSecurityMode(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Security: SecurityConfig{
			Mode: "invalid-mode", // Invalid mode / 無効なモード
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}

	// Validation should fail for invalid security mode
	// 無効なセキュリティモードに対して検証は失敗するべき
	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid security mode")
	}
}

// TestValidate_ValidSecurityModes tests that all three security modes are accepted.
// This ensures the allowlist for security modes is correctly implemented.
//
// TestValidate_ValidSecurityModesは3つのセキュリティモードすべてが受け入れられることをテストします。
// セキュリティモードの許可リストが正しく実装されていることを確認します。
func TestValidate_ValidSecurityModes(t *testing.T) {
	// All valid security modes / すべての有効なセキュリティモード
	modes := []string{"strict", "moderate", "permissive"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Security: SecurityConfig{
					Mode: mode,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			}

			// Validation should pass for all valid modes
			// すべての有効なモードに対して検証は成功するべき
			err := cfg.Validate()
			if err != nil {
				t.Errorf("Validate() error = %v for mode %s", err, mode)
			}
		})
	}
}

// TestValidate_InvalidLogLevel tests that Validate rejects unknown log levels.
// Only "debug", "info", "warn", and "error" are valid.
//
// TestValidate_InvalidLogLevelは不明なログレベルをValidateが拒否することをテストします。
// "debug"、"info"、"warn"、"error"のみが有効です。
func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Security: SecurityConfig{
			Mode: "moderate",
		},
		Logging: LoggingConfig{
			Level: "invalid-level", // Invalid level / 無効なレベル
		},
	}

	// Validation should fail for invalid log level
	// 無効なログレベルに対して検証は失敗するべき
	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid log level")
	}
}

// TestValidate_ValidLogLevels tests that all four log levels are accepted.
// This ensures the allowlist for log levels is correctly implemented.
//
// TestValidate_ValidLogLevelsは4つのログレベルすべてが受け入れられることをテストします。
// ログレベルの許可リストが正しく実装されていることを確認します。
func TestValidate_ValidLogLevels(t *testing.T) {
	// All valid log levels / すべての有効なログレベル
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Security: SecurityConfig{
					Mode: "moderate",
				},
				Logging: LoggingConfig{
					Level: level,
				},
			}

			// Validation should pass for all valid levels
			// すべての有効なレベルに対して検証は成功するべき
			err := cfg.Validate()
			if err != nil {
				t.Errorf("Validate() error = %v for log level %s", err, level)
			}
		})
	}
}

// TestValidate_EmptyAllowedContainers tests that an empty container list is valid.
// An empty list means no containers are accessible, which is a valid security choice.
//
// TestValidate_EmptyAllowedContainersは空のコンテナリストが有効であることをテストします。
// 空のリストはアクセス可能なコンテナがないことを意味し、有効なセキュリティ選択です。
func TestValidate_EmptyAllowedContainers(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Security: SecurityConfig{
			Mode:              "strict",
			AllowedContainers: []string{}, // Empty container list / 空のコンテナリスト
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}

	// Empty allowed containers should be valid (means no containers accessible)
	// 空の許可コンテナリストは有効（アクセス可能なコンテナなしを意味）
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, should allow empty container list", err)
	}
}

// TestLoad_WithDefaults tests that missing config values are filled with defaults.
// This ensures users don't need to specify every option.
//
// TestLoad_WithDefaultsは欠けている設定値がデフォルトで埋められることをテストします。
// ユーザーがすべてのオプションを指定する必要がないことを確認します。
func TestLoad_WithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "minimal.yaml")

	// Create a minimal config that omits server and logging sections
	// serverとloggingセクションを省略した最小限の設定を作成
	minimalContent := `
security:
  mode: "moderate"
  allowed_containers:
    - "test-*"
`

	err := os.WriteFile(configFile, []byte(minimalContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Load should succeed and apply defaults for missing fields
	// Loadは成功し、欠けているフィールドにデフォルトを適用するべき
	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify default port is applied (should be 8080)
	// デフォルトポートが適用されていることを確認（8080のはず）
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}

	// Verify default host is applied (should be "0.0.0.0")
	// デフォルトホストが適用されていることを確認（"0.0.0.0"のはず）
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want \"0.0.0.0\"", cfg.Server.Host)
	}

	// Verify default log level is applied (should be "info")
	// デフォルトログレベルが適用されていることを確認（"info"のはず）
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q, want \"info\"", cfg.Logging.Level)
	}

	// Verify default permissions are applied
	// デフォルトパーミッションが適用されていることを確認
	if !cfg.Security.Permissions.Logs {
		t.Error("Security.Permissions.Logs should default to true")
	}
	if !cfg.Security.Permissions.Stats {
		t.Error("Security.Permissions.Stats should default to true")
	}
	if cfg.Security.Permissions.Lifecycle {
		t.Error("Security.Permissions.Lifecycle should default to false")
	}
}

// TestLoad_LifecyclePermission tests that lifecycle permission is correctly parsed.
// Verifies both explicit true and default false behavior.
//
// TestLoad_LifecyclePermissionはlifecycleパーミッションが正しく解析されることをテストします。
// 明示的なtrueとデフォルトのfalseの両方の動作を確認します。
func TestLoad_LifecyclePermission(t *testing.T) {
	tmpDir := t.TempDir()

	// Test 1: lifecycle: true is parsed correctly
	configFile := filepath.Join(tmpDir, "lifecycle-true.yaml")
	configContent := `
security:
  mode: "moderate"
  permissions:
    logs: true
    inspect: true
    stats: true
    exec: true
    lifecycle: true
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Security.Permissions.Lifecycle {
		t.Error("Security.Permissions.Lifecycle should be true when set to true")
	}

	// Test 2: lifecycle defaults to false when omitted
	configFile2 := filepath.Join(tmpDir, "lifecycle-default.yaml")
	configContent2 := `
security:
  mode: "moderate"
  permissions:
    logs: true
    exec: true
`
	err = os.WriteFile(configFile2, []byte(configContent2), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg2, err := Load(configFile2)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg2.Security.Permissions.Lifecycle {
		t.Error("Security.Permissions.Lifecycle should default to false when omitted")
	}
}

// TestHostAccessConfig_Defaults tests that HostAccessConfig has correct default values.
// Both host_tools and host_commands should be disabled by default.
//
// TestHostAccessConfig_DefaultsはHostAccessConfigが正しいデフォルト値を持つことをテストします。
// host_toolsとhost_commandsの両方がデフォルトで無効であるべきです。
func TestHostAccessConfig_Defaults(t *testing.T) {
	cfg := NewDefaultConfig()

	// HostAccess should exist with empty workspace_root
	// HostAccessは空のworkspace_rootで存在するべき
	if cfg.HostAccess.WorkspaceRoot != "" {
		t.Errorf("HostAccess.WorkspaceRoot = %q, want empty", cfg.HostAccess.WorkspaceRoot)
	}

	// HostTools should be disabled by default
	// HostToolsはデフォルトで無効であるべき
	if cfg.HostAccess.HostTools.Enabled {
		t.Error("HostAccess.HostTools.Enabled should be false by default")
	}
	// Default directories (legacy)
	if len(cfg.HostAccess.HostTools.Directories) != 1 || cfg.HostAccess.HostTools.Directories[0] != ".sandbox/host-tools" {
		t.Errorf("HostAccess.HostTools.Directories = %v, want [\".sandbox/host-tools\"]", cfg.HostAccess.HostTools.Directories)
	}
	// Default staging dirs
	if len(cfg.HostAccess.HostTools.StagingDirs) != 1 || cfg.HostAccess.HostTools.StagingDirs[0] != ".sandbox/host-tools" {
		t.Errorf("HostAccess.HostTools.StagingDirs = %v, want [\".sandbox/host-tools\"]", cfg.HostAccess.HostTools.StagingDirs)
	}
	// ApprovedDir should be empty by default
	if cfg.HostAccess.HostTools.ApprovedDir != "" {
		t.Errorf("HostAccess.HostTools.ApprovedDir = %q, want empty", cfg.HostAccess.HostTools.ApprovedDir)
	}
	// Common should be true by default
	if !cfg.HostAccess.HostTools.Common {
		t.Error("HostAccess.HostTools.Common should be true by default")
	}
	// IsSecureMode should be false by default (no approved_dir)
	if cfg.HostAccess.HostTools.IsSecureMode() {
		t.Error("HostAccess.HostTools.IsSecureMode() should be false by default")
	}
	// Default allowed extensions
	expectedExts := []string{".sh", ".go", ".py"}
	if len(cfg.HostAccess.HostTools.AllowedExtensions) != len(expectedExts) {
		t.Errorf("HostAccess.HostTools.AllowedExtensions length = %d, want %d",
			len(cfg.HostAccess.HostTools.AllowedExtensions), len(expectedExts))
	}
	// Default timeout
	if cfg.HostAccess.HostTools.Timeout != 60 {
		t.Errorf("HostAccess.HostTools.Timeout = %d, want 60", cfg.HostAccess.HostTools.Timeout)
	}

	// HostCommands should be disabled by default
	// HostCommandsはデフォルトで無効であるべき
	if cfg.HostAccess.HostCommands.Enabled {
		t.Error("HostAccess.HostCommands.Enabled should be false by default")
	}
	if cfg.HostAccess.HostCommands.Whitelist == nil {
		t.Error("HostAccess.HostCommands.Whitelist should be initialized (empty map)")
	}
	if cfg.HostAccess.HostCommands.Deny == nil {
		t.Error("HostAccess.HostCommands.Deny should be initialized (empty map)")
	}
	if cfg.HostAccess.HostCommands.Dangerously.Enabled {
		t.Error("HostAccess.HostCommands.Dangerously.Enabled should be false by default")
	}
	if cfg.HostAccess.HostCommands.Dangerously.Commands == nil {
		t.Error("HostAccess.HostCommands.Dangerously.Commands should be initialized (empty map)")
	}
}

// TestHostAccessConfig_ParseYAML tests that HostAccessConfig is correctly parsed from YAML.
//
// TestHostAccessConfig_ParseYAMLはHostAccessConfigがYAMLから正しく解析されることをテストします。
func TestHostAccessConfig_ParseYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "dkmcp.yaml")

	configContent := `
server:
  port: 8080

security:
  mode: "moderate"

logging:
  level: "info"

host_access:
  workspace_root: "/home/user/project"

  host_tools:
    enabled: true
    directories:
      - ".sandbox/host-tools"
      - "tools"
    allowed_extensions:
      - ".sh"
      - ".go"
    timeout: 120

  host_commands:
    enabled: true
    allowed_containers: ["securenote-*", "demo-*"]
    allowed_projects: ["demo-apps"]
    whitelist:
      "docker":
        - "ps"
        - "logs *"
      "git":
        - "status"
        - "diff *"
    deny:
      "docker":
        - "rm *"
        - "rmi *"
    dangerously:
      enabled: true
      commands:
        "docker":
          - "restart"
          - "stop"
        "git":
          - "checkout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify workspace_root
	if cfg.HostAccess.WorkspaceRoot != "/home/user/project" {
		t.Errorf("HostAccess.WorkspaceRoot = %q, want \"/home/user/project\"", cfg.HostAccess.WorkspaceRoot)
	}

	// Verify host_tools
	if !cfg.HostAccess.HostTools.Enabled {
		t.Error("HostAccess.HostTools.Enabled should be true")
	}
	if len(cfg.HostAccess.HostTools.Directories) != 2 {
		t.Errorf("HostAccess.HostTools.Directories length = %d, want 2", len(cfg.HostAccess.HostTools.Directories))
	}
	if len(cfg.HostAccess.HostTools.AllowedExtensions) != 2 {
		t.Errorf("HostAccess.HostTools.AllowedExtensions length = %d, want 2", len(cfg.HostAccess.HostTools.AllowedExtensions))
	}
	if cfg.HostAccess.HostTools.Timeout != 120 {
		t.Errorf("HostAccess.HostTools.Timeout = %d, want 120", cfg.HostAccess.HostTools.Timeout)
	}
	// Legacy mode: no approved_dir set
	if cfg.HostAccess.HostTools.IsSecureMode() {
		t.Error("HostAccess.HostTools.IsSecureMode() should be false when approved_dir is not set")
	}

	// Verify host_commands
	if !cfg.HostAccess.HostCommands.Enabled {
		t.Error("HostAccess.HostCommands.Enabled should be true")
	}
	if len(cfg.HostAccess.HostCommands.AllowedContainers) != 2 {
		t.Errorf("HostAccess.HostCommands.AllowedContainers length = %d, want 2", len(cfg.HostAccess.HostCommands.AllowedContainers))
	}
	if len(cfg.HostAccess.HostCommands.AllowedProjects) != 1 {
		t.Errorf("HostAccess.HostCommands.AllowedProjects length = %d, want 1", len(cfg.HostAccess.HostCommands.AllowedProjects))
	}

	// Verify whitelist
	dockerWhitelist := cfg.HostAccess.HostCommands.Whitelist["docker"]
	if len(dockerWhitelist) != 2 {
		t.Errorf("docker whitelist length = %d, want 2", len(dockerWhitelist))
	}
	gitWhitelist := cfg.HostAccess.HostCommands.Whitelist["git"]
	if len(gitWhitelist) != 2 {
		t.Errorf("git whitelist length = %d, want 2", len(gitWhitelist))
	}

	// Verify deny
	dockerDeny := cfg.HostAccess.HostCommands.Deny["docker"]
	if len(dockerDeny) != 2 {
		t.Errorf("docker deny length = %d, want 2", len(dockerDeny))
	}

	// Verify dangerously
	if !cfg.HostAccess.HostCommands.Dangerously.Enabled {
		t.Error("HostAccess.HostCommands.Dangerously.Enabled should be true")
	}
	dockerDangerous := cfg.HostAccess.HostCommands.Dangerously.Commands["docker"]
	if len(dockerDangerous) != 2 {
		t.Errorf("docker dangerous commands length = %d, want 2", len(dockerDangerous))
	}
	gitDangerous := cfg.HostAccess.HostCommands.Dangerously.Commands["git"]
	if len(gitDangerous) != 1 {
		t.Errorf("git dangerous commands length = %d, want 1", len(gitDangerous))
	}
}

// TestHostToolsConfig_SecureMode tests parsing of secure mode configuration.
//
// TestHostToolsConfig_SecureModeはセキュアモード設定の解析をテストします。
func TestHostToolsConfig_SecureMode(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "dkmcp.yaml")

	configContent := `
server:
  port: 8080

security:
  mode: "moderate"

logging:
  level: "info"

host_access:
  workspace_root: "/home/user/project"

  host_tools:
    enabled: true
    approved_dir: "~/.dkmcp/host-tools"
    staging_dirs:
      - ".sandbox/host-tools"
      - "tools"
    common: true
    allowed_extensions:
      - ".sh"
    timeout: 30
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.HostAccess.HostTools.IsSecureMode() {
		t.Error("HostAccess.HostTools.IsSecureMode() should be true when approved_dir is set")
	}
	if cfg.HostAccess.HostTools.ApprovedDir != "~/.dkmcp/host-tools" {
		t.Errorf("ApprovedDir = %q, want \"~/.dkmcp/host-tools\"", cfg.HostAccess.HostTools.ApprovedDir)
	}
	if len(cfg.HostAccess.HostTools.StagingDirs) != 2 {
		t.Errorf("StagingDirs length = %d, want 2", len(cfg.HostAccess.HostTools.StagingDirs))
	}
	if !cfg.HostAccess.HostTools.Common {
		t.Error("Common should be true")
	}
}

// TestHostAccessConfig_Validation tests validation of HostAccessConfig.
// Ensures invalid values like negative timeouts are rejected.
//
// TestHostAccessConfig_ValidationはHostAccessConfigの検証をテストします。
// 負のタイムアウトのような無効な値が拒否されることを確認します。
func TestHostAccessConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(cfg *Config)
		wantErr bool
	}{
		{
			name: "valid host_tools config",
			modify: func(cfg *Config) {
				cfg.HostAccess.HostTools.Enabled = true
				cfg.HostAccess.HostTools.Timeout = 30
			},
			wantErr: false,
		},
		{
			name: "negative timeout rejected",
			modify: func(cfg *Config) {
				cfg.HostAccess.HostTools.Enabled = true
				cfg.HostAccess.HostTools.Timeout = -1
			},
			wantErr: true,
		},
		{
			name: "zero timeout rejected",
			modify: func(cfg *Config) {
				cfg.HostAccess.HostTools.Enabled = true
				cfg.HostAccess.HostTools.Timeout = 0
			},
			wantErr: true,
		},
		{
			name: "disabled host_tools skips validation",
			modify: func(cfg *Config) {
				cfg.HostAccess.HostTools.Enabled = false
				cfg.HostAccess.HostTools.Timeout = -1 // invalid but ignored when disabled
			},
			wantErr: false,
		},
		{
			name: "host_commands requires workspace_root",
			modify: func(cfg *Config) {
				cfg.HostAccess.HostCommands.Enabled = true
				cfg.HostAccess.WorkspaceRoot = ""
			},
			wantErr: true,
		},
		{
			name: "host_commands with workspace_root is valid",
			modify: func(cfg *Config) {
				cfg.HostAccess.HostCommands.Enabled = true
				cfg.HostAccess.WorkspaceRoot = "/workspace"
			},
			wantErr: false,
		},
		{
			name: "disabled host_commands skips workspace_root check",
			modify: func(cfg *Config) {
				cfg.HostAccess.HostCommands.Enabled = false
				cfg.HostAccess.WorkspaceRoot = "" // empty but ignored when disabled
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
