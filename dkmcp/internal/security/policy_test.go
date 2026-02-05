// Package security tests verify the security policy enforcement logic.
// These tests ensure that container access control, command whitelisting,
// and permission checks work correctly.
//
// securityパッケージのテストはセキュリティポリシー適用ロジックを検証します。
// これらのテストはコンテナアクセス制御、コマンドホワイトリスト、
// パーミッションチェックが正しく動作することを確認します。
package security

import (
	"strings"
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// TestNewPolicy verifies that NewPolicy creates a properly initialized policy.
// TestNewPolicyはNewPolicyが適切に初期化されたポリシーを作成することを検証します。
func TestNewPolicy(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"test-*", "demo-app"},
		ExecWhitelist: map[string][]string{
			"demo-app": {"npm test", "npm run lint"},
		},
		Permissions: config.SecurityPermissions{
			Logs:    true,
			Inspect: true,
			Stats:   true,
			Exec:    true,
		},
	}

	policy := NewPolicy(cfg)

	// Verify policy was created
	// ポリシーが作成されたことを確認
	if policy == nil {
		t.Fatal("NewPolicy returned nil")
	}

	// Verify config was stored
	// 設定が保存されたことを確認
	if policy.config != cfg {
		t.Error("Policy config not set correctly")
	}
}

// TestCanAccessContainer_AllowedPattern tests container access with glob patterns.
// Uses table-driven tests to verify various pattern matching scenarios.
//
// TestCanAccessContainer_AllowedPatternはglobパターンを使用したコンテナアクセスをテストします。
// テーブル駆動テストを使用して様々なパターンマッチングシナリオを検証します。
func TestCanAccessContainer_AllowedPattern(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"test-*", "demo-app"},
	}

	policy := NewPolicy(cfg)

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name to check / チェックするコンテナ名
		want      bool   // Expected result / 期待される結果
	}{
		{"exact match", "demo-app", true},                  // Exact container name match / 完全一致
		{"wildcard prefix match", "test-api", true},        // Matches test-* pattern / test-*パターンにマッチ
		{"wildcard prefix match 2", "test-web", true},      // Also matches test-* / test-*にもマッチ
		{"not allowed", "production-db", false},            // Not in allowed list / 許可リストにない
		{"partial match not allowed", "demo-app-old", false}, // Must be exact match / 完全一致が必要
		{"empty container", "", false},                     // Empty name not allowed / 空の名前は許可されない
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := policy.CanAccessContainer(tt.container)
			if allowed != tt.want {
				t.Errorf("CanAccessContainer(%q) = %v, want %v", tt.container, allowed, tt.want)
			}
		})
	}
}

// TestCanAccessContainer_EmptyAllowedList tests that an empty allowed list permits all containers.
// This is the default behavior when no restrictions are configured.
//
// TestCanAccessContainer_EmptyAllowedListは空の許可リストが全コンテナを許可することをテストします。
// これは制限が設定されていない場合のデフォルト動作です。
func TestCanAccessContainer_EmptyAllowedList(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "strict",
		AllowedContainers: []string{}, // Empty = allow all / 空 = 全て許可
	}

	policy := NewPolicy(cfg)

	// Any container should be allowed when list is empty
	// リストが空の場合、任意のコンテナが許可されるべき
	allowed := policy.CanAccessContainer("any-container")
	if !allowed {
		t.Error("expected container to be allowed when allowed list is empty")
	}
}

// TestCanExec_PermissionDisabled tests that exec is denied when the permission is disabled.
// Even whitelisted commands should be rejected.
//
// TestCanExec_PermissionDisabledはパーミッションが無効な場合にexecが拒否されることをテストします。
// ホワイトリストに登録されたコマンドでも拒否されるべきです。
func TestCanExec_PermissionDisabled(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "strict",
		AllowedContainers: []string{"demo-app"},
		ExecWhitelist: map[string][]string{
			"demo-app": {"npm test"},
		},
		Permissions: config.SecurityPermissions{
			Exec: false, // Exec permission disabled / Execパーミッション無効
		},
	}

	policy := NewPolicy(cfg)

	// Should fail because exec permission is disabled
	// execパーミッションが無効なため失敗するべき
	allowed, err := policy.CanExec("demo-app", "npm test")
	if err == nil {
		t.Fatal("expected error when exec permission is disabled")
	}
	if allowed {
		t.Error("expected exec to be denied when permission is disabled")
	}
}

// TestCanExec_Whitelist tests command execution with the whitelist in moderate mode.
// Verifies that only whitelisted commands are allowed.
//
// TestCanExec_Whitelistはmoderateモードでのホワイトリストを使用したコマンド実行をテストします。
// ホワイトリストに登録されたコマンドのみが許可されることを検証します。
func TestCanExec_Whitelist(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"demo-app"},
		ExecWhitelist: map[string][]string{
			"demo-app": {
				"npm test",
				"npm run lint",
				"pytest /app/tests",
			},
		},
		Permissions: config.SecurityPermissions{
			Exec: true,
		},
	}

	policy := NewPolicy(cfg)

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name / コンテナ名
		command   string // Command to execute / 実行するコマンド
		want      bool   // Should be allowed / 許可されるべきか
		wantError bool   // Should return error / エラーを返すべきか
	}{
		{"whitelisted exact match", "demo-app", "npm test", true, false},
		{"whitelisted command 2", "demo-app", "npm run lint", true, false},
		{"whitelisted command 3", "demo-app", "pytest /app/tests", true, false},
		{"not whitelisted", "demo-app", "rm -rf /", false, true},       // Dangerous command blocked / 危険なコマンドはブロック
		{"container not allowed", "other-app", "npm test", false, true}, // Wrong container / 間違ったコンテナ
		{"empty command", "demo-app", "", false, true},                  // Empty command / 空のコマンド
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := policy.CanExec(tt.container, tt.command)

			// Check error expectation
			// エラー期待値をチェック
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Check allowed result
			// 許可結果をチェック
			if allowed != tt.want {
				t.Errorf("CanExec(%q, %q) = %v, want %v", tt.container, tt.command, allowed, tt.want)
			}
		})
	}
}

// TestCanExec_NoWhitelistForContainer tests that containers without whitelists cannot exec.
// Even if the container is in the allowed list, it needs an explicit exec whitelist.
//
// TestCanExec_NoWhitelistForContainerはホワイトリストのないコンテナがexecできないことをテストします。
// コンテナが許可リストにあっても、明示的なexecホワイトリストが必要です。
func TestCanExec_NoWhitelistForContainer(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"demo-app", "test-app"},
		ExecWhitelist: map[string][]string{
			"demo-app": {"npm test"},
			// test-app has no whitelist entry
			// test-appにはホワイトリストエントリがない
		},
		Permissions: config.SecurityPermissions{
			Exec: true,
		},
	}

	policy := NewPolicy(cfg)

	// test-app is allowed to access but has no exec whitelist
	// test-appはアクセス許可されているがexecホワイトリストがない
	allowed, err := policy.CanExec("test-app", "any command")
	if err == nil {
		t.Fatal("expected error when container has no exec whitelist")
	}
	if allowed {
		t.Error("expected exec to be denied when no whitelist exists")
	}
}

// TestCanExec_PermissiveMode tests that permissive mode allows any command without whitelist.
// In permissive mode, any command can be executed as long as the container is accessible.
//
// TestCanExec_PermissiveModeはpermissiveモードでホワイトリストなしで任意のコマンドが許可されることをテストします。
// permissiveモードでは、コンテナにアクセス可能であれば任意のコマンドが実行可能です。
func TestCanExec_PermissiveMode(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "permissive",
		AllowedContainers: []string{"demo-app"},
		ExecWhitelist:     map[string][]string{}, // Empty whitelist
		Permissions: config.SecurityPermissions{
			Exec: true,
		},
	}

	policy := NewPolicy(cfg)

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name / コンテナ名
		command   string // Command to execute / 実行するコマンド
		wantOK    bool   // Expected success / 成功を期待
	}{
		{"any command allowed", "demo-app", "rm -rf /", true},
		{"arbitrary script", "demo-app", "bash -c 'echo test'", true},
		{"unlisted command", "demo-app", "custom-command --flag", true},
		{"container not accessible", "other-app", "npm test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := policy.CanExec(tt.container, tt.command)
			if tt.wantOK {
				if err != nil {
					t.Errorf("CanExec(%q, %q) error = %v, want no error", tt.container, tt.command, err)
				}
				if !allowed {
					t.Errorf("CanExec(%q, %q) = false, want true", tt.container, tt.command)
				}
			} else {
				if allowed {
					t.Errorf("CanExec(%q, %q) = true, want false", tt.container, tt.command)
				}
			}
		})
	}
}

// TestCanLogs_Permission tests the logs permission check.
// TestCanLogs_Permissionはログパーミッションチェックをテストします。
func TestCanLogs_Permission(t *testing.T) {
	tests := []struct {
		name       string // Test case name / テストケース名
		permission bool   // Permission setting / パーミッション設定
		want       bool   // Expected result / 期待される結果
	}{
		{"logs enabled", true, true},
		{"logs disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SecurityConfig{
				Mode:              "moderate",
				AllowedContainers: []string{"demo-app"},
				Permissions: config.SecurityPermissions{
					Logs: tt.permission,
				},
			}

			policy := NewPolicy(cfg)

			allowed := policy.CanGetLogs()
			if allowed != tt.want {
				t.Errorf("CanGetLogs() = %v, want %v", allowed, tt.want)
			}
		})
	}
}

// TestCanInspect_Permission tests the inspect permission check.
// TestCanInspect_Permissionはinspectパーミッションチェックをテストします。
func TestCanInspect_Permission(t *testing.T) {
	tests := []struct {
		name       string // Test case name / テストケース名
		permission bool   // Permission setting / パーミッション設定
		want       bool   // Expected result / 期待される結果
	}{
		{"inspect enabled", true, true},
		{"inspect disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SecurityConfig{
				Mode:              "moderate",
				AllowedContainers: []string{"demo-app"},
				Permissions: config.SecurityPermissions{
					Inspect: tt.permission,
				},
			}

			policy := NewPolicy(cfg)

			allowed := policy.CanInspect()
			if allowed != tt.want {
				t.Errorf("CanInspect() = %v, want %v", allowed, tt.want)
			}
		})
	}
}

// TestCanStats_Permission tests the stats permission check.
// TestCanStats_Permissionはstatsパーミッションチェックをテストします。
func TestCanStats_Permission(t *testing.T) {
	tests := []struct {
		name       string // Test case name / テストケース名
		permission bool   // Permission setting / パーミッション設定
		want       bool   // Expected result / 期待される結果
	}{
		{"stats enabled", true, true},
		{"stats disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SecurityConfig{
				Mode:              "moderate",
				AllowedContainers: []string{"demo-app"},
				Permissions: config.SecurityPermissions{
					Stats: tt.permission,
				},
			}

			policy := NewPolicy(cfg)

			allowed := policy.CanGetStats()
			if allowed != tt.want {
				t.Errorf("CanGetStats() = %v, want %v", allowed, tt.want)
			}
		})
	}
}

// TestIsCommandWhitelisted tests the internal whitelist matching logic.
// Verifies exact matches, wildcard patterns, and container-specific vs global whitelists.
//
// TestIsCommandWhitelistedは内部のホワイトリストマッチングロジックをテストします。
// 完全一致、ワイルドカードパターン、コンテナ固有とグローバルホワイトリストを検証します。
func TestIsCommandWhitelisted(t *testing.T) {
	policy := &Policy{
		config: &config.SecurityConfig{
			ExecWhitelist: map[string][]string{
				"demo-app": {
					"npm test",
					"echo *",              // Wildcard pattern / ワイルドカードパターン
					"pytest /app/tests",
				},
			},
		},
	}

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name / コンテナ名
		command   string // Command to check / チェックするコマンド
		want      bool   // Expected result / 期待される結果
	}{
		{"exact match", "demo-app", "npm test", true},
		{"exact match 2", "demo-app", "pytest /app/tests", true},
		{"wildcard match", "demo-app", "echo hello world", true}, // Matches "echo *" / "echo *"にマッチ
		{"not matched", "demo-app", "rm -rf /", false},
		{"container not in whitelist", "other-app", "npm test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := policy.isCommandWhitelisted(tt.container, tt.command)
			if got != tt.want {
				t.Errorf("isCommandWhitelisted(%q, %q) = %v, want %v",
					tt.container, tt.command, got, tt.want)
			}
		})
	}
}

// TestGetAllowedCommands tests retrieval of allowed commands for a container.
// Verifies that both container-specific and global commands are returned.
//
// TestGetAllowedCommandsはコンテナの許可コマンド取得をテストします。
// コンテナ固有とグローバルの両方のコマンドが返されることを検証します。
func TestGetAllowedCommands(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode: "moderate",
		ExecWhitelist: map[string][]string{
			"demo-app": {"npm test", "npm run lint"},
			"*":        {"echo *", "pwd"}, // Global commands / グローバルコマンド
		},
	}

	policy := NewPolicy(cfg)

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name / コンテナ名
		wantLen   int    // Expected number of commands / 期待されるコマンド数
	}{
		{"container with specific commands", "demo-app", 4}, // 2 specific + 2 global
		{"container with only default", "other-app", 2},     // 2 global only / グローバルのみ2つ
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := policy.GetAllowedCommands(tt.container)
			if len(commands) != tt.wantLen {
				t.Errorf("GetAllowedCommands(%q) returned %d commands, want %d",
					tt.container, len(commands), tt.wantLen)
			}
		})
	}
}

// TestGetSecurityPolicy tests the security policy export function.
// Verifies that all policy settings are correctly exposed.
//
// TestGetSecurityPolicyはセキュリティポリシーエクスポート関数をテストします。
// 全てのポリシー設定が正しく公開されることを検証します。
func TestGetSecurityPolicy(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"test-*", "demo-app"},
		ExecWhitelist: map[string][]string{
			"demo-app": {"npm test"},
		},
		Permissions: config.SecurityPermissions{
			Logs:    true,
			Inspect: true,
			Stats:   false, // Stats disabled / Stats無効
			Exec:    true,
		},
	}

	policy := NewPolicy(cfg)
	result := policy.GetSecurityPolicy()

	// Verify mode is correct
	// モードが正しいことを確認
	if result["mode"] != "moderate" {
		t.Errorf("mode = %v, want moderate", result["mode"])
	}

	// Verify permissions are correctly exported
	// パーミッションが正しくエクスポートされていることを確認
	permissions, ok := result["permissions"].(map[string]bool)
	if !ok {
		t.Fatal("permissions not found in result")
	}

	if permissions["stats"] != false {
		t.Error("expected stats permission to be false")
	}
}

// TestGetAllContainersWithCommands tests the container command listing function.
// Verifies that all containers with whitelists are returned including global "*".
//
// TestGetAllContainersWithCommandsはコンテナコマンドリスト関数をテストします。
// ホワイトリストを持つ全コンテナがグローバル"*"を含めて返されることを検証します。
func TestGetAllContainersWithCommands(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode: "moderate",
		ExecWhitelist: map[string][]string{
			"demo-app": {"npm test"},
			"test-app": {"pytest"},
			"*":        {"echo *"},
		},
	}

	policy := NewPolicy(cfg)
	result := policy.GetAllContainersWithCommands()

	// Should have 3 entries: demo-app, test-app, and *
	// 3つのエントリがあるべき: demo-app, test-app, *
	if len(result) != 3 {
		t.Errorf("expected 3 entries, got %d", len(result))
	}

	// Verify specific containers are present
	// 特定のコンテナが存在することを確認
	if _, ok := result["demo-app"]; !ok {
		t.Error("expected demo-app in result")
	}

	if _, ok := result["*"]; !ok {
		t.Error("expected default '*' in result")
	}
}

// TestInitBlockedPaths tests the blocked paths initialization.
// Verifies that paths are only blocked after InitBlockedPaths is called.
//
// TestInitBlockedPathsはブロックパス初期化をテストします。
// InitBlockedPathsが呼び出された後にのみパスがブロックされることを検証します。
func TestInitBlockedPaths(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode: "moderate",
		BlockedPaths: config.BlockedPathsConfig{
			Manual: map[string][]string{
				"test-app": {".env"},
			},
		},
	}

	policy := NewPolicy(cfg)

	// Before init, IsPathBlocked should return nil
	// 初期化前、IsPathBlockedはnilを返すべき
	if policy.IsPathBlocked("test-app", ".env") != nil {
		t.Error("expected nil before InitBlockedPaths")
	}

	// Initialize blocked paths with container list
	// コンテナリストでブロックパスを初期化
	if err := policy.InitBlockedPaths([]string{"test-app"}); err != nil {
		t.Fatalf("InitBlockedPaths failed: %v", err)
	}

	// After init, IsPathBlocked should detect blocked paths
	// 初期化後、IsPathBlockedはブロックパスを検出するべき
	blocked := policy.IsPathBlocked("test-app", ".env")
	if blocked == nil {
		t.Error("expected path to be blocked after InitBlockedPaths")
	}
}

// TestGetBlockedPaths tests retrieval of all blocked paths.
// Verifies that GetBlockedPaths returns nil before init and paths after init.
//
// TestGetBlockedPathsは全ブロックパスの取得をテストします。
// GetBlockedPathsが初期化前はnilを、初期化後はパスを返すことを検証します。
func TestGetBlockedPaths(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode: "moderate",
		BlockedPaths: config.BlockedPathsConfig{
			Manual: map[string][]string{
				"test-app": {".env", "secrets/*"},
			},
		},
	}

	policy := NewPolicy(cfg)

	// Before init, should return nil
	// 初期化前、nilを返すべき
	paths := policy.GetBlockedPaths()
	if paths != nil {
		t.Error("expected nil before InitBlockedPaths")
	}

	// Initialize with container list
	// コンテナリストで初期化
	if err := policy.InitBlockedPaths([]string{"test-app"}); err != nil {
		t.Fatalf("InitBlockedPaths failed: %v", err)
	}

	// After init, should return the blocked paths
	// 初期化後、ブロックパスを返すべき
	paths = policy.GetBlockedPaths()
	if len(paths) != 2 {
		t.Errorf("expected 2 blocked paths, got %d", len(paths))
	}
}

// TestExtractBaseCommand tests the base command extraction from command strings.
// TestExtractBaseCommandはコマンド文字列からのベースコマンド抽出をテストします。
func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		name    string // Test case name / テストケース名
		command string // Input command / 入力コマンド
		want    string // Expected base command / 期待されるベースコマンド
	}{
		{"simple command", "tail", "tail"},
		{"command with args", "tail -f /var/log/app.log", "tail"},
		{"command with multiple args", "grep -n error /var/log/app.log", "grep"},
		{"command with options", "ls -la /app", "ls"},
		{"empty command", "", ""},
		{"whitespace only", "   ", ""},
		{"command with leading space", "  tail -f /log", "tail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBaseCommand(tt.command)
			if got != tt.want {
				t.Errorf("extractBaseCommand(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

// TestExtractPathsFromCommand tests file path extraction from command strings.
// TestExtractPathsFromCommandはコマンド文字列からのファイルパス抽出をテストします。
func TestExtractPathsFromCommand(t *testing.T) {
	tests := []struct {
		name    string   // Test case name / テストケース名
		command string   // Input command / 入力コマンド
		want    []string // Expected paths / 期待されるパス
	}{
		{"absolute path", "tail /var/log/app.log", []string{"/var/log/app.log"}},
		{"multiple paths", "cat /etc/config.json /var/log/app.log", []string{"/etc/config.json", "/var/log/app.log"}},
		{"with options", "tail -f -n 100 /var/log/app.log", []string{"/var/log/app.log"}},
		{"relative path with slash", "cat ./config/app.yaml", []string{"./config/app.yaml"}},
		{"hidden file", "cat .env", []string{".env"}},
		{"hidden file with path", "cat /app/.gitignore", []string{"/app/.gitignore"}},
		{"no paths", "pwd", nil},
		{"only options", "ls -la", nil},
		{"empty command", "", nil},
		{"pattern argument skipped", "grep error /var/log/app.log", []string{"/var/log/app.log"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathsFromCommand(tt.command)

			// Check length first
			// まず長さをチェック
			if len(got) != len(tt.want) {
				t.Errorf("extractPathsFromCommand(%q) returned %d paths, want %d: got %v",
					tt.command, len(got), len(tt.want), got)
				return
			}

			// Check each path
			// 各パスをチェック
			for i, path := range got {
				if path != tt.want[i] {
					t.Errorf("extractPathsFromCommand(%q)[%d] = %q, want %q",
						tt.command, i, path, tt.want[i])
				}
			}
		})
	}
}

// TestCanExecDangerously_Disabled tests that dangerous mode is rejected when disabled.
// TestCanExecDangerously_Disabledは危険モードが無効な場合に拒否されることをテストします。
func TestCanExecDangerously_Disabled(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"demo-app"},
		Permissions: config.SecurityPermissions{
			Exec: true,
		},
		ExecDangerously: config.ExecDangerouslyConfig{
			Enabled: false, // Disabled / 無効
			Commands: map[string][]string{
				"demo-app": {"tail", "cat"},
			},
		},
	}

	policy := NewPolicy(cfg)

	allowed, err := policy.CanExecDangerously("demo-app", "tail /var/log/app.log")
	if err == nil {
		t.Fatal("expected error when dangerous mode is disabled")
	}
	if allowed {
		t.Error("expected exec to be denied when dangerous mode is disabled")
	}
}

// TestCanExecDangerously_Enabled tests dangerous mode execution with various scenarios.
// TestCanExecDangerously_Enabledは様々なシナリオでの危険モード実行をテストします。
func TestCanExecDangerously_Enabled(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"demo-app", "test-*"},
		Permissions: config.SecurityPermissions{
			Exec: true,
		},
		ExecDangerously: config.ExecDangerouslyConfig{
			Enabled: true,
			Commands: map[string][]string{
				"demo-app": {"tail", "cat", "grep"},
				"*":        {"tail", "ls"},
			},
		},
		BlockedPaths: config.BlockedPathsConfig{
			Manual: map[string][]string{
				"demo-app": {".env", "/secrets/*"},
			},
		},
	}

	policy := NewPolicy(cfg)

	// Initialize blocked paths
	// ブロックパスを初期化
	if err := policy.InitBlockedPaths([]string{"demo-app", "test-api"}); err != nil {
		t.Fatalf("InitBlockedPaths failed: %v", err)
	}

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name / コンテナ名
		command   string // Command to execute / 実行するコマンド
		want      bool   // Should be allowed / 許可されるべきか
		wantError bool   // Should return error / エラーを返すべきか
	}{
		// Allowed cases / 許可されるケース
		{"allowed command", "demo-app", "tail /var/log/app.log", true, false},
		{"allowed command with options", "demo-app", "tail -f -n 100 /var/log/app.log", true, false},
		{"grep with pattern", "demo-app", "grep error /var/log/app.log", true, false},
		{"cat file", "demo-app", "cat /etc/config.json", true, false},
		{"global command on other container", "test-api", "tail /var/log/app.log", true, false},
		{"ls command via global", "test-api", "ls /app", true, false},

		// Denied cases / 拒否されるケース
		{"command not in list", "demo-app", "rm /var/log/app.log", false, true},
		{"container not allowed", "production-db", "tail /var/log/app.log", false, true},
		{"blocked path", "demo-app", "cat .env", false, true},
		{"blocked path absolute", "demo-app", "cat /secrets/key.pem", false, true},
		{"pipe not allowed", "demo-app", "cat /etc/passwd | grep root", false, true},
		{"redirect not allowed", "demo-app", "tail /var/log/app.log > /tmp/out", false, true},
		{"path traversal", "demo-app", "cat ../secrets/key", false, true},
		{"empty command", "demo-app", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := policy.CanExecDangerously(tt.container, tt.command)

			// Check error expectation
			// エラー期待値をチェック
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Check allowed result
			// 許可結果をチェック
			if allowed != tt.want {
				t.Errorf("CanExecDangerously(%q, %q) = %v, want %v (error: %v)",
					tt.container, tt.command, allowed, tt.want, err)
			}
		})
	}
}

// TestCanExecDangerously_StrictMode tests that dangerous mode is blocked in strict mode.
// TestCanExecDangerously_StrictModeはstrictモードで危険モードがブロックされることをテストします。
func TestCanExecDangerously_StrictMode(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode:              "strict", // Strict mode / strictモード
		AllowedContainers: []string{"demo-app"},
		Permissions: config.SecurityPermissions{
			Exec: true,
		},
		ExecDangerously: config.ExecDangerouslyConfig{
			Enabled: true,
			Commands: map[string][]string{
				"demo-app": {"tail"},
			},
		},
	}

	policy := NewPolicy(cfg)

	allowed, err := policy.CanExecDangerously("demo-app", "tail /var/log/app.log")
	if err == nil {
		t.Fatal("expected error in strict mode")
	}
	if allowed {
		t.Error("expected exec to be denied in strict mode")
	}
}


// TestIsDangerousCommandAllowed tests the dangerous command list lookup.
// TestIsDangerousCommandAllowedは危険コマンドリストのルックアップをテストします。
func TestIsDangerousCommandAllowed(t *testing.T) {
	policy := &Policy{
		config: &config.SecurityConfig{
			ExecDangerously: config.ExecDangerouslyConfig{
				Enabled: true,
				Commands: map[string][]string{
					"demo-app": {"tail", "cat", "grep"},
					"*":        {"tail", "ls"},
				},
			},
		},
	}

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name / コンテナ名
		command   string // Base command / ベースコマンド
		want      bool   // Expected result / 期待される結果
	}{
		{"container-specific allowed", "demo-app", "cat", true},
		{"container-specific allowed 2", "demo-app", "grep", true},
		{"global allowed", "other-app", "tail", true},
		{"global allowed 2", "other-app", "ls", true},
		{"not in any list", "demo-app", "rm", false},
		{"not in global list", "other-app", "cat", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.isDangerousCommandAllowed(tt.container, tt.command)
			if got != tt.want {
				t.Errorf("isDangerousCommandAllowed(%q, %q) = %v, want %v",
					tt.container, tt.command, got, tt.want)
			}
		})
	}
}

// TestIsDangerousCommandAvailable tests that full commands (with arguments) are correctly
// checked against the dangerous commands list by extracting the base command.
//
// TestIsDangerousCommandAvailableは完全なコマンド（引数付き）がベースコマンドを
// 抽出して危険コマンドリストに対して正しくチェックされることをテストします。
func TestIsDangerousCommandAvailable(t *testing.T) {
	policy := &Policy{
		config: &config.SecurityConfig{
			ExecDangerously: config.ExecDangerouslyConfig{
				Enabled: true,
				Commands: map[string][]string{
					"demo-app": {"tail", "cat", "grep"},
					"*":        {"ls"},
				},
			},
		},
	}

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name / コンテナ名
		command   string // Full command with arguments / 引数付きの完全なコマンド
		want      bool   // Expected result / 期待される結果
	}{
		{"base command only", "demo-app", "tail", true},
		{"command with args", "demo-app", "tail -f /var/log/app.log", true},
		{"command with options", "demo-app", "grep -r pattern /app", true},
		{"global command", "other-app", "ls -la /app", true},
		{"not in list", "demo-app", "rm -rf /", false},
		{"not in global", "other-app", "cat /etc/passwd", false},
		{"empty command", "demo-app", "", false},
		{"whitespace only", "demo-app", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.isDangerousCommandAvailable(tt.container, tt.command)
			if got != tt.want {
				t.Errorf("isDangerousCommandAvailable(%q, %q) = %v, want %v",
					tt.container, tt.command, got, tt.want)
			}
		})
	}
}

// TestIsCommandWhitelistedHint tests that the error message includes a hint
// when a non-whitelisted command is available in dangerous mode.
//
// TestIsCommandWhitelistedHintはホワイトリストにないコマンドが危険モードで
// 利用可能な場合にエラーメッセージにヒントが含まれることをテストします。
func TestIsCommandWhitelistedHint(t *testing.T) {
	policy := &Policy{
		config: &config.SecurityConfig{
			Mode: "moderate",
			AllowedContainers: []string{"demo-*"},
			ExecWhitelist: map[string][]string{
				"demo-app": {"npm test"},
			},
			ExecDangerously: config.ExecDangerouslyConfig{
				Enabled: true,
				Commands: map[string][]string{
					"demo-app": {"tail", "cat"},
					"*":        {"ls"},
				},
			},
		},
	}

	tests := []struct {
		name        string // Test case name / テストケース名
		container   string // Container name / コンテナ名
		command     string // Command to check / チェックするコマンド
		wantHint    bool   // Whether hint is expected / ヒントが期待されるか
	}{
		{"whitelisted command", "demo-app", "npm test", false},
		{"dangerous command available", "demo-app", "tail /var/log/app.log", true},
		{"dangerous global command", "demo-app", "ls -la", true},
		{"not available anywhere", "demo-app", "rm -rf /", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := policy.isCommandWhitelisted(tt.container, tt.command)

			if allowed && tt.wantHint {
				t.Error("expected command to be rejected but it was allowed")
				return
			}

			if !allowed {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}

				hasHint := strings.Contains(err.Error(), "dangerously=true")
				if hasHint != tt.wantHint {
					t.Errorf("hint presence = %v, want %v, error: %s",
						hasHint, tt.wantHint, err.Error())
				}
			}
		})
	}
}

// TestGetDangerousCommandsForContainer tests retrieval of dangerous commands for a container.
// TestGetDangerousCommandsForContainerはコンテナの危険コマンド取得をテストします。
func TestGetDangerousCommandsForContainer(t *testing.T) {
	cfg := &config.SecurityConfig{
		ExecDangerously: config.ExecDangerouslyConfig{
			Enabled: true,
			Commands: map[string][]string{
				"demo-app": {"tail", "cat"},
				"*":        {"ls", "pwd"},
			},
		},
	}

	policy := NewPolicy(cfg)

	tests := []struct {
		name      string // Test case name / テストケース名
		container string // Container name / コンテナ名
		wantLen   int    // Expected number of commands / 期待されるコマンド数
	}{
		{"container with specific commands", "demo-app", 4}, // 2 specific + 2 global
		{"container with only global", "other-app", 2},      // 2 global only
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := policy.GetDangerousCommandsForContainer(tt.container)
			if len(commands) != tt.wantLen {
				t.Errorf("GetDangerousCommandsForContainer(%q) returned %d commands, want %d: %v",
					tt.container, len(commands), tt.wantLen, commands)
			}
		})
	}
}

// TestGetAllDangerousCommands tests retrieval of all dangerous commands for all containers.
// TestGetAllDangerousCommandsは全コンテナの危険コマンド取得をテストします。
func TestGetAllDangerousCommands(t *testing.T) {
	cfg := &config.SecurityConfig{
		ExecDangerously: config.ExecDangerouslyConfig{
			Enabled: true,
			Commands: map[string][]string{
				"demo-app":      {"tail", "cat"},
				"securenote-api": {"head", "grep"},
				"*":             {"ls", "pwd"},
			},
		},
	}

	policy := NewPolicy(cfg)

	result := policy.GetAllDangerousCommands()

	// Should have 3 entries: demo-app, securenote-api, *
	// 3つのエントリがあるはず: demo-app, securenote-api, *
	if len(result) != 3 {
		t.Errorf("GetAllDangerousCommands() returned %d entries, want 3", len(result))
	}

	// Check demo-app commands
	// demo-appのコマンドを確認
	if commands, ok := result["demo-app"]; ok {
		if len(commands) != 2 {
			t.Errorf("demo-app has %d commands, want 2", len(commands))
		}
	} else {
		t.Error("demo-app not found in result")
	}

	// Check securenote-api commands
	// securenote-apiのコマンドを確認
	if commands, ok := result["securenote-api"]; ok {
		if len(commands) != 2 {
			t.Errorf("securenote-api has %d commands, want 2", len(commands))
		}
	} else {
		t.Error("securenote-api not found in result")
	}

	// Check global commands
	// グローバルコマンドを確認
	if commands, ok := result["*"]; ok {
		if len(commands) != 2 {
			t.Errorf("* (global) has %d commands, want 2", len(commands))
		}
	} else {
		t.Error("* (global) not found in result")
	}
}

// TestSetDangerousModeEnabled tests runtime enabling/disabling of dangerous mode.
// TestSetDangerousModeEnabledは危険モードの実行時有効化/無効化をテストします。
func TestSetDangerousModeEnabled(t *testing.T) {
	cfg := &config.SecurityConfig{
		ExecDangerously: config.ExecDangerouslyConfig{
			Enabled: false,
		},
	}

	policy := NewPolicy(cfg)

	// Initially disabled
	// 初期状態は無効
	if policy.IsDangerousModeEnabled() {
		t.Error("expected dangerous mode to be initially disabled")
	}

	// Enable at runtime
	// 実行時に有効化
	policy.SetDangerousModeEnabled(true)
	if !policy.IsDangerousModeEnabled() {
		t.Error("expected dangerous mode to be enabled after SetDangerousModeEnabled(true)")
	}

	// Disable again
	// 再度無効化
	policy.SetDangerousModeEnabled(false)
	if policy.IsDangerousModeEnabled() {
		t.Error("expected dangerous mode to be disabled after SetDangerousModeEnabled(false)")
	}
}

// TestSetDangerousCommands tests runtime setting of dangerous commands.
// TestSetDangerousCommandsは危険コマンドの実行時設定をテストします。
func TestSetDangerousCommands(t *testing.T) {
	cfg := &config.SecurityConfig{
		ExecDangerously: config.ExecDangerouslyConfig{
			Enabled:  true,
			Commands: nil, // Empty initially / 初期状態は空
		},
	}

	policy := NewPolicy(cfg)

	// Initially no commands
	// 初期状態はコマンドなし
	commands := policy.GetDangerousCommandsForContainer("test-app")
	if len(commands) != 0 {
		t.Errorf("expected no commands initially, got %d", len(commands))
	}

	// Set commands for specific container
	// 特定のコンテナにコマンドを設定
	policy.SetDangerousCommands("test-app", []string{"tail", "cat"})
	commands = policy.GetDangerousCommandsForContainer("test-app")
	if len(commands) != 2 {
		t.Errorf("expected 2 commands after SetDangerousCommands, got %d", len(commands))
	}

	// Set global commands
	// グローバルコマンドを設定
	policy.SetDangerousCommands("*", []string{"ls"})
	commands = policy.GetDangerousCommandsForContainer("other-app")
	if len(commands) != 1 {
		t.Errorf("expected 1 global command, got %d", len(commands))
	}
}

// TestMaskHostPaths tests host path masking functionality.
// TestMaskHostPathsはホストパスマスキング機能をテストします。
func TestMaskHostPaths(t *testing.T) {
	tests := []struct {
		name     string // Test case name / テストケース名
		enabled  bool   // Whether masking is enabled / マスキング有効かどうか
		input    string // Input string / 入力文字列
		expected string // Expected output / 期待される出力
	}{
		{
			name:     "macOS path",
			enabled:  true,
			input:    `/Users/john/workspace/project/file.txt`,
			expected: `[HOST_PATH]/workspace/project/file.txt`,
		},
		{
			name:     "Linux path",
			enabled:  true,
			input:    `/home/developer/projects/app/src`,
			expected: `[HOST_PATH]/projects/app/src`,
		},
		{
			name:     "JSON with paths",
			enabled:  true,
			input:    `{"Source": "/Users/john/workspace/demo-apps/securenote-api/.env"}`,
			expected: `{"Source": "[HOST_PATH]/workspace/demo-apps/securenote-api/.env"}`,
		},
		{
			name:     "Multiple paths",
			enabled:  true,
			input:    `"/Users/john/a" and "/home/jane/b"`,
			expected: `"[HOST_PATH]/a" and "[HOST_PATH]/b"`,
		},
		{
			name:     "Disabled masking",
			enabled:  false,
			input:    `/Users/john/workspace/project`,
			expected: `/Users/john/workspace/project`,
		},
		{
			name:     "No path to mask",
			enabled:  true,
			input:    `Just some text without paths`,
			expected: `Just some text without paths`,
		},
		{
			name:     "Path without subdirectory",
			enabled:  true,
			input:    `/Users/john`,
			expected: `[HOST_PATH]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SecurityConfig{
				Mode: "moderate",
				HostPathMasking: config.HostPathMaskingConfig{
					Enabled:     tt.enabled,
					Replacement: "[HOST_PATH]",
				},
			}

			policy := NewPolicy(cfg)
			result := policy.MaskHostPaths(tt.input)

			if result != tt.expected {
				t.Errorf("MaskHostPaths() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestMaskHostPaths_Windows tests Windows path masking.
// TestMaskHostPaths_WindowsはWindowsパスマスキングをテストします。
func TestMaskHostPaths_Windows(t *testing.T) {
	tests := []struct {
		name     string // Test case name / テストケース名
		input    string // Input string / 入力文字列
		expected string // Expected output / 期待される出力
	}{
		{
			name:     "Windows path with backslash",
			input:    `C:\Users\john\Documents\project`,
			expected: `[HOST_PATH]\Documents\project`,
		},
		{
			name:     "Windows path with forward slash",
			input:    `C:/Users/jane/workspace/app`,
			expected: `[HOST_PATH]/workspace/app`,
		},
		{
			name:     "Windows path lowercase drive",
			input:    `c:\Users\developer\code`,
			expected: `[HOST_PATH]\code`,
		},
		{
			name:     "Multiple Windows paths",
			input:    `"C:\Users\john\a" and "D:\Users\jane\b"`,
			expected: `"[HOST_PATH]\a" and "[HOST_PATH]\b"`,
		},
		{
			name:     "Windows path without subdirectory",
			input:    `C:\Users\john`,
			expected: `[HOST_PATH]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SecurityConfig{
				Mode: "moderate",
				HostPathMasking: config.HostPathMaskingConfig{
					Enabled:     true,
					Replacement: "[HOST_PATH]",
				},
			}

			policy := NewPolicy(cfg)
			result := policy.MaskHostPaths(tt.input)

			if result != tt.expected {
				t.Errorf("MaskHostPaths() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestMaskHostPaths_CustomReplacement tests custom replacement string.
// TestMaskHostPaths_CustomReplacementはカスタム置換文字列をテストします。
func TestMaskHostPaths_CustomReplacement(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode: "moderate",
		HostPathMasking: config.HostPathMaskingConfig{
			Enabled:     true,
			Replacement: "[HIDDEN]",
		},
	}

	policy := NewPolicy(cfg)
	input := `/Users/john/workspace/project`
	expected := `[HIDDEN]/workspace/project`

	result := policy.MaskHostPaths(input)
	if result != expected {
		t.Errorf("MaskHostPaths() = %q, want %q", result, expected)
	}
}

// TestMaskHostPaths_DefaultReplacement tests default replacement when not configured.
// TestMaskHostPaths_DefaultReplacementは未設定時のデフォルト置換をテストします。
func TestMaskHostPaths_DefaultReplacement(t *testing.T) {
	cfg := &config.SecurityConfig{
		Mode: "moderate",
		HostPathMasking: config.HostPathMaskingConfig{
			Enabled:     true,
			Replacement: "", // Empty = use default
		},
	}

	policy := NewPolicy(cfg)
	input := `/home/user/project`
	expected := `[HOST_PATH]/project`

	result := policy.MaskHostPaths(input)
	if result != expected {
		t.Errorf("MaskHostPaths() with empty replacement = %q, want %q", result, expected)
	}
}

// TestMaskPathWithSeparators tests the shared helper function for path masking.
// Verifies correct behavior with different separators and terminators.
//
// TestMaskPathWithSeparatorsはパスマスキングの共通ヘルパー関数をテストします。
// 異なる区切り文字とターミネータでの正しい動作を検証します。
func TestMaskPathWithSeparators(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		prefix      string
		replacement string
		separators  string
		terminators string
		want        string
	}{
		{
			name:        "Unix path with / separator",
			input:       "/Users/john/workspace/project",
			prefix:      "/Users/",
			replacement: "[MASKED]",
			separators:  "/",
			terminators: pathTerminators,
			want:        "[MASKED]/workspace/project",
		},
		{
			name:        "Windows path with \\ separator",
			input:       `C:\Users\john\Documents\file.txt`,
			prefix:      `C:\Users\`,
			replacement: "[MASKED]",
			separators:  "\\/",
			terminators: pathTerminators,
			want:        `[MASKED]\Documents\file.txt`,
		},
		{
			name:        "path terminated by space",
			input:       "/home/user/path value",
			prefix:      "/home/",
			replacement: "[MASKED]",
			separators:  "/",
			terminators: pathTerminators,
			want:        "[MASKED]/path value",
		},
		{
			name:        "path terminated by double quote",
			input:       `"/home/user/project"`,
			prefix:      "/home/",
			replacement: "[MASKED]",
			separators:  "/",
			terminators: pathTerminators,
			want:        `"[MASKED]/project"`,
		},
		{
			name:        "multiple occurrences",
			input:       "/home/alice/a /home/bob/b",
			prefix:      "/home/",
			replacement: "[X]",
			separators:  "/",
			terminators: pathTerminators,
			want:        "[X]/a [X]/b",
		},
		{
			name:        "prefix not found",
			input:       "no matching path here",
			prefix:      "/Users/",
			replacement: "[MASKED]",
			separators:  "/",
			terminators: pathTerminators,
			want:        "no matching path here",
		},
		{
			name:        "tab terminates path",
			input:       "/home/user\tnext",
			prefix:      "/home/",
			replacement: "[MASKED]",
			separators:  "/",
			terminators: pathTerminators,
			want:        "[MASKED]\tnext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskPathWithSeparators(tt.input, tt.prefix, tt.replacement, tt.separators, tt.terminators)
			if got != tt.want {
				t.Errorf("maskPathWithSeparators() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestIsHostPathMaskingEnabled tests the enabled check.
// TestIsHostPathMaskingEnabledは有効チェックをテストします。
func TestIsHostPathMaskingEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    bool
	}{
		{"enabled", true, true},
		{"disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SecurityConfig{
				Mode: "moderate",
				HostPathMasking: config.HostPathMaskingConfig{
					Enabled: tt.enabled,
				},
			}

			policy := NewPolicy(cfg)
			if got := policy.IsHostPathMaskingEnabled(); got != tt.want {
				t.Errorf("IsHostPathMaskingEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseCommandArgs tests the command argument parsing with quotes and escapes.
// TestParseCommandArgsは引用符とエスケープを使用したコマンド引数解析をテストします。
func TestParseCommandArgs(t *testing.T) {
	tests := []struct {
		name    string   // Test case name / テストケース名
		command string   // Input command / 入力コマンド
		want    []string // Expected arguments / 期待される引数
	}{
		// Simple cases / シンプルなケース
		{"empty command", "", nil},
		{"single word", "echo", []string{"echo"}},
		{"simple command", "tail -f /var/log/app.log", []string{"tail", "-f", "/var/log/app.log"}},
		{"multiple spaces", "cat   file.txt", []string{"cat", "file.txt"}},
		{"leading/trailing spaces", "  npm test  ", []string{"npm", "test"}},

		// Double quoted strings / ダブルクォート文字列
		{"double quoted arg", `echo "hello world"`, []string{"echo", "hello world"}},
		{"double quoted with spaces", `grep "error message" file`, []string{"grep", "error message", "file"}},
		{"multiple double quoted", `echo "foo bar" "baz qux"`, []string{"echo", "foo bar", "baz qux"}},
		{"empty double quoted", `echo "" file`, []string{"echo", "", "file"}},

		// Single quoted strings / シングルクォート文字列
		{"single quoted arg", `echo 'hello world'`, []string{"echo", "hello world"}},
		{"single quoted with spaces", `grep 'error message' file`, []string{"grep", "error message", "file"}},
		{"multiple single quoted", `echo 'foo bar' 'baz qux'`, []string{"echo", "foo bar", "baz qux"}},

		// Mixed quotes / 混合クォート
		{"mixed quotes", `echo 'single' "double"`, []string{"echo", "single", "double"}},
		{"adjacent quoted", `echo "foo"'bar'`, []string{"echo", "foobar"}},

		// Escaped characters / エスケープ文字
		{"escaped space", `echo hello\ world`, []string{"echo", "hello world"}},
		{"escaped backslash", `echo foo\\bar`, []string{"echo", `foo\bar`}},
		{"escaped quote", `echo "hello \"world\""`, []string{"echo", `hello "world"`}},
		{"escaped single in double", `echo "it's fine"`, []string{"echo", "it's fine"}},

		// Paths with spaces / スペース付きパス
		{"quoted path", `cat "/path/with spaces/file.txt"`, []string{"cat", "/path/with spaces/file.txt"}},
		{"escaped path", `cat /path/with\ spaces/file.txt`, []string{"cat", "/path/with spaces/file.txt"}},

		// Complex cases / 複雑なケース
		{"json in quotes", `curl -d '{"key":"value"}'`, []string{"curl", "-d", `{"key":"value"}`}},
		{"grep pattern", `grep "error\|warn" /var/log/app.log`, []string{"grep", `error\|warn`, "/var/log/app.log"}},
		{"tar command", `tar -czvf "backup file.tar.gz" dir1 dir2`, []string{"tar", "-czvf", "backup file.tar.gz", "dir1", "dir2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommandArgs(tt.command)

			// Check length
			// 長さをチェック
			if len(got) != len(tt.want) {
				t.Errorf("parseCommandArgs(%q) returned %d args, want %d: got %v, want %v",
					tt.command, len(got), len(tt.want), got, tt.want)
				return
			}

			// Check each argument
			// 各引数をチェック
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseCommandArgs(%q)[%d] = %q, want %q",
						tt.command, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestExtractPathsFromCommand_QuotedPaths tests path extraction with quoted paths.
// TestExtractPathsFromCommand_QuotedPathsは引用符付きパスのパス抽出をテストします。
func TestExtractPathsFromCommand_QuotedPaths(t *testing.T) {
	tests := []struct {
		name    string   // Test case name / テストケース名
		command string   // Input command / 入力コマンド
		want    []string // Expected paths / 期待されるパス
	}{
		{"quoted path", `cat "/var/log/my app/error.log"`, []string{"/var/log/my app/error.log"}},
		{"escaped space path", `tail /var/log/my\ app/error.log`, []string{"/var/log/my app/error.log"}},
		{"multiple quoted paths", `cat "/etc/config 1.json" "/etc/config 2.json"`, []string{"/etc/config 1.json", "/etc/config 2.json"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathsFromCommand(tt.command)

			if len(got) != len(tt.want) {
				t.Errorf("extractPathsFromCommand(%q) returned %d paths, want %d: got %v",
					tt.command, len(got), len(tt.want), got)
				return
			}

			for i, path := range got {
				if path != tt.want[i] {
					t.Errorf("extractPathsFromCommand(%q)[%d] = %q, want %q",
						tt.command, i, path, tt.want[i])
				}
			}
		})
	}
}

// TestExtractBaseCommand_QuotedCommand tests base command extraction with quotes.
// TestExtractBaseCommand_QuotedCommandは引用符付きコマンドのベースコマンド抽出をテストします。
func TestExtractBaseCommand_QuotedCommand(t *testing.T) {
	tests := []struct {
		name    string // Test case name / テストケース名
		command string // Input command / 入力コマンド
		want    string // Expected base command / 期待されるベースコマンド
	}{
		{"simple", "tail", "tail"},
		{"with args", `tail -f "/var/log/app.log"`, "tail"},
		{"quoted command path", `"/usr/bin/cat" /etc/passwd`, "/usr/bin/cat"},
		{"with quoted args", `grep "pattern" /file`, "grep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBaseCommand(tt.command)
			if got != tt.want {
				t.Errorf("extractBaseCommand(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}
