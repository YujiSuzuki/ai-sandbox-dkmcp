// serve_test.go contains unit tests for the serve command functionality.
// It tests the applyAllowExecFlags and applyDangerouslyFlags functions.
//
// serve_test.goはserveコマンドの機能のユニットテストを含みます。
// applyAllowExecFlagsおよびapplyDangerouslyFlags関数をテストします。
package cli

import (
	"testing"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// TestApplyAllowExecFlags tests the applyAllowExecFlags function with various inputs.
// It uses table-driven tests to cover different scenarios.
//
// TestApplyAllowExecFlagsは様々な入力でapplyAllowExecFlags関数をテストします。
// 異なるシナリオをカバーするためにテーブル駆動テストを使用します。
func TestApplyAllowExecFlags(t *testing.T) {
	// Define test cases as a table.
	// テストケースをテーブルとして定義します。
	tests := []struct {
		name          string         // Test case name / テストケース名
		flags         []string       // Input flags / 入力フラグ
		initialConfig *config.Config // Initial configuration / 初期設定
		wantErr       bool           // Whether error is expected / エラーが期待されるかどうか
		validate      func(*testing.T, *config.Config)
	}{
		{
			// Test case: empty flags should not modify config.
			// テストケース：空のフラグは設定を変更しないこと。
			name:  "empty flags",
			flags: []string{},
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecWhitelist: nil,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// ExecWhitelist should remain nil when no flags provided.
				// フラグが提供されていない場合、ExecWhitelistはnilのままであるべきです。
				if cfg.Security.ExecWhitelist != nil {
					t.Error("Expected ExecWhitelist to remain nil")
				}
			},
		},
		{
			// Test case: single valid flag should create whitelist entry.
			// テストケース：単一の有効なフラグがホワイトリストエントリを作成すること。
			name:  "single valid flag",
			flags: []string{"mycontainer:npm test"},
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecWhitelist: nil,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Verify ExecWhitelist was initialized.
				// ExecWhitelistが初期化されたことを確認します。
				if cfg.Security.ExecWhitelist == nil {
					t.Fatal("Expected ExecWhitelist to be initialized")
				}
				// Verify the container entry exists.
				// コンテナエントリが存在することを確認します。
				commands, ok := cfg.Security.ExecWhitelist["mycontainer"]
				if !ok {
					t.Fatal("Expected mycontainer in ExecWhitelist")
				}
				// Verify the command was added.
				// コマンドが追加されたことを確認します。
				if len(commands) != 1 || commands[0] != "npm test" {
					t.Errorf("Expected [npm test], got %v", commands)
				}
			},
		},
		{
			// Test case: multiple flags for same container should append.
			// テストケース：同じコンテナへの複数のフラグが追加されること。
			name:  "multiple flags for same container",
			flags: []string{"mycontainer:npm test", "mycontainer:npm install"},
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecWhitelist: nil,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Verify both commands were added.
				// 両方のコマンドが追加されたことを確認します。
				commands := cfg.Security.ExecWhitelist["mycontainer"]
				if len(commands) != 2 {
					t.Fatalf("Expected 2 commands, got %d", len(commands))
				}
				if commands[0] != "npm test" || commands[1] != "npm install" {
					t.Errorf("Expected [npm test, npm install], got %v", commands)
				}
			},
		},
		{
			// Test case: flags for different containers should create separate entries.
			// テストケース：異なるコンテナへのフラグが別々のエントリを作成すること。
			name:  "multiple flags for different containers",
			flags: []string{"container1:cmd1", "container2:cmd2"},
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecWhitelist: nil,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Verify both containers were added.
				// 両方のコンテナが追加されたことを確認します。
				if len(cfg.Security.ExecWhitelist) != 2 {
					t.Fatalf("Expected 2 containers, got %d", len(cfg.Security.ExecWhitelist))
				}
				if cfg.Security.ExecWhitelist["container1"][0] != "cmd1" {
					t.Error("container1 command mismatch")
				}
				if cfg.Security.ExecWhitelist["container2"][0] != "cmd2" {
					t.Error("container2 command mismatch")
				}
			},
		},
		{
			// Test case: command containing colon should be parsed correctly.
			// テストケース：コロンを含むコマンドが正しく解析されること。
			name:  "command with colon",
			flags: []string{"mycontainer:echo foo:bar"},
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecWhitelist: nil,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Verify the command with colon was preserved.
				// Only the first colon should be used as separator.
				//
				// コロンを含むコマンドが保持されたことを確認します。
				// 最初のコロンのみがセパレータとして使用されるべきです。
				commands := cfg.Security.ExecWhitelist["mycontainer"]
				if commands[0] != "echo foo:bar" {
					t.Errorf("Expected 'echo foo:bar', got '%s'", commands[0])
				}
			},
		},
		{
			// Test case: invalid format without colon should return error.
			// テストケース：コロンなしの無効なフォーマットがエラーを返すこと。
			name:  "invalid format - no colon",
			flags: []string{"mycontainer-npm-test"},
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecWhitelist: nil,
				},
			},
			wantErr: true,
			validate: func(t *testing.T, cfg *config.Config) {
				// Config should not be modified on error.
				// エラー時に設定は変更されないべきです。
			},
		},
		{
			// Test case: only colon should return error.
			// テストケース：コロンのみがエラーを返すこと。
			name:  "invalid format - only colon",
			flags: []string{":"},
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecWhitelist: nil,
				},
			},
			wantErr: true,
			validate: func(t *testing.T, cfg *config.Config) {
				// Config should not be modified on error.
				// エラー時に設定は変更されないべきです。
			},
		},
		{
			// Test case: new flag should append to existing whitelist.
			// テストケース：新しいフラグが既存のホワイトリストに追加されること。
			name:  "append to existing whitelist",
			flags: []string{"mycontainer:new command"},
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecWhitelist: map[string][]string{
						"mycontainer": {"existing command"},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Verify both existing and new commands are present.
				// 既存のコマンドと新しいコマンドの両方が存在することを確認します。
				commands := cfg.Security.ExecWhitelist["mycontainer"]
				if len(commands) != 2 {
					t.Fatalf("Expected 2 commands, got %d", len(commands))
				}
				if commands[0] != "existing command" || commands[1] != "new command" {
					t.Errorf("Expected [existing command, new command], got %v", commands)
				}
			},
		},
	}

	// Run each test case.
	// 各テストケースを実行します。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function under test.
			// テスト対象の関数を呼び出します。
			err := applyAllowExecFlags(tt.initialConfig, tt.flags)

			// Check error expectation.
			// エラーの期待を確認します。
			if (err != nil) != tt.wantErr {
				t.Errorf("applyAllowExecFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Run validation if no error expected.
			// エラーが期待されない場合は検証を実行します。
			if !tt.wantErr {
				tt.validate(t, tt.initialConfig)
			}
		})
	}
}

// TestApplyDangerouslyFlags tests the applyDangerouslyFlags function with various inputs.
// It uses table-driven tests to cover different scenarios including:
// - Empty flags (no changes)
// - --dangerously-all flag (enables for all containers, merges with existing)
// - --dangerously=container flag (clears existing config, enables only for specified)
// - Error when both flags are used together
//
// TestApplyDangerouslyFlagsは様々な入力でapplyDangerouslyFlags関数をテストします。
// 以下のシナリオをカバーするテーブル駆動テストを使用します：
// - 空のフラグ（変更なし）
// - --dangerously-allフラグ（全コンテナに有効化、既存設定とマージ）
// - --dangerously=containerフラグ（既存設定をクリア、指定コンテナのみ有効化）
// - 両方のフラグを同時に使用した場合のエラー
func TestApplyDangerouslyFlags(t *testing.T) {
	tests := []struct {
		name           string         // Test case name / テストケース名
		dangerously    string         // --dangerously flag value / --dangerouslyフラグの値
		dangerouslyAll bool           // --dangerously-all flag value / --dangerously-allフラグの値
		initialConfig  *config.Config // Initial configuration / 初期設定
		wantErr        bool           // Whether error is expected / エラーが期待されるかどうか
		validate       func(*testing.T, *config.Config)
	}{
		{
			// Test case: no flags should not modify config.
			// テストケース：フラグなしの場合は設定を変更しない。
			name:           "no flags",
			dangerously:    "",
			dangerouslyAll: false,
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecDangerously: config.ExecDangerouslyConfig{
						Enabled: false,
						Commands: map[string][]string{
							"existing": {"tail"},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// ExecDangerously should remain unchanged.
				// ExecDangerouslyは変更されないべき。
				if cfg.Security.ExecDangerously.Enabled {
					t.Error("Expected Enabled to remain false")
				}
				if _, ok := cfg.Security.ExecDangerously.Commands["existing"]; !ok {
					t.Error("Expected existing commands to remain")
				}
			},
		},
		{
			// Test case: --dangerously-all should enable for all containers.
			// テストケース：--dangerously-allは全コンテナに対して有効化。
			name:           "dangerously-all flag",
			dangerously:    "",
			dangerouslyAll: true,
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecDangerously: config.ExecDangerouslyConfig{
						Enabled:  false,
						Commands: nil,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Dangerous mode should be enabled.
				// 危険モードが有効化されているべき。
				if !cfg.Security.ExecDangerously.Enabled {
					t.Error("Expected Enabled to be true")
				}
				// Global commands should be set.
				// グローバルコマンドが設定されているべき。
				if cmds, ok := cfg.Security.ExecDangerously.Commands["*"]; !ok {
					t.Error("Expected * (global) commands to be set")
				} else if len(cmds) == 0 {
					t.Error("Expected default commands to be added")
				}
			},
		},
		{
			// Test case: --dangerously-all should merge with existing config.
			// テストケース：--dangerously-allは既存設定とマージ。
			name:           "dangerously-all merges with existing",
			dangerously:    "",
			dangerouslyAll: true,
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecDangerously: config.ExecDangerouslyConfig{
						Enabled: false,
						Commands: map[string][]string{
							"existing-container": {"custom-cmd"},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Existing commands should be preserved.
				// 既存のコマンドが保持されているべき。
				if _, ok := cfg.Security.ExecDangerously.Commands["existing-container"]; !ok {
					t.Error("Expected existing-container commands to be preserved")
				}
				// Global commands should also be set.
				// グローバルコマンドも設定されているべき。
				if _, ok := cfg.Security.ExecDangerously.Commands["*"]; !ok {
					t.Error("Expected * (global) commands to be set")
				}
			},
		},
		{
			// Test case: --dangerously=container should clear existing and enable only for specified.
			// テストケース：--dangerously=containerは既存をクリアし指定コンテナのみ有効化。
			name:           "dangerously clears existing config",
			dangerously:    "securenote-web",
			dangerouslyAll: false,
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecDangerously: config.ExecDangerouslyConfig{
						Enabled: false,
						Commands: map[string][]string{
							"securenote-api": {"tail", "cat"},
							"*":              {"ls"},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Dangerous mode should be enabled.
				// 危険モードが有効化されているべき。
				if !cfg.Security.ExecDangerously.Enabled {
					t.Error("Expected Enabled to be true")
				}
				// Only specified container should have commands.
				// 指定されたコンテナのみがコマンドを持つべき。
				if _, ok := cfg.Security.ExecDangerously.Commands["securenote-web"]; !ok {
					t.Error("Expected securenote-web to have commands")
				}
				// Existing config should be cleared.
				// 既存の設定はクリアされているべき。
				if _, ok := cfg.Security.ExecDangerously.Commands["securenote-api"]; ok {
					t.Error("Expected securenote-api to be cleared")
				}
				if _, ok := cfg.Security.ExecDangerously.Commands["*"]; ok {
					t.Error("Expected * (global) to be cleared")
				}
			},
		},
		{
			// Test case: --dangerously with multiple containers.
			// テストケース：複数コンテナを指定した--dangerously。
			name:           "dangerously with multiple containers",
			dangerously:    "container1,container2",
			dangerouslyAll: false,
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecDangerously: config.ExecDangerouslyConfig{
						Enabled:  false,
						Commands: nil,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Both containers should have commands.
				// 両方のコンテナがコマンドを持つべき。
				if _, ok := cfg.Security.ExecDangerously.Commands["container1"]; !ok {
					t.Error("Expected container1 to have commands")
				}
				if _, ok := cfg.Security.ExecDangerously.Commands["container2"]; !ok {
					t.Error("Expected container2 to have commands")
				}
				// Should only have 2 entries.
				// 2つのエントリのみを持つべき。
				if len(cfg.Security.ExecDangerously.Commands) != 2 {
					t.Errorf("Expected 2 entries, got %d", len(cfg.Security.ExecDangerously.Commands))
				}
			},
		},
		{
			// Test case: both flags should return error.
			// テストケース：両方のフラグを指定するとエラー。
			name:           "both flags error",
			dangerously:    "container1",
			dangerouslyAll: true,
			initialConfig: &config.Config{
				Security: config.SecurityConfig{
					ExecDangerously: config.ExecDangerouslyConfig{
						Enabled:  false,
						Commands: nil,
					},
				},
			},
			wantErr: true,
			validate: func(t *testing.T, cfg *config.Config) {
				// Config should not be modified on error.
				// エラー時に設定は変更されないべき。
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyDangerouslyFlags(tt.initialConfig, tt.dangerously, tt.dangerouslyAll)

			if (err != nil) != tt.wantErr {
				t.Errorf("applyDangerouslyFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				tt.validate(t, tt.initialConfig)
			}
		})
	}
}
