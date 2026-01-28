// Package security tests verify the blocked paths functionality.
// These tests ensure that file path blocking works correctly across
// various scenarios: manual blocks, global patterns, auto-import, etc.
//
// securityパッケージのテストはブロックパス機能を検証します。
// これらのテストはファイルパスブロックが様々なシナリオで正しく動作することを確認します：
// 手動ブロック、グローバルパターン、自動インポートなど。
package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// TestNewBlockedPathsManager verifies that NewBlockedPathsManager creates
// a properly initialized manager instance.
//
// TestNewBlockedPathsManagerはNewBlockedPathsManagerが適切に初期化された
// マネージャインスタンスを作成することを検証します。
func TestNewBlockedPathsManager(t *testing.T) {
	cfg := &config.BlockedPathsConfig{
		Manual: map[string][]string{
			"test-app": {".env", "secrets/*"},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{"test-app"})

	// Verify manager was created
	// マネージャが作成されたことを確認
	if manager == nil {
		t.Fatal("NewBlockedPathsManager returned nil")
	}
}

// TestLoadBlockedPaths_ManualPaths tests that manually configured blocked paths
// are correctly loaded and stored.
//
// TestLoadBlockedPaths_ManualPathsは手動設定されたブロックパスが正しくロードされ
// 保存されることをテストします。
func TestLoadBlockedPaths_ManualPaths(t *testing.T) {
	cfg := &config.BlockedPathsConfig{
		Manual: map[string][]string{
			"test-app":  {".env", "secrets/*"},
			"other-app": {"config.json"},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{"test-app", "other-app"})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	paths := manager.GetBlockedPaths()

	// Verify at least 3 paths are loaded (2 for test-app + 1 for other-app)
	// 少なくとも3つのパスがロードされていることを確認（test-app用2つ + other-app用1つ）
	if len(paths) < 3 {
		t.Errorf("expected at least 3 blocked paths, got %d", len(paths))
	}

	// Verify specific manual path is loaded with correct metadata
	// 特定の手動パスが正しいメタデータでロードされていることを確認
	found := false
	for _, p := range paths {
		if p.Container == "test-app" && p.Pattern == ".env" && p.Reason == "manual_block" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find manual blocked path for test-app:.env")
	}
}

// TestLoadBlockedPaths_GlobalPatterns tests that global patterns are loaded
// and apply to all containers (Container = "*").
//
// TestLoadBlockedPaths_GlobalPatternsはグローバルパターンがロードされ
// 全コンテナに適用される（Container = "*"）ことをテストします。
func TestLoadBlockedPaths_GlobalPatterns(t *testing.T) {
	cfg := &config.BlockedPathsConfig{
		AutoImport: config.AutoImportConfig{
			GlobalPatterns: []string{".env", "*.key", "*.pem"},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	paths := manager.GetBlockedPaths()

	// Verify exactly 3 global patterns are loaded
	// 正確に3つのグローバルパターンがロードされていることを確認
	if len(paths) != 3 {
		t.Errorf("expected 3 global patterns, got %d", len(paths))
	}

	// Verify all global patterns have "*" as container and correct reason
	// 全グローバルパターンがコンテナとして"*"と正しいreasonを持つことを確認
	for _, p := range paths {
		if p.Container != "*" {
			t.Errorf("expected container to be '*' for global pattern, got %q", p.Container)
		}
		if p.Reason != "global_pattern" {
			t.Errorf("expected reason to be 'global_pattern', got %q", p.Reason)
		}
	}
}

// TestIsPathBlocked_ManualBlock tests path blocking with manual block rules.
// Uses table-driven tests to verify various path matching scenarios.
//
// TestIsPathBlocked_ManualBlockは手動ブロックルールでのパスブロックをテストします。
// テーブル駆動テストを使用して様々なパスマッチングシナリオを検証します。
func TestIsPathBlocked_ManualBlock(t *testing.T) {
	cfg := &config.BlockedPathsConfig{
		Manual: map[string][]string{
			"test-app": {".env", "/secrets/*"},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{"test-app"})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container to check / チェックするコンテナ
		path      string // Path to check / チェックするパス
		blocked   bool   // Expected result / 期待される結果
	}{
		{"exact match", "test-app", ".env", true},           // Filename exact match / ファイル名の完全一致
		{"path with .env", "test-app", "/app/.env", true},   // .env in any directory / 任意のディレクトリの.env
		{"secrets dir", "test-app", "/secrets/key.pem", true}, // secrets/* pattern / secrets/*パターン
		{"not blocked", "test-app", "/app/index.js", false}, // Not matching any pattern / どのパターンにもマッチしない
		{"different container", "other-app", ".env", false}, // Wrong container / 間違ったコンテナ
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := manager.IsPathBlocked(tt.container, tt.path)
			if (blocked != nil) != tt.blocked {
				t.Errorf("IsPathBlocked(%q, %q) = %v, want blocked=%v",
					tt.container, tt.path, blocked != nil, tt.blocked)
			}
		})
	}
}

// TestIsPathBlocked_GlobalPattern tests that global patterns apply to all containers.
// Global patterns (Container = "*") should block paths in any container.
//
// TestIsPathBlocked_GlobalPatternはグローバルパターンが全コンテナに適用されることをテストします。
// グローバルパターン（Container = "*"）は任意のコンテナのパスをブロックすべきです。
func TestIsPathBlocked_GlobalPattern(t *testing.T) {
	cfg := &config.BlockedPathsConfig{
		AutoImport: config.AutoImportConfig{
			GlobalPatterns: []string{".env", "*.key"},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container to check / チェックするコンテナ
		path      string // Path to check / チェックするパス
		blocked   bool   // Expected result / 期待される結果
	}{
		{"global .env any container", "any-app", "/app/.env", true},       // .env blocked everywhere / .envはどこでもブロック
		{"global .key any container", "any-app", "/config/secret.key", true}, // *.key blocked everywhere / *.keyはどこでもブロック
		{"not blocked", "any-app", "/app/index.js", false},                // No matching pattern / マッチするパターンなし
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := manager.IsPathBlocked(tt.container, tt.path)
			if (blocked != nil) != tt.blocked {
				t.Errorf("IsPathBlocked(%q, %q) = %v, want blocked=%v",
					tt.container, tt.path, blocked != nil, tt.blocked)
			}
		})
	}
}

// TestGetBlockedPathsForContainer tests that GetBlockedPathsForContainer returns
// both container-specific and global paths.
//
// TestGetBlockedPathsForContainerはGetBlockedPathsForContainerがコンテナ固有と
// グローバルの両方のパスを返すことをテストします。
func TestGetBlockedPathsForContainer(t *testing.T) {
	cfg := &config.BlockedPathsConfig{
		Manual: map[string][]string{
			"test-app":  {".env"},
			"other-app": {"config.json"},
		},
		AutoImport: config.AutoImportConfig{
			GlobalPatterns: []string{"*.key"},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{"test-app", "other-app"})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	// test-app should have its own .env + global *.key
	// test-appは自身の.env + グローバル*.keyを持つべき
	testAppPaths := manager.GetBlockedPathsForContainer("test-app")
	if len(testAppPaths) != 2 {
		t.Errorf("expected 2 blocked paths for test-app, got %d", len(testAppPaths))
	}

	// other-app should have its own config.json + global *.key
	// other-appは自身のconfig.json + グローバル*.keyを持つべき
	otherAppPaths := manager.GetBlockedPathsForContainer("other-app")
	if len(otherAppPaths) != 2 {
		t.Errorf("expected 2 blocked paths for other-app, got %d", len(otherAppPaths))
	}
}

// TestParseDockerCompose tests auto-import from docker-compose.yml files.
// Verifies that /dev/null mounts and tmpfs mounts are detected as blocked paths.
//
// TestParseDockerComposeはdocker-compose.ymlファイルからの自動インポートをテストします。
// /dev/nullマウントとtmpfsマウントがブロックパスとして検出されることを検証します。
func TestParseDockerCompose(t *testing.T) {
	// Create temporary docker-compose.yml for testing
	// テスト用の一時的なdocker-compose.ymlを作成
	tmpDir := t.TempDir()
	composeContent := `
services:
  devcontainer:
    volumes:
      - /dev/null:/workspace/test-app/.env:ro
    tmpfs:
      - /workspace/test-app/secrets:ro
`
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &config.BlockedPathsConfig{
		AutoImport: config.AutoImportConfig{
			Enabled:       true,
			WorkspaceRoot: tmpDir,
			ScanFiles:     []string{"docker-compose.yml"},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{"test-app"})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	paths := manager.GetBlockedPaths()

	// Verify at least 2 paths are detected from docker-compose
	// docker-composeから少なくとも2つのパスが検出されていることを確認
	if len(paths) < 2 {
		t.Errorf("expected at least 2 blocked paths from docker-compose, got %d", len(paths))
	}

	// Verify both /dev/null mount and tmpfs mount are detected with correct values
	// /dev/nullマウントとtmpfsマウントの両方が正しい値で検出されていることを確認
	var envPath, secretsPath *BlockedPath
	for i := range paths {
		p := &paths[i]
		if p.Reason == "volume_mount_to_dev_null" {
			envPath = p
		}
		if p.Reason == "tmpfs_mount" {
			secretsPath = p
		}
	}

	if envPath == nil {
		t.Error("expected to find blocked path from /dev/null volume mount")
	} else {
		// Verify the /dev/null mount has correct Container and Pattern
		// /dev/nullマウントが正しいContainerとPatternを持つことを確認
		// Container comes from containers list passed to NewBlockedPathsManager
		// Containerは NewBlockedPathsManager に渡された containers リストから取得される
		if envPath.Container != "test-app" {
			t.Errorf("envPath.Container = %q, want \"test-app\"", envPath.Container)
		}
		// Pattern is the container-relative path extracted from the mount
		// Patternはマウントから抽出されたコンテナ相対パス
		if envPath.Pattern != "/.env" {
			t.Errorf("envPath.Pattern = %q, want \"/.env\"", envPath.Pattern)
		}
	}

	if secretsPath == nil {
		t.Error("expected to find blocked path from tmpfs mount")
	} else {
		// Verify the tmpfs mount has correct Container and Pattern
		// tmpfsマウントが正しいContainerとPatternを持つことを確認
		if secretsPath.Container != "test-app" {
			t.Errorf("secretsPath.Container = %q, want \"test-app\"", secretsPath.Container)
		}
		// Pattern for tmpfs mount is the directory path
		// tmpfsマウントのPatternはディレクトリパス
		if secretsPath.Pattern != "/secrets" && secretsPath.Pattern != "/secrets/" {
			t.Errorf("secretsPath.Pattern = %q, want \"/secrets\" or \"/secrets/\"", secretsPath.Pattern)
		}
	}
}

// TestMatchPath tests the internal path matching logic.
// Verifies exact matches, filename matches, wildcard patterns, and directory patterns.
//
// TestMatchPathは内部のパスマッチングロジックをテストします。
// 完全一致、ファイル名一致、ワイルドカードパターン、ディレクトリパターンを検証します。
func TestMatchPath(t *testing.T) {
	manager := &BlockedPathsManager{}

	tests := []struct {
		name    string // Test case name / テストケース名
		path    string // Path to match / マッチするパス
		pattern string // Pattern to match against / マッチングパターン
		want    bool   // Expected result / 期待される結果
	}{
		{"exact match", "/app/.env", "/app/.env", true},          // Full path match / フルパスマッチ
		{"filename match", "/app/.env", ".env", true},            // Filename only match / ファイル名のみマッチ
		{"wildcard match", "/app/secret.key", "*.key", true},     // Extension wildcard / 拡張子ワイルドカード
		{"directory pattern", "/app/secrets/key.pem", "secrets/*", true}, // Directory/* pattern / ディレクトリ/*パターン
		{"no match", "/app/index.js", ".env", false},             // Different filename / 異なるファイル名
		{"no wildcard match", "/app/secret.txt", "*.key", false}, // Wrong extension / 間違った拡張子

		// Path boundary tests - ensure no false positives from substring matching
		// パス境界テスト - 部分文字列マッチングによる誤検知がないことを確認
		{"no false positive - app vs myapp", "/myapp/file.txt", "app/*", false},          // "app/*" should NOT match "/myapp/" / "app/*"は"/myapp/"にマッチしてはいけない
		{"no false positive - secrets vs mysecrets", "/mysecrets/key.pem", "secrets/*", false}, // "secrets/*" should NOT match "/mysecrets/" / "secrets/*"は"/mysecrets/"にマッチしてはいけない
		{"match at path start", "/app/file.txt", "app/*", true},                          // "app/*" at path start should match / パス先頭の"app/*"はマッチすべき
		{"match after slash", "/data/app/file.txt", "app/*", true},                       // "app/*" after / should match / /の後の"app/*"はマッチすべき
		{"match nested secrets", "/data/secrets/key.pem", "secrets/*", true},             // "secrets/*" nested should match / ネストされた"secrets/*"はマッチすべき
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.matchPath(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v",
					tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestExtractBlockedPath tests the path extraction logic for DevContainer mounts.
// Verifies that container names are correctly identified from mount paths.
//
// TestExtractBlockedPathはDevContainerマウント用のパス抽出ロジックをテストします。
// マウントパスからコンテナ名が正しく識別されることを検証します。
func TestExtractBlockedPath(t *testing.T) {
	manager := &BlockedPathsManager{
		containers: []string{"securenote-api", "securenote-web"},
	}

	tests := []struct {
		name          string // Test case name / テストケース名
		path          string // Mount path to extract from / 抽出元のマウントパス
		wantContainer string // Expected container / 期待されるコンテナ
		wantPattern   string // Expected pattern / 期待されるパターン
	}{
		{
			name:          "container in path",
			path:          "/workspace/demo-apps/securenote-api/.env",
			wantContainer: "securenote-api",
			wantPattern:   "/.env",
		},
		{
			name:          "no container match",
			path:          "/workspace/unknown/.env",
			wantContainer: "*",       // Falls back to global / グローバルにフォールバック
			wantPattern:   ".env",    // Just filename / ファイル名のみ
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := manager.extractBlockedPath(tt.path, "test.yml", "test")

			// Verify extraction succeeded
			// 抽出が成功したことを確認
			if blocked == nil {
				t.Fatal("extractBlockedPath returned nil")
			}

			// Verify container is correctly identified
			// コンテナが正しく識別されていることを確認
			if blocked.Container != tt.wantContainer {
				t.Errorf("Container = %q, want %q", blocked.Container, tt.wantContainer)
			}

			// Verify pattern is correctly extracted
			// パターンが正しく抽出されていることを確認
			if blocked.Pattern != tt.wantPattern {
				t.Errorf("Pattern = %q, want %q", blocked.Pattern, tt.wantPattern)
			}
		})
	}
}

// TestParseClaudeCodeSettings tests auto-import from Claude Code settings.json.
// Verifies that Read() patterns are extracted from the deny list.
//
// TestParseClaudeCodeSettingsはClaude Codeのsettings.jsonからの自動インポートをテストします。
// Read()パターンがdenyリストから抽出されることを検証します。
func TestParseClaudeCodeSettings(t *testing.T) {
	// Create temporary Claude Code settings.json for testing
	// テスト用の一時的なClaude Code settings.jsonを作成
	tmpDir := t.TempDir()
	settingsContent := `{
  "permissions": {
    "deny": [
      "Read(./.env)",
      "Read(./.env.*)",
      "Read(./secrets/**)",
      "Read(**/.env)",
      "Bash(curl:*)"
    ],
    "allow": [
      "Read(./README.md)"
    ]
  }
}`
	settingsPath := filepath.Join(tmpDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &config.BlockedPathsConfig{
		AutoImport: config.AutoImportConfig{
			Enabled:       true,
			WorkspaceRoot: tmpDir,
			ClaudeCodeSettings: config.ClaudeCodeSettingsConfig{
				Enabled:       true,
				SettingsFiles: []string{"settings.json"},
			},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{"securenote-api"})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	paths := manager.GetBlockedPaths()

	// Count paths from Claude Code settings (should be 4, not 5 - Bash is ignored)
	// Claude Code設定からのパスをカウント（5ではなく4のはず - Bashは無視）
	readPaths := 0
	for _, p := range paths {
		if p.Reason == "claude_code_settings_deny" {
			readPaths++
		}
	}

	// Should have 4 blocked paths from Read() patterns (Bash pattern is ignored)
	// Read()パターンから4つのブロックパスがあるべき（Bashパターンは無視）
	if readPaths != 4 {
		t.Errorf("expected 4 blocked paths from Claude Code settings, got %d", readPaths)
	}

	// Verify .env pattern is included
	// .envパターンが含まれていることを確認
	foundEnv := false
	for _, p := range paths {
		if p.Pattern == ".env" || p.Pattern == "**/.env" {
			foundEnv = true
			break
		}
	}
	if !foundEnv {
		t.Error("expected to find .env pattern from Claude Code settings")
	}
}

// TestConvertClaudeCodePattern tests the pattern conversion logic for Claude Code.
// Verifies that patterns are correctly parsed and container names are identified.
//
// TestConvertClaudeCodePatternはClaude Code用のパターン変換ロジックをテストします。
// パターンが正しく解析され、コンテナ名が識別されることを検証します。
func TestConvertClaudeCodePattern(t *testing.T) {
	manager := &BlockedPathsManager{
		containers: []string{"securenote-api", "securenote-web"},
	}

	tests := []struct {
		name          string // Test case name / テストケース名
		pattern       string // Input pattern / 入力パターン
		wantContainer string // Expected container / 期待されるコンテナ
		wantPattern   string // Expected output pattern / 期待される出力パターン
	}{
		{
			name:          "simple .env",
			pattern:       "./.env",
			wantContainer: "*",     // No container = global / コンテナなし = グローバル
			wantPattern:   ".env",
		},
		{
			name:          "glob pattern",
			pattern:       "**/.env",
			wantContainer: "*",
			wantPattern:   "**/.env",
		},
		{
			name:          "path with container",
			pattern:       "securenote-api/secrets/**",
			wantContainer: "securenote-api", // Container identified / コンテナ識別
			wantPattern:   "secrets/**",
		},
		{
			name:          "wildcard extension",
			pattern:       "./.env.*",
			wantContainer: "*",
			wantPattern:   ".env.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := manager.convertClaudeCodePattern(tt.pattern, "test.json")

			// Verify conversion succeeded
			// 変換が成功したことを確認
			if blocked == nil {
				t.Fatal("convertClaudeCodePattern returned nil")
			}

			// Verify container is correct
			// コンテナが正しいことを確認
			if blocked.Container != tt.wantContainer {
				t.Errorf("Container = %q, want %q", blocked.Container, tt.wantContainer)
			}

			// Verify pattern is correct
			// パターンが正しいことを確認
			if blocked.Pattern != tt.wantPattern {
				t.Errorf("Pattern = %q, want %q", blocked.Pattern, tt.wantPattern)
			}

			// Verify reason is always claude_code_settings_deny
			// reasonが常にclaude_code_settings_denyであることを確認
			if blocked.Reason != "claude_code_settings_deny" {
				t.Errorf("Reason = %q, want claude_code_settings_deny", blocked.Reason)
			}
		})
	}
}

// TestClaudeCodeSettingsMaxDepth tests the max_depth setting for Claude Code settings scanning.
// Verifies that settings files are only found at the specified depth levels.
//
// TestClaudeCodeSettingsMaxDepthはClaude Code設定スキャンのmax_depth設定をテストします。
// 設定ファイルが指定された深度レベルでのみ見つかることを検証します。
func TestClaudeCodeSettingsMaxDepth(t *testing.T) {
	// Create test directory structure:
	// テスト用ディレクトリ構造を作成:
	// tmpDir/
	// ├── .claude/settings.json           (depth 0)
	// ├── project-a/
	// │   └── .claude/settings.json       (depth 1)
	// ├── project-b/
	// │   └── .claude/settings.json       (depth 1)
	// └── project-a/subproject/
	//     └── .claude/settings.json       (depth 2)

	tmpDir := t.TempDir()

	// Create root settings (depth 0)
	// ルート設定を作成（深度0）
	rootSettings := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(rootSettings, 0755)
	os.WriteFile(filepath.Join(rootSettings, "settings.json"), []byte(`{
		"permissions": {"deny": ["Read(root-secret.txt)"]}
	}`), 0644)

	// Create project-a settings (depth 1)
	// project-a設定を作成（深度1）
	projectASettings := filepath.Join(tmpDir, "project-a", ".claude")
	os.MkdirAll(projectASettings, 0755)
	os.WriteFile(filepath.Join(projectASettings, "settings.json"), []byte(`{
		"permissions": {"deny": ["Read(project-a-secret.txt)"]}
	}`), 0644)

	// Create project-b settings (depth 1)
	// project-b設定を作成（深度1）
	projectBSettings := filepath.Join(tmpDir, "project-b", ".claude")
	os.MkdirAll(projectBSettings, 0755)
	os.WriteFile(filepath.Join(projectBSettings, "settings.json"), []byte(`{
		"permissions": {"deny": ["Read(project-b-secret.txt)"]}
	}`), 0644)

	// Create subproject settings (depth 2)
	// サブプロジェクト設定を作成（深度2）
	subprojectSettings := filepath.Join(tmpDir, "project-a", "subproject", ".claude")
	os.MkdirAll(subprojectSettings, 0755)
	os.WriteFile(filepath.Join(subprojectSettings, "settings.json"), []byte(`{
		"permissions": {"deny": ["Read(subproject-secret.txt)"]}
	}`), 0644)

	tests := []struct {
		name         string   // Test case name / テストケース名
		maxDepth     int      // Max depth to scan / スキャンする最大深度
		wantPatterns []string // Patterns that should be found / 見つかるべきパターン
		dontWant     []string // Patterns that should NOT be found / 見つかるべきでないパターン
	}{
		{
			name:         "depth 0 - only root",
			maxDepth:     0,
			wantPatterns: []string{"root-secret.txt"},
			dontWant:     []string{"project-a-secret.txt", "project-b-secret.txt", "subproject-secret.txt"},
		},
		{
			name:         "depth 1 - root and immediate subdirs",
			maxDepth:     1,
			wantPatterns: []string{"root-secret.txt", "project-a-secret.txt", "project-b-secret.txt"},
			dontWant:     []string{"subproject-secret.txt"},
		},
		{
			name:         "depth 2 - all levels",
			maxDepth:     2,
			wantPatterns: []string{"root-secret.txt", "project-a-secret.txt", "project-b-secret.txt", "subproject-secret.txt"},
			dontWant:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.BlockedPathsConfig{
				AutoImport: config.AutoImportConfig{
					Enabled:       true,
					WorkspaceRoot: tmpDir,
					ClaudeCodeSettings: config.ClaudeCodeSettingsConfig{
						Enabled:       true,
						MaxDepth:      tt.maxDepth,
						SettingsFiles: []string{".claude/settings.json"},
					},
				},
			}

			manager := NewBlockedPathsManager(cfg, []string{"test-container"})
			if err := manager.LoadBlockedPaths(); err != nil {
				t.Fatalf("LoadBlockedPaths failed: %v", err)
			}

			paths := manager.GetBlockedPaths()

			// Check wanted patterns are present
			// 必要なパターンが存在することを確認
			for _, wantPattern := range tt.wantPatterns {
				found := false
				for _, p := range paths {
					if p.Pattern == wantPattern {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern %q to be blocked, but it wasn't", wantPattern)
				}
			}

			// Check unwanted patterns are NOT present
			// 不要なパターンが存在しないことを確認
			for _, dontWant := range tt.dontWant {
				for _, p := range paths {
					if p.Pattern == dontWant {
						t.Errorf("did not expect pattern %q to be blocked at max_depth=%d", dontWant, tt.maxDepth)
					}
				}
			}
		})
	}
}

// TestScanSettingsAtDepth_Generic tests the unified scanSettingsAtDepth function
// with a custom parse callback, verifying depth control and directory skip behavior.
//
// TestScanSettingsAtDepth_Genericは統一されたscanSettingsAtDepth関数を
// カスタムパースコールバックでテストし、深度制御とディレクトリスキップ動作を検証します。
func TestScanSettingsAtDepth_Generic(t *testing.T) {
	// Create test directory structure:
	// tmpDir/
	// ├── project-a/
	// │   └── settings.txt       (depth 1)
	// ├── project-b/
	// │   └── settings.txt       (depth 1)
	// ├── project-a/nested/
	// │   └── settings.txt       (depth 2)
	// ├── .hidden/
	// │   └── settings.txt       (should be skipped)
	// └── node_modules/
	//     └── settings.txt       (should be skipped)
	tmpDir := t.TempDir()

	// Create depth-1 settings
	for _, dir := range []string{"project-a", "project-b"} {
		os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
		os.WriteFile(filepath.Join(tmpDir, dir, "settings.txt"), []byte("data"), 0644)
	}

	// Create depth-2 settings
	os.MkdirAll(filepath.Join(tmpDir, "project-a", "nested"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "project-a", "nested", "settings.txt"), []byte("data"), 0644)

	// Create directories that should be skipped
	for _, dir := range []string{".hidden", "node_modules", "vendor", "__pycache__"} {
		os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
		os.WriteFile(filepath.Join(tmpDir, dir, "settings.txt"), []byte("data"), 0644)
	}

	tests := []struct {
		name      string
		maxDepth  int
		wantCount int // number of files the callback should be called with
	}{
		{
			name:      "depth 1 - direct subdirs only",
			maxDepth:  1,
			wantCount: 2, // project-a, project-b (skipped dirs excluded)
		},
		{
			name:      "depth 2 - includes nested",
			maxDepth:  2,
			wantCount: 3, // project-a, project-b, project-a/nested
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.BlockedPathsConfig{}
			manager := NewBlockedPathsManager(cfg, []string{"test-container"})

			var parsedFiles []string
			parseFunc := func(filePath string) error {
				parsedFiles = append(parsedFiles, filePath)
				return nil
			}

			err := manager.scanSettingsAtDepth(tmpDir, []string{"settings.txt"}, 1, tt.maxDepth, "Test", parseFunc)
			if err != nil {
				t.Fatalf("scanSettingsAtDepth failed: %v", err)
			}

			if len(parsedFiles) != tt.wantCount {
				t.Errorf("expected %d parsed files, got %d: %v", tt.wantCount, len(parsedFiles), parsedFiles)
			}

			// Verify skipped directories are not included
			for _, f := range parsedFiles {
				for _, skip := range []string{".hidden", "node_modules", "vendor", "__pycache__"} {
					if strings.Contains(f, skip) {
						t.Errorf("expected %q to be skipped, but found in parsed files: %s", skip, f)
					}
				}
			}
		})
	}
}

// TestParseGeminiExcludeFile tests auto-import from Gemini .aiexclude files.
// Verifies that gitignore-style patterns are correctly extracted.
//
// TestParseGeminiExcludeFileはGeminiの.aiexcludeファイルからの自動インポートをテストします。
// gitignore形式のパターンが正しく抽出されることを検証します。
func TestParseGeminiExcludeFile(t *testing.T) {
	// Create temporary .aiexclude file for testing
	// テスト用の一時的な.aiexcludeファイルを作成
	tmpDir := t.TempDir()
	excludeContent := `# This is a comment
.env
*.key
*.pem
secrets/
config/credentials.json

# Negation patterns are skipped
!important.key
`
	excludePath := filepath.Join(tmpDir, ".aiexclude")
	if err := os.WriteFile(excludePath, []byte(excludeContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &config.BlockedPathsConfig{
		AutoImport: config.AutoImportConfig{
			Enabled:       true,
			WorkspaceRoot: tmpDir,
			GeminiSettings: config.GeminiSettingsConfig{
				Enabled:       true,
				SettingsFiles: []string{".aiexclude"},
			},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{"test-app"})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	paths := manager.GetBlockedPaths()

	// Count paths from Gemini exclude file (should be 5, excluding comment and negation)
	// Gemini除外ファイルからのパスをカウント（コメントと否定を除いて5つのはず）
	geminiPaths := 0
	for _, p := range paths {
		if p.Reason == "gemini_exclude_file" {
			geminiPaths++
		}
	}

	// Should have 5 blocked paths (.env, *.key, *.pem, secrets/*, config/credentials.json)
	// 5つのブロックパスがあるべき
	if geminiPaths != 5 {
		t.Errorf("expected 5 blocked paths from Gemini exclude file, got %d", geminiPaths)
	}

	// Verify .env pattern is included
	// .envパターンが含まれていることを確認
	foundEnv := false
	for _, p := range paths {
		if p.Pattern == ".env" && p.Reason == "gemini_exclude_file" {
			foundEnv = true
			break
		}
	}
	if !foundEnv {
		t.Error("expected to find .env pattern from Gemini exclude file")
	}

	// Verify negation pattern is NOT included
	// 否定パターンが含まれていないことを確認
	for _, p := range paths {
		if p.Pattern == "important.key" || p.Pattern == "!important.key" {
			t.Error("negation pattern should not be included")
		}
	}
}

// TestConvertGitignorePattern tests the pattern conversion logic for Gemini.
// Verifies that gitignore-style patterns are correctly converted to BlockedPath.
//
// TestConvertGitignorePatternはGemini用のパターン変換ロジックをテストします。
// gitignore形式のパターンがBlockedPathに正しく変換されることを検証します。
func TestConvertGitignorePattern(t *testing.T) {
	manager := &BlockedPathsManager{
		containers: []string{"test-app"},
	}

	tests := []struct {
		name          string // Test case name / テストケース名
		pattern       string // Input pattern / 入力パターン
		wantContainer string // Expected container / 期待されるコンテナ
		wantPattern   string // Expected output pattern / 期待される出力パターン
	}{
		{
			name:          "simple file",
			pattern:       ".env",
			wantContainer: "*", // Gemini patterns are always global / Geminiパターンは常にグローバル
			wantPattern:   ".env",
		},
		{
			name:          "wildcard extension",
			pattern:       "*.key",
			wantContainer: "*",
			wantPattern:   "*.key",
		},
		{
			name:          "directory pattern",
			pattern:       "secrets/",
			wantContainer: "*",
			wantPattern:   "secrets/*", // Trailing / converted to /*
		},
		{
			name:          "absolute path",
			pattern:       "/config.json",
			wantContainer: "*",
			wantPattern:   "config.json", // Leading / removed
		},
		{
			name:          "nested path",
			pattern:       "config/credentials.json",
			wantContainer: "*",
			wantPattern:   "config/credentials.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := manager.convertGitignorePattern(tt.pattern, "test.aiexclude")

			// Verify conversion succeeded
			// 変換が成功したことを確認
			if blocked == nil {
				t.Fatal("convertGitignorePattern returned nil")
			}

			// Verify container is always "*" for Gemini patterns
			// Geminiパターンのコンテナは常に"*"であることを確認
			if blocked.Container != tt.wantContainer {
				t.Errorf("Container = %q, want %q", blocked.Container, tt.wantContainer)
			}

			// Verify pattern is correct
			// パターンが正しいことを確認
			if blocked.Pattern != tt.wantPattern {
				t.Errorf("Pattern = %q, want %q", blocked.Pattern, tt.wantPattern)
			}

			// Verify reason is always gemini_exclude_file
			// reasonが常にgemini_exclude_fileであることを確認
			if blocked.Reason != "gemini_exclude_file" {
				t.Errorf("Reason = %q, want gemini_exclude_file", blocked.Reason)
			}
		})
	}
}

// TestGeminiSettingsMaxDepth tests the max_depth setting for Gemini settings scanning.
// Verifies that .aiexclude files are only found at the specified depth levels.
//
// TestGeminiSettingsMaxDepthはGemini設定スキャンのmax_depth設定をテストします。
// .aiexcludeファイルが指定された深度レベルでのみ見つかることを検証します。
func TestGeminiSettingsMaxDepth(t *testing.T) {
	// Create test directory structure:
	// テスト用ディレクトリ構造を作成:
	// tmpDir/
	// ├── .aiexclude                         (depth 0)
	// ├── project-a/
	// │   └── .aiexclude                     (depth 1)
	// ├── project-b/
	// │   └── .aiexclude                     (depth 1)
	// └── project-a/subproject/
	//     └── .aiexclude                     (depth 2)

	tmpDir := t.TempDir()

	// Create root .aiexclude (depth 0)
	// ルート.aiexcludeを作成（深度0）
	os.WriteFile(filepath.Join(tmpDir, ".aiexclude"), []byte("root-secret.txt\n"), 0644)

	// Create project-a .aiexclude (depth 1)
	// project-a .aiexcludeを作成（深度1）
	projectA := filepath.Join(tmpDir, "project-a")
	os.MkdirAll(projectA, 0755)
	os.WriteFile(filepath.Join(projectA, ".aiexclude"), []byte("project-a-secret.txt\n"), 0644)

	// Create project-b .aiexclude (depth 1)
	// project-b .aiexcludeを作成（深度1）
	projectB := filepath.Join(tmpDir, "project-b")
	os.MkdirAll(projectB, 0755)
	os.WriteFile(filepath.Join(projectB, ".aiexclude"), []byte("project-b-secret.txt\n"), 0644)

	// Create subproject .aiexclude (depth 2)
	// サブプロジェクト.aiexcludeを作成（深度2）
	subproject := filepath.Join(tmpDir, "project-a", "subproject")
	os.MkdirAll(subproject, 0755)
	os.WriteFile(filepath.Join(subproject, ".aiexclude"), []byte("subproject-secret.txt\n"), 0644)

	tests := []struct {
		name         string   // Test case name / テストケース名
		maxDepth     int      // Max depth to scan / スキャンする最大深度
		wantPatterns []string // Patterns that should be found / 見つかるべきパターン
		dontWant     []string // Patterns that should NOT be found / 見つかるべきでないパターン
	}{
		{
			name:         "depth 0 - only root",
			maxDepth:     0,
			wantPatterns: []string{"root-secret.txt"},
			dontWant:     []string{"project-a-secret.txt", "project-b-secret.txt", "subproject-secret.txt"},
		},
		{
			name:         "depth 1 - root and immediate subdirs",
			maxDepth:     1,
			wantPatterns: []string{"root-secret.txt", "project-a-secret.txt", "project-b-secret.txt"},
			dontWant:     []string{"subproject-secret.txt"},
		},
		{
			name:         "depth 2 - all levels",
			maxDepth:     2,
			wantPatterns: []string{"root-secret.txt", "project-a-secret.txt", "project-b-secret.txt", "subproject-secret.txt"},
			dontWant:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.BlockedPathsConfig{
				AutoImport: config.AutoImportConfig{
					Enabled:       true,
					WorkspaceRoot: tmpDir,
					GeminiSettings: config.GeminiSettingsConfig{
						Enabled:       true,
						MaxDepth:      tt.maxDepth,
						SettingsFiles: []string{".aiexclude"},
					},
				},
			}

			manager := NewBlockedPathsManager(cfg, []string{"test-container"})
			if err := manager.LoadBlockedPaths(); err != nil {
				t.Fatalf("LoadBlockedPaths failed: %v", err)
			}

			paths := manager.GetBlockedPaths()

			// Check wanted patterns are present
			// 必要なパターンが存在することを確認
			for _, wantPattern := range tt.wantPatterns {
				found := false
				for _, p := range paths {
					if p.Pattern == wantPattern {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern %q to be blocked, but it wasn't", wantPattern)
				}
			}

			// Check unwanted patterns are NOT present
			// 不要なパターンが存在しないことを確認
			for _, dontWant := range tt.dontWant {
				for _, p := range paths {
					if p.Pattern == dontWant {
						t.Errorf("did not expect pattern %q to be blocked at max_depth=%d", dontWant, tt.maxDepth)
					}
				}
			}
		})
	}
}

// TestParseGeminiIgnoreFile tests auto-import from .geminiignore files (Gemini CLI).
// Verifies that the same gitignore-style parsing works for .geminiignore.
//
// TestParseGeminiIgnoreFileは.geminiignoreファイル（Gemini CLI）からの自動インポートをテストします。
// .geminiignoreでも同じgitignore形式のパースが動作することを検証します。
func TestParseGeminiIgnoreFile(t *testing.T) {
	// Create temporary .geminiignore file for testing
	// テスト用の一時的な.geminiignoreファイルを作成
	tmpDir := t.TempDir()
	ignoreContent := `# Gemini CLI ignore file
.env
*.pem
private/
`
	ignorePath := filepath.Join(tmpDir, ".geminiignore")
	if err := os.WriteFile(ignorePath, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &config.BlockedPathsConfig{
		AutoImport: config.AutoImportConfig{
			Enabled:       true,
			WorkspaceRoot: tmpDir,
			GeminiSettings: config.GeminiSettingsConfig{
				Enabled:       true,
				SettingsFiles: []string{".geminiignore"},
			},
		},
	}

	manager := NewBlockedPathsManager(cfg, []string{"test-app"})
	if err := manager.LoadBlockedPaths(); err != nil {
		t.Fatalf("LoadBlockedPaths failed: %v", err)
	}

	paths := manager.GetBlockedPaths()

	// Should have 3 blocked paths (.env, *.pem, private/*)
	// 3つのブロックパスがあるべき
	geminiPaths := 0
	for _, p := range paths {
		if p.Reason == "gemini_exclude_file" {
			geminiPaths++
		}
	}

	if geminiPaths != 3 {
		t.Errorf("expected 3 blocked paths from .geminiignore, got %d", geminiPaths)
	}
}
