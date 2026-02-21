// Run tests / テスト実行:
//   go test -v ./search-history.go ./search-history_test.go
//
// File-specific invocation is required because multiple package main
// files coexist in this directory.
// 同ディレクトリに複数の package main ファイルがあるため、ファイル指定が必要。

package main

import (
	"os"
	"testing"
)

// TestClassifyUserMessage verifies that classifyUserMessage correctly categorizes
// various user message content types.
// TestClassifyUserMessage はユーザーメッセージが正しく分類されることを検証する。
func TestClassifyUserMessage(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		want    string
	}{
		// String content / 文字列コンテンツ
		{"plain string is human", "hello world", "human"},
		{"empty string is empty", "", "empty"},
		{"nil is empty", nil, "empty"},
		{"automated string prefix", "<system-reminder>some content</system-reminder>", "automated"},
		{"ide_opened_file is automated", "<ide_opened_file>/path/to/file</ide_opened_file>", "automated"},

		// Array content: single type / 配列: 単一タイプ
		{"tool_result only", []interface{}{
			map[string]interface{}{"type": "tool_result", "content": "output"},
		}, "tool_result"},
		{"ide event only", []interface{}{
			map[string]interface{}{"type": "text", "text": "<ide_opened_file>/path</ide_opened_file>"},
		}, "ide"},
		{"ide_selection only", []interface{}{
			map[string]interface{}{"type": "text", "text": "<ide_selection>selected code</ide_selection>"},
		}, "ide"},
		{"system-reminder only", []interface{}{
			map[string]interface{}{"type": "text", "text": "<system-reminder>reminder</system-reminder>"},
		}, "system"},
		{"user-prompt-submit-hook only", []interface{}{
			map[string]interface{}{"type": "text", "text": "<user-prompt-submit-hook>hook output</user-prompt-submit-hook>"},
		}, "system"},
		{"command-name only", []interface{}{
			map[string]interface{}{"type": "text", "text": "<command-name>/commit</command-name>"},
		}, "command"},
		{"command-message only", []interface{}{
			map[string]interface{}{"type": "text", "text": "<command-message>commit msg</command-message>"},
		}, "command"},
		{"interrupt only", []interface{}{
			map[string]interface{}{"type": "text", "text": "[Request interrupted by user for new message]"},
		}, "interrupt"},
		{"human text only", []interface{}{
			map[string]interface{}{"type": "text", "text": "Please fix the bug"},
		}, "human"},

		// Array content: mixed types (priority test) / 配列: 混合タイプ（優先順位テスト）
		{"human + tool_result => human wins", []interface{}{
			map[string]interface{}{"type": "tool_result", "content": "output"},
			map[string]interface{}{"type": "text", "text": "What happened?"},
		}, "human"},
		{"human + ide => human wins", []interface{}{
			map[string]interface{}{"type": "text", "text": "<ide_opened_file>/path</ide_opened_file>"},
			map[string]interface{}{"type": "text", "text": "Fix this"},
		}, "human"},
		{"human + system => human wins", []interface{}{
			map[string]interface{}{"type": "text", "text": "<system-reminder>reminder</system-reminder>"},
			map[string]interface{}{"type": "text", "text": "Do something"},
		}, "human"},
		{"interrupt + tool_result => interrupt wins", []interface{}{
			map[string]interface{}{"type": "tool_result", "content": "output"},
			map[string]interface{}{"type": "text", "text": "[Request interrupted by user for new message]"},
		}, "interrupt"},
		{"command + ide => command wins", []interface{}{
			map[string]interface{}{"type": "text", "text": "<ide_opened_file>/path</ide_opened_file>"},
			map[string]interface{}{"type": "text", "text": "<command-name>/review</command-name>"},
		}, "command"},
		{"ide + system => ide wins", []interface{}{
			map[string]interface{}{"type": "text", "text": "<system-reminder>info</system-reminder>"},
			map[string]interface{}{"type": "text", "text": "<ide_selection>code</ide_selection>"},
		}, "ide"},

		// Edge cases / エッジケース
		{"empty array", []interface{}{}, "empty"},
		{"array with empty text", []interface{}{
			map[string]interface{}{"type": "text", "text": ""},
		}, "empty"},
		{"array with whitespace-only text", []interface{}{
			map[string]interface{}{"type": "text", "text": "   "},
		}, "empty"},
		{"non-map item in array is skipped", []interface{}{
			"not a map",
			map[string]interface{}{"type": "text", "text": "hello"},
		}, "human"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyUserMessage(tt.content)
			if got != tt.want {
				t.Errorf("classifyUserMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestIsHumanInput verifies that isHumanInput correctly identifies actual human-typed messages.
// TestIsHumanInput は実際のユーザー入力を正しく判定することを検証する。
func TestIsHumanInput(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		want    bool
	}{
		// String content / 文字列コンテンツ
		{"normal text is human", "hello", true},
		{"empty string is not human", "", false},
		{"system-reminder is not human", "<system-reminder>content</system-reminder>", false},
		{"ide_opened_file is not human", "<ide_opened_file>/path</ide_opened_file>", false},
		{"command-name is not human", "<command-name>/commit</command-name>", false},

		// Array content / 配列コンテンツ
		{"array with human text", []interface{}{
			map[string]interface{}{"type": "text", "text": "Fix the bug"},
		}, true},
		{"array with only tool_result", []interface{}{
			map[string]interface{}{"type": "tool_result", "content": "output"},
		}, false},
		{"array with only ide event", []interface{}{
			map[string]interface{}{"type": "text", "text": "<ide_opened_file>/path</ide_opened_file>"},
		}, false},
		{"array with human + automated", []interface{}{
			map[string]interface{}{"type": "text", "text": "<system-reminder>info</system-reminder>"},
			map[string]interface{}{"type": "text", "text": "Do this please"},
		}, true},
		{"array with all automated", []interface{}{
			map[string]interface{}{"type": "text", "text": "<ide_opened_file>/f</ide_opened_file>"},
			map[string]interface{}{"type": "text", "text": "<system-reminder>r</system-reminder>"},
		}, false},

		// Edge cases / エッジケース
		{"nil content", nil, false},
		{"integer content", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHumanInput(tt.content)
			if got != tt.want {
				t.Errorf("isHumanInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsAutomatedText verifies detection of all known automated text prefixes.
// TestIsAutomatedText は既知の自動生成テキストプレフィックスの検出を検証する。
func TestIsAutomatedText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		// Automated prefixes / 自動生成プレフィックス
		{"ide_opened_file", "<ide_opened_file>/path/to/file</ide_opened_file>", true},
		{"ide_selection", "<ide_selection>selected code</ide_selection>", true},
		{"system-reminder", "<system-reminder>some reminder</system-reminder>", true},
		{"user-prompt-submit-hook", "<user-prompt-submit-hook>hook</user-prompt-submit-hook>", true},
		{"command-name", "<command-name>/commit</command-name>", true},
		{"command-message", "<command-message>msg</command-message>", true},
		{"request interrupted", "[Request interrupted by user for new message]", true},

		// With leading whitespace / 先頭空白あり
		{"leading spaces + system-reminder", "  <system-reminder>r</system-reminder>", true},
		{"leading newline + ide", "\n<ide_opened_file>/f</ide_opened_file>", true},

		// Not automated / 自動生成ではない
		{"normal text", "hello world", false},
		{"empty string", "", false},
		{"partial prefix", "<ide_opened", false},
		{"text containing prefix mid-string", "some <system-reminder> text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAutomatedText(tt.text)
			if got != tt.want {
				t.Errorf("isAutomatedText(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// TestIsJapaneseLocale verifies Japanese locale detection from environment variables.
// TestIsJapaneseLocale は環境変数による日本語ロケール検出を検証する。
func TestIsJapaneseLocale(t *testing.T) {
	// Save and restore original env / 元の環境変数を保存・復元
	envKeys := []string{"LANG", "LC_ALL", "LC_MESSAGES", "LANGUAGE"}
	origValues := make(map[string]string)
	for _, key := range envKeys {
		origValues[key] = os.Getenv(key)
	}
	t.Cleanup(func() {
		for _, key := range envKeys {
			if origValues[key] != "" {
				os.Setenv(key, origValues[key])
			} else {
				os.Unsetenv(key)
			}
		}
	})

	clearLocaleEnv := func() {
		for _, key := range envKeys {
			os.Unsetenv(key)
		}
	}

	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"LANG=ja_JP.UTF-8", map[string]string{"LANG": "ja_JP.UTF-8"}, true},
		{"LANG=en_US.UTF-8", map[string]string{"LANG": "en_US.UTF-8"}, false},
		{"LC_ALL=ja_JP", map[string]string{"LC_ALL": "ja_JP"}, true},
		{"LANGUAGE=ja", map[string]string{"LANGUAGE": "ja"}, true},
		{"no locale vars set", map[string]string{}, false},
		{"LANG=C", map[string]string{"LANG": "C"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearLocaleEnv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			got := isJapaneseLocale()
			if got != tt.want {
				t.Errorf("isJapaneseLocale() = %v, want %v (env: %v)", got, tt.want, tt.env)
			}
		})
	}
}

// TestStatsCount verifies TotalUser and Total calculations.
// TestStatsCount は TotalUser と Total の計算を検証する。
func TestStatsCount(t *testing.T) {
	s := statsCount{
		HumanInput:    10,
		Interrupted:   2,
		SlashCommands: 3,
		IDEEvents:     5,
		SystemMsgs:    8,
		ToolResults:   20,
		AssistantMsgs: 50,
		Sessions:      4,
	}

	t.Run("TotalUser sums user-side categories", func(t *testing.T) {
		// TotalUser = HumanInput + Interrupted + SlashCommands + IDEEvents + SystemMsgs + ToolResults
		want := 10 + 2 + 3 + 5 + 8 + 20
		got := s.TotalUser()
		if got != want {
			t.Errorf("TotalUser() = %d, want %d", got, want)
		}
	})

	t.Run("Total sums all messages", func(t *testing.T) {
		// Total = TotalUser + AssistantMsgs
		want := 10 + 2 + 3 + 5 + 8 + 20 + 50
		got := s.Total()
		if got != want {
			t.Errorf("Total() = %d, want %d", got, want)
		}
	})

	t.Run("zero values", func(t *testing.T) {
		z := statsCount{}
		if z.TotalUser() != 0 {
			t.Errorf("zero TotalUser() = %d, want 0", z.TotalUser())
		}
		if z.Total() != 0 {
			t.Errorf("zero Total() = %d, want 0", z.Total())
		}
	})
}
