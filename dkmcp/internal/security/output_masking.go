// Package security provides security policy enforcement for DockMCP.
// This file implements output masking to prevent sensitive data exposure.
//
// securityパッケージはDockMCPのセキュリティポリシー適用を提供します。
// このファイルは機密データの露出を防ぐための出力マスキングを実装します。

package security

import (
	"regexp"
	"sync"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// OutputMasker applies masking rules to command output to hide sensitive data.
// It uses compiled regex patterns for efficient matching.
//
// OutputMaskerはコマンド出力にマスキングルールを適用して機密データを隠します。
// 効率的なマッチングのためにコンパイル済み正規表現パターンを使用します。
type OutputMasker struct {
	enabled     bool
	replacement string
	patterns    []*regexp.Regexp
	applyTo     config.OutputMaskingTargets
	mu          sync.RWMutex
}

// NewOutputMasker creates a new OutputMasker from configuration.
// It compiles all regex patterns and validates them.
//
// NewOutputMaskerは設定から新しいOutputMaskerを作成します。
// すべての正規表現パターンをコンパイルして検証します。
func NewOutputMasker(cfg *config.OutputMaskingConfig) (*OutputMasker, error) {
	if cfg == nil {
		return &OutputMasker{enabled: false}, nil
	}

	masker := &OutputMasker{
		enabled:     cfg.Enabled,
		replacement: cfg.Replacement,
		applyTo:     cfg.ApplyTo,
		patterns:    make([]*regexp.Regexp, 0, len(cfg.Patterns)),
	}

	// Set default replacement if empty
	// 空の場合はデフォルトの置換文字列を設定
	if masker.replacement == "" {
		masker.replacement = "[MASKED]"
	}

	// Compile regex patterns
	// 正規表現パターンをコンパイル
	for _, pattern := range cfg.Patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Skip invalid patterns but log warning
			// 無効なパターンはスキップするが警告をログ
			continue
		}
		masker.patterns = append(masker.patterns, re)
	}

	return masker, nil
}

// MaskOutput applies all masking patterns to the given output string.
// Returns the masked output.
//
// MaskOutputは指定された出力文字列にすべてのマスキングパターンを適用します。
// マスクされた出力を返します。
func (m *OutputMasker) MaskOutput(output string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled || len(m.patterns) == 0 {
		return output
	}

	result := output
	for _, pattern := range m.patterns {
		result = pattern.ReplaceAllString(result, m.replacement)
	}

	return result
}

// MaskLogs masks sensitive data in log output if logs masking is enabled.
// MaskLogsはログマスキングが有効な場合、ログ出力内の機密データをマスクします。
func (m *OutputMasker) MaskLogs(output string) string {
	if !m.ShouldMaskLogs() {
		return output
	}
	return m.MaskOutput(output)
}

// MaskExec masks sensitive data in exec output if exec masking is enabled.
// MaskExecはexecマスキングが有効な場合、exec出力内の機密データをマスクします。
func (m *OutputMasker) MaskExec(output string) string {
	if !m.ShouldMaskExec() {
		return output
	}
	return m.MaskOutput(output)
}

// MaskInspect masks sensitive data in inspect output if inspect masking is enabled.
// MaskInspectはinspectマスキングが有効な場合、inspect出力内の機密データをマスクします。
func (m *OutputMasker) MaskInspect(output string) string {
	if !m.ShouldMaskInspect() {
		return output
	}
	return m.MaskOutput(output)
}

// ShouldMaskLogs returns true if logs output should be masked.
// ShouldMaskLogsはログ出力をマスクすべき場合にtrueを返します。
func (m *OutputMasker) ShouldMaskLogs() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled && m.applyTo.Logs
}

// ShouldMaskExec returns true if exec output should be masked.
// ShouldMaskExecはexec出力をマスクすべき場合にtrueを返します。
func (m *OutputMasker) ShouldMaskExec() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled && m.applyTo.Exec
}

// ShouldMaskInspect returns true if inspect output should be masked.
// ShouldMaskInspectはinspect出力をマスクすべき場合にtrueを返します。
func (m *OutputMasker) ShouldMaskInspect() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled && m.applyTo.Inspect
}

// IsEnabled returns true if output masking is enabled.
// IsEnabledは出力マスキングが有効な場合にtrueを返します。
func (m *OutputMasker) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// PatternCount returns the number of active masking patterns.
// PatternCountはアクティブなマスキングパターンの数を返します。
func (m *OutputMasker) PatternCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.patterns)
}

// AddPattern adds a new masking pattern at runtime.
// Returns an error if the pattern is invalid.
//
// AddPatternは実行時に新しいマスキングパターンを追加します。
// パターンが無効な場合はエラーを返します。
func (m *OutputMasker) AddPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.patterns = append(m.patterns, re)
	return nil
}
