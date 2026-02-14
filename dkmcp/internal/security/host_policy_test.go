// host_policy_test.go contains tests for host command policy enforcement.
// These tests verify command whitelisting, dangerous mode, container restrictions,
// and pipe/redirect rejection for host OS command execution.
//
// host_policy_test.goはホストコマンドポリシーの適用テストを含みます。
// コマンドホワイトリスト、危険モード、コンテナ制限、
// パイプ/リダイレクト拒否の検証を行います。
package security

import (
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// newTestHostConfig creates a test configuration for host command policy tests.
// It includes basic whitelisted commands (docker, git, df, free), denied commands,
// and dangerous mode with lifecycle commands enabled.
//
// newTestHostConfigはホストコマンドポリシーテスト用の設定を作成します。
// 基本的なホワイトリストコマンド（docker、git、df、free）、拒否コマンド、
// ライフサイクルコマンドを有効にした危険モードを含みます。
func newTestHostConfig() *config.HostCommandsConfig {
	return &config.HostCommandsConfig{
		Enabled:           true,
		AllowedContainers: []string{"securenote-*", "demo-*"},
		AllowedProjects:   []string{"demo-apps"},
		Whitelist: map[string][]string{
			"docker": {"ps", "logs *", "stats *", "inspect *"},
			"git":    {"status", "diff *", "log --oneline *"},
			"df":     {"-h"},
			"free":   {"-m"},
		},
		Deny: map[string][]string{
			"docker": {"rm *", "rmi *", "system prune *"},
		},
		Dangerously: config.HostCommandsDangerously{
			Enabled: true,
			Commands: map[string][]string{
				"docker": {"restart", "stop", "start"},
				"git":    {"checkout", "pull"},
			},
		},
	}
}

// --- Normal mode tests ---

// TestHostCommandPolicy_NormalMode_Allowed verifies that whitelisted commands
// are accepted in normal mode. Tests docker, git, df, and free commands
// with various arguments matching the whitelist patterns.
//
// TestHostCommandPolicy_NormalMode_Allowedは通常モードでホワイトリストコマンドが
// 許可されることを検証します。ホワイトリストパターンに一致する
// docker、git、df、freeコマンドを様々な引数でテストします。
func TestHostCommandPolicy_NormalMode_Allowed(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []struct {
		name    string
		command string
	}{
		{"docker ps", "docker ps"},
		{"docker logs with container", "docker logs securenote-api"},
		{"git status", "git status"},
		{"git diff", "git diff HEAD"},
		{"df -h", "df -h"},
		{"free -m", "free -m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := p.CanExecHostCommand(tt.command)
			if err != nil {
				t.Errorf("CanExecHostCommand(%q) error = %v", tt.command, err)
			}
			if !ok {
				t.Errorf("CanExecHostCommand(%q) = false, want true", tt.command)
			}
		})
	}
}

// TestHostCommandPolicy_NormalMode_Denied verifies that non-whitelisted commands
// and explicitly denied commands are rejected in normal mode.
// Tests commands not in the whitelist and those in the deny list.
//
// TestHostCommandPolicy_NormalMode_Deniedは通常モードで非ホワイトリストコマンドと
// 明示的に拒否されたコマンドが拒否されることを検証します。
// ホワイトリストに無いコマンドと拒否リストのコマンドをテストします。
func TestHostCommandPolicy_NormalMode_Denied(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []struct {
		name    string
		command string
	}{
		{"not in whitelist", "curl http://localhost"},
		{"docker rm denied", "docker rm securenote-api"},
		{"docker rmi denied", "docker rmi myimage"},
		{"docker system prune denied", "docker system prune -a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := p.CanExecHostCommand(tt.command)
			if ok {
				t.Errorf("CanExecHostCommand(%q) = true, want false", tt.command)
			}
			if err == nil {
				t.Errorf("CanExecHostCommand(%q) should return error", tt.command)
			}
		})
	}
}

// TestHostCommandPolicy_NormalMode_PipeRejected verifies that commands containing
// pipes, redirects, command separators, command substitution, or newlines are rejected
// in normal mode to prevent shell injection attacks.
//
// TestHostCommandPolicy_NormalMode_PipeRejectedは通常モードでパイプ、リダイレクト、
// コマンド区切り、コマンド置換、改行を含むコマンドがシェルインジェクション攻撃を
// 防ぐために拒否されることを検証します。
func TestHostCommandPolicy_NormalMode_PipeRejected(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []string{
		"docker ps | grep api",
		"git status > /tmp/out",
		"docker logs api; rm -rf /",
		"docker ps && echo hacked",
		"git diff $(cat /etc/passwd)",
		"git diff `cat /etc/passwd`",
		"git diff HEAD\nrm -rf /",
	}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			ok, err := p.CanExecHostCommand(cmd)
			if ok {
				t.Errorf("CanExecHostCommand(%q) should reject pipes/redirects", cmd)
			}
			if err == nil {
				t.Error("should return error")
			}
		})
	}
}

// TestHostCommandPolicy_NormalMode_HintForDangerous verifies that when a dangerous
// command is rejected in normal mode, the error message hints about using dangerous mode.
// This helps users understand that the command is available with --dangerously flag.
//
// TestHostCommandPolicy_NormalMode_HintForDangerousは通常モードで危険なコマンドが
// 拒否された際、エラーメッセージに危険モードの使用をヒントすることを検証します。
// これにより、--dangerouslyフラグでコマンドが利用可能であることをユーザーに伝えます。
func TestHostCommandPolicy_NormalMode_HintForDangerous(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	_, err := p.CanExecHostCommand("docker restart securenote-api")
	if err == nil {
		t.Error("should return error")
	}
	// Error message should hint about dangerously mode
	if err != nil {
		errMsg := err.Error()
		if !searchSubstring(errMsg, "dangerously") {
			t.Errorf("error should mention dangerously hint, got: %s", errMsg)
		}
	}
}

// --- Dangerous mode tests ---

// TestHostCommandPolicy_DangerousMode_Allowed verifies that both dangerous commands
// (docker restart, stop, git checkout, pull) and normal whitelisted commands
// are accepted when using dangerous mode.
//
// TestHostCommandPolicy_DangerousMode_Allowedは危険モード使用時に
// 危険コマンド（docker restart、stop、git checkout、pull）と
// 通常のホワイトリストコマンドの両方が許可されることを検証します。
func TestHostCommandPolicy_DangerousMode_Allowed(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []struct {
		name    string
		command string
	}{
		{"docker restart", "docker restart securenote-api"},
		{"docker stop", "docker stop securenote-api"},
		{"git checkout", "git checkout main"},
		{"git pull", "git pull origin main"},
		// Normal whitelist commands should also work in dangerous mode
		{"docker ps via dangerous", "docker ps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := p.CanExecHostCommandDangerously(tt.command)
			if err != nil {
				t.Errorf("CanExecHostCommandDangerously(%q) error = %v", tt.command, err)
			}
			if !ok {
				t.Errorf("CanExecHostCommandDangerously(%q) = false, want true", tt.command)
			}
		})
	}
}

// TestHostCommandPolicy_DangerousMode_Denied verifies that commands not in either
// the whitelist or dangerous list are still rejected in dangerous mode.
// Dangerous mode expands allowed commands but does not permit arbitrary execution.
//
// TestHostCommandPolicy_DangerousMode_Deniedはホワイトリストにも危険リストにも
// 無いコマンドが危険モードでも拒否されることを検証します。
// 危険モードは許可コマンドを拡張しますが、任意のコマンド実行は許可しません。
func TestHostCommandPolicy_DangerousMode_Denied(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	tests := []struct {
		name    string
		command string
	}{
		{"not in any list", "curl http://localhost"},
		{"docker rm still denied", "docker rm securenote-api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := p.CanExecHostCommandDangerously(tt.command)
			if ok {
				t.Errorf("CanExecHostCommandDangerously(%q) = true, want false", tt.command)
			}
			if err == nil {
				t.Errorf("CanExecHostCommandDangerously(%q) should return error", tt.command)
			}
		})
	}
}

// TestHostCommandPolicy_DangerousMode_Disabled verifies that dangerous commands
// are rejected when dangerous mode is disabled in the configuration.
// This ensures the dangerously.enabled setting is respected.
//
// TestHostCommandPolicy_DangerousMode_Disabledは設定で危険モードが無効の場合に
// 危険コマンドが拒否されることを検証します。
// dangerously.enabled設定が尊重されることを確認します。
func TestHostCommandPolicy_DangerousMode_Disabled(t *testing.T) {
	cfg := newTestHostConfig()
	cfg.Dangerously.Enabled = false
	p := NewHostCommandPolicy(cfg)

	_, err := p.CanExecHostCommandDangerously("docker restart securenote-api")
	if err == nil {
		t.Error("should return error when dangerous mode is disabled")
	}
}

// TestHostCommandPolicy_DangerousMode_PathTraversal verifies that path traversal
// attempts (../) are rejected even in dangerous mode to prevent unauthorized
// file system access.
//
// TestHostCommandPolicy_DangerousMode_PathTraversalは危険モードでも
// パストラバーサル（../）が不正なファイルシステムアクセスを防ぐために
// 拒否されることを検証します。
func TestHostCommandPolicy_DangerousMode_PathTraversal(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	_, err := p.CanExecHostCommandDangerously("git checkout ../../etc/passwd")
	if err == nil {
		t.Error("should reject path traversal in dangerous mode")
	}
}

// --- Disabled tests ---

// TestHostCommandPolicy_Disabled verifies that all host commands are rejected
// when host command execution is disabled in the configuration.
// Both normal and dangerous mode should fail when enabled=false.
//
// TestHostCommandPolicy_Disabledは設定でホストコマンド実行が無効の場合に
// すべてのホストコマンドが拒否されることを検証します。
// enabled=falseの場合、通常モードと危険モードの両方が失敗すべきです。
func TestHostCommandPolicy_Disabled(t *testing.T) {
	cfg := newTestHostConfig()
	cfg.Enabled = false
	p := NewHostCommandPolicy(cfg)

	_, err := p.CanExecHostCommand("docker ps")
	if err == nil {
		t.Error("should return error when host commands are disabled")
	}

	_, err = p.CanExecHostCommandDangerously("docker ps")
	if err == nil {
		t.Error("should return error when host commands are disabled")
	}
}

// --- GetAllowedHostCommands ---

// TestHostCommandPolicy_GetAllowedHostCommands verifies that GetAllowedHostCommands
// returns the complete whitelist mapping from the configuration.
// Tests that all expected commands (docker, git, df, free) are included.
//
// TestHostCommandPolicy_GetAllowedHostCommandsはGetAllowedHostCommandsが
// 設定からホワイトリストマッピング全体を返すことを検証します。
// 期待されるすべてのコマンド（docker、git、df、free）が含まれることをテストします。
func TestHostCommandPolicy_GetAllowedHostCommands(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())
	cmds := p.GetAllowedHostCommands()

	if len(cmds) != 4 {
		t.Errorf("GetAllowedHostCommands() returned %d commands, want 4", len(cmds))
	}

	if _, ok := cmds["docker"]; !ok {
		t.Error("should include docker commands")
	}
	if _, ok := cmds["git"]; !ok {
		t.Error("should include git commands")
	}
}

// --- Wildcard pattern tests ---

// TestHostCommandPolicy_WildcardPatterns verifies that wildcard patterns (*)
// in the whitelist correctly match arbitrary arguments.
// Tests "logs *" and "inspect *" patterns with container names.
//
// TestHostCommandPolicy_WildcardPatternsはホワイトリストのワイルドカードパターン（*）が
// 任意の引数に正しくマッチすることを検証します。
// コンテナ名を使った「logs *」と「inspect *」パターンをテストします。
func TestHostCommandPolicy_WildcardPatterns(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	// "logs *" should match any container name
	ok, err := p.CanExecHostCommand("docker logs securenote-api")
	if err != nil || !ok {
		t.Errorf("should match 'logs *' pattern: ok=%v, err=%v", ok, err)
	}

	// "inspect *" should match any container name
	ok, err = p.CanExecHostCommand("docker inspect securenote-api")
	if err != nil || !ok {
		t.Errorf("should match 'inspect *' pattern: ok=%v, err=%v", ok, err)
	}
}

// --- Container restriction tests ---

// TestHostCommandPolicy_ContainerRestrictions verifies that container name restrictions
// are enforced for docker commands. Only containers matching allowed_containers patterns
// should be accessible, rejecting unauthorized container names.
//
// TestHostCommandPolicy_ContainerRestrictionsはdockerコマンドに対してコンテナ名制限が
// 適用されることを検証します。allowed_containersパターンにマッチするコンテナのみが
// アクセス可能で、不正なコンテナ名は拒否されるべきです。
func TestHostCommandPolicy_ContainerRestrictions(t *testing.T) {
	p := NewHostCommandPolicy(newTestHostConfig())

	// Allowed container (matches "securenote-*")
	ok, err := p.CanExecHostCommand("docker logs securenote-api")
	if err != nil || !ok {
		t.Errorf("should allow securenote-api: ok=%v, err=%v", ok, err)
	}

	// Allowed container (matches "demo-*")
	ok, err = p.CanExecHostCommand("docker logs demo-web")
	if err != nil || !ok {
		t.Errorf("should allow demo-web: ok=%v, err=%v", ok, err)
	}

	// Disallowed container
	ok, err = p.CanExecHostCommand("docker logs malicious-container")
	if ok {
		t.Error("should reject container not in allowed list")
	}
}

// --- Helper functions ---

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
