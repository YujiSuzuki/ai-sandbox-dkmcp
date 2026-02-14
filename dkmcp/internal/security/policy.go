// Package security provides security policy enforcement for DockMCP.
// It controls which containers can be accessed, what operations are allowed,
// and which commands can be executed.
//
// securityパッケージはDockMCPのセキュリティポリシー適用を提供します。
// どのコンテナにアクセスできるか、どの操作が許可されるか、
// どのコマンドを実行できるかを制御します。
//
// Security Modes (セキュリティモード):
//   - strict: Only explicitly allowed containers/commands (最も厳格)
//   - moderate: Balanced security with whitelist enforcement (バランス型)
//   - permissive: Less restrictive, more access allowed (緩和型)
package security

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// Policy handles security policy enforcement for all DockMCP operations.
// It evaluates requests against the configured security rules and determines
// whether access should be granted or denied.
//
// Policyは全てのDockMCP操作に対するセキュリティポリシー適用を処理します。
// 設定されたセキュリティルールに対してリクエストを評価し、
// アクセスを許可するか拒否するかを決定します。
type Policy struct {
	// config holds the security configuration from dkmcp.yaml
	// configはdkmcp.yamlからのセキュリティ設定を保持します
	config *config.SecurityConfig

	// blockedPathsManager handles file path blocking
	// blockedPathsManagerはファイルパスのブロックを処理します
	blockedPathsManager *BlockedPathsManager

	// outputMasker handles masking of sensitive data in output
	// outputMaskerは出力内の機密データのマスキングを処理します
	outputMasker *OutputMasker
}

// NewPolicy creates a new security policy with the given configuration.
// The policy is initialized but blocked paths are not loaded until
// InitBlockedPaths is called with the container list.
//
// NewPolicyは指定された設定で新しいセキュリティポリシーを作成します。
// ポリシーは初期化されますが、ブロックパスはコンテナリストを指定して
// InitBlockedPathsを呼び出すまでロードされません。
func NewPolicy(cfg *config.SecurityConfig) *Policy {
	// Initialize output masker
	// 出力マスカーを初期化
	masker, _ := NewOutputMasker(&cfg.OutputMasking)

	return &Policy{
		config:              cfg,
		blockedPathsManager: nil,
		outputMasker:        masker,
	}
}

// InitBlockedPaths initializes the blocked paths manager with the given container list.
// This must be called before using path blocking features.
// The container list is used to match container names in path patterns.
//
// InitBlockedPathsは指定されたコンテナリストでブロックパスマネージャを初期化します。
// パスブロック機能を使用する前にこれを呼び出す必要があります。
// コンテナリストはパスパターン内のコンテナ名のマッチングに使用されます。
func (p *Policy) InitBlockedPaths(containers []string) error {
	p.blockedPathsManager = NewBlockedPathsManager(&p.config.BlockedPaths, containers)
	return p.blockedPathsManager.LoadBlockedPaths()
}

// IsPathBlocked checks if a file path is blocked for a specific container.
// Returns BlockedPath info if blocked, nil if allowed.
// This is used to prevent AI from reading sensitive files.
//
// IsPathBlockedはファイルパスが特定のコンテナに対してブロックされているかチェックします。
// ブロックされている場合はBlockedPath情報を返し、許可されている場合はnilを返します。
// これはAIが機密ファイルを読むことを防ぐために使用されます。
func (p *Policy) IsPathBlocked(containerName string, path string) *BlockedPath {
	if p.blockedPathsManager == nil {
		return nil
	}
	return p.blockedPathsManager.IsPathBlocked(containerName, path)
}

// GetBlockedPaths returns all configured blocked paths across all containers.
// Useful for debugging and displaying security configuration.
//
// GetBlockedPathsは全コンテナの全てのブロックパス設定を返します。
// デバッグやセキュリティ設定の表示に便利です。
func (p *Policy) GetBlockedPaths() []BlockedPath {
	if p.blockedPathsManager == nil {
		return nil
	}
	return p.blockedPathsManager.GetBlockedPaths()
}

// GetBlockedPathsForContainer returns blocked paths for a specific container.
// Includes both container-specific paths and global patterns (marked with "*").
//
// GetBlockedPathsForContainerは特定のコンテナのブロックパスを返します。
// コンテナ固有のパスとグローバルパターン（"*"でマーク）の両方を含みます。
func (p *Policy) GetBlockedPathsForContainer(containerName string) []BlockedPath {
	if p.blockedPathsManager == nil {
		return nil
	}
	return p.blockedPathsManager.GetBlockedPathsForContainer(containerName)
}

// CanAccessContainer checks if a container can be accessed based on the allowed list.
// Uses glob pattern matching (e.g., "app-*" matches "app-web", "app-api").
// If no allowed list is configured, all containers are accessible.
//
// CanAccessContainerは許可リストに基づいてコンテナにアクセスできるかチェックします。
// globパターンマッチングを使用します（例: "app-*"は"app-web"、"app-api"にマッチ）。
// 許可リストが設定されていない場合、全てのコンテナにアクセス可能です。
func (p *Policy) CanAccessContainer(containerName string) bool {
	// If no whitelist is specified, allow all containers
	// ホワイトリストが指定されていない場合、全コンテナを許可
	if len(p.config.AllowedContainers) == 0 {
		return true
	}

	// Check against whitelist using glob pattern matching
	// globパターンマッチングを使用してホワイトリストをチェック
	for _, pattern := range p.config.AllowedContainers {
		matched, err := filepath.Match(pattern, containerName)
		if err != nil {
			continue // Skip invalid patterns / 無効なパターンはスキップ
		}
		if matched {
			return true
		}
	}

	return false
}

// CanGetLogs checks if retrieving container logs is allowed.
// This is controlled by the permissions.logs setting.
//
// CanGetLogsはコンテナログの取得が許可されているかチェックします。
// これはpermissions.logs設定で制御されます。
func (p *Policy) CanGetLogs() bool {
	return p.config.Permissions.Logs
}

// CanInspect checks if container inspection is allowed.
// This is controlled by the permissions.inspect setting.
//
// CanInspectはコンテナ検査が許可されているかチェックします。
// これはpermissions.inspect設定で制御されます。
func (p *Policy) CanInspect() bool {
	return p.config.Permissions.Inspect
}

// CanGetStats checks if retrieving container stats is allowed.
// This is controlled by the permissions.stats setting.
//
// CanGetStatsはコンテナ統計の取得が許可されているかチェックします。
// これはpermissions.stats設定で制御されます。
func (p *Policy) CanGetStats() bool {
	return p.config.Permissions.Stats
}

// CanLifecycle checks if container lifecycle operations (start/stop/restart) are allowed
// for the specified container. Uses Docker API directly (no shell execution).
//
// This involves multiple checks:
//   1. Is lifecycle globally enabled? (permissions.lifecycle)
//   2. Is the container accessible? (allowed_containers)
//   3. Does the security mode allow lifecycle? (denied in strict mode)
//
// CanLifecycleはコンテナのライフサイクル操作（start/stop/restart）が
// 指定されたコンテナに対して許可されているかチェックします。
// Docker APIを直接使用します（シェル実行なし）。
func (p *Policy) CanLifecycle(containerName string) (bool, error) {
	if !p.config.Permissions.Lifecycle {
		return false, fmt.Errorf("lifecycle operations are disabled in security policy")
	}

	if !p.CanAccessContainer(containerName) {
		return false, fmt.Errorf("container not in allowed list: %s", containerName)
	}

	if p.config.Mode == "strict" {
		return false, fmt.Errorf("lifecycle operations are not allowed in strict mode")
	}

	return true, nil
}

// CanExec checks if executing a command in a container is allowed.
// This involves multiple checks:
//   1. Is exec globally enabled? (permissions.exec)
//   2. Is the container accessible? (allowed_containers)
//   3. Does the security mode allow exec?
//   4. Is the command whitelisted? (in moderate mode)
//
// CanExecはコンテナ内でのコマンド実行が許可されているかチェックします。
// これは複数のチェックを含みます：
//   1. execがグローバルに有効か？（permissions.exec）
//   2. コンテナにアクセス可能か？（allowed_containers）
//   3. セキュリティモードがexecを許可しているか？
//   4. コマンドがホワイトリストに登録されているか？（moderateモード）
func (p *Policy) CanExec(containerName string, command string) (bool, error) {
	// Check if exec is globally enabled
	// execがグローバルに有効かチェック
	if !p.config.Permissions.Exec {
		return false, fmt.Errorf("exec is disabled in security policy")
	}

	// Check if container is accessible
	// コンテナにアクセス可能かチェック
	if !p.CanAccessContainer(containerName) {
		return false, fmt.Errorf("container not in allowed list: %s", containerName)
	}

	// In strict mode, exec is never allowed regardless of whitelist
	// strictモードでは、ホワイトリストに関係なくexecは許可されない
	if p.config.Mode == "strict" {
		return false, fmt.Errorf("exec is not allowed in strict mode")
	}

	// In permissive mode, allow any command
	// permissiveモードでは、任意のコマンドを許可
	if p.config.Mode == "permissive" {
		return true, nil
	}

	// In moderate mode, check whitelist
	// moderateモードでは、ホワイトリストをチェック
	if p.config.Mode == "moderate" {
		return p.isCommandWhitelisted(containerName, command)
	}

	return false, fmt.Errorf("unknown security mode: %s", p.config.Mode)
}

// isCommandWhitelisted checks if a command is whitelisted for a specific container.
// First checks container-specific whitelist, then falls back to global whitelist (*).
//
// isCommandWhitelistedはコマンドが特定のコンテナに対してホワイトリスト登録されているかチェックします。
// まずコンテナ固有のホワイトリストをチェックし、次にグローバルホワイトリスト（*）にフォールバックします。
func (p *Policy) isCommandWhitelisted(containerName string, command string) (bool, error) {
	// Check container-specific whitelist first
	// まずコンテナ固有のホワイトリストをチェック
	if whitelist, ok := p.config.ExecWhitelist[containerName]; ok {
		if p.matchesWhitelist(command, whitelist) {
			return true, nil
		}
	}

	// Check default whitelist (applies to all containers)
	// デフォルトホワイトリストをチェック（全コンテナに適用）
	if whitelist, ok := p.config.ExecWhitelist["*"]; ok {
		if p.matchesWhitelist(command, whitelist) {
			return true, nil
		}
	}

	// Check if the command is available in dangerous mode and provide a hint
	// 危険モードで利用可能かチェックしてヒントを提供
	if p.isDangerousCommandAvailable(containerName, command) {
		return false, fmt.Errorf("command not whitelisted: %s (hint: this command is available with dangerously=true)", command)
	}

	return false, fmt.Errorf("command not whitelisted: %s", command)
}

// matchesWhitelist checks if a command matches any pattern in the whitelist.
// Supports exact matches and wildcard patterns (e.g., "echo *" matches "echo hello").
//
// matchesWhitelistはコマンドがホワイトリスト内のいずれかのパターンに一致するかチェックします。
// 完全一致とワイルドカードパターンをサポートします（例: "echo *"は"echo hello"にマッチ）。
func (p *Policy) matchesWhitelist(command string, whitelist []string) bool {
	// Normalize command by trimming whitespace
	// ホワイトスペースを削除してコマンドを正規化
	command = strings.TrimSpace(command)

	for _, pattern := range whitelist {
		pattern = strings.TrimSpace(pattern)

		// Exact match check
		// 完全一致チェック
		if command == pattern {
			return true
		}

		// Pattern with wildcards - check prefix matching
		// For example, "echo *" matches any command starting with "echo "
		//
		// ワイルドカード付きパターン - プレフィックスマッチングをチェック
		// 例えば、"echo *"は"echo "で始まる任意のコマンドにマッチ
		if strings.Contains(pattern, "*") {
			prefix := strings.Split(pattern, "*")[0]
			if strings.HasPrefix(command, prefix) {
				return true
			}
		}
	}

	return false
}

// GetMode returns the current security mode (strict/moderate/permissive).
// GetModeは現在のセキュリティモード（strict/moderate/permissive）を返します。
func (p *Policy) GetMode() string {
	return p.config.Mode
}

// GetAllowedCommands returns all commands allowed for a specific container.
// Includes both container-specific commands and global commands (*).
//
// GetAllowedCommandsは特定のコンテナに許可された全コマンドを返します。
// コンテナ固有のコマンドとグローバルコマンド（*）の両方を含みます。
func (p *Policy) GetAllowedCommands(containerName string) []string {
	var commands []string

	// Add container-specific commands
	// コンテナ固有のコマンドを追加
	if whitelist, ok := p.config.ExecWhitelist[containerName]; ok {
		commands = append(commands, whitelist...)
	}

	// Add default commands (available to all containers)
	// デフォルトコマンドを追加（全コンテナで利用可能）
	if whitelist, ok := p.config.ExecWhitelist["*"]; ok {
		commands = append(commands, whitelist...)
	}

	return commands
}

// GetSecurityPolicy returns the current security policy configuration as a map.
// This is useful for exposing the policy via the MCP get_security_policy tool.
//
// GetSecurityPolicyは現在のセキュリティポリシー設定をマップとして返します。
// これはMCPのget_security_policyツール経由でポリシーを公開するのに便利です。
func (p *Policy) GetSecurityPolicy() map[string]any {
	return map[string]any{
		"mode":               p.config.Mode,
		"allowed_containers": p.config.AllowedContainers,
		"permissions": map[string]bool{
			"logs":      p.config.Permissions.Logs,
			"inspect":   p.config.Permissions.Inspect,
			"stats":     p.config.Permissions.Stats,
			"exec":      p.config.Permissions.Exec,
			"lifecycle": p.config.Permissions.Lifecycle,
		},
		"exec_whitelist": p.config.ExecWhitelist,
	}
}

// GetAllContainersWithCommands returns all containers that have whitelisted commands.
// Returns a map of container name to list of allowed commands.
// Includes the special "*" entry for global commands.
//
// GetAllContainersWithCommandsはホワイトリストコマンドを持つ全コンテナを返します。
// コンテナ名から許可コマンドリストへのマップを返します。
// グローバルコマンドの特別な"*"エントリを含みます。
func (p *Policy) GetAllContainersWithCommands() map[string][]string {
	result := make(map[string][]string)

	// Copy all container-specific commands
	// 全てのコンテナ固有コマンドをコピー
	for container, commands := range p.config.ExecWhitelist {
		if container != "*" {
			result[container] = commands
		}
	}

	// Add global commands under "*" key
	// "*"キーの下にグローバルコマンドを追加
	if defaultCmds, ok := p.config.ExecWhitelist["*"]; ok {
		result["*"] = defaultCmds
	}

	return result
}

// MaskLogs masks sensitive data in log output.
// Returns the original string if masking is disabled or masker is not initialized.
//
// MaskLogsはログ出力内の機密データをマスクします。
// マスキングが無効またはマスカーが初期化されていない場合は元の文字列を返します。
func (p *Policy) MaskLogs(output string) string {
	if p.outputMasker == nil {
		return output
	}
	return p.outputMasker.MaskLogs(output)
}

// MaskExec masks sensitive data in exec command output.
// Returns the original string if masking is disabled or masker is not initialized.
//
// MaskExecはexecコマンド出力内の機密データをマスクします。
// マスキングが無効またはマスカーが初期化されていない場合は元の文字列を返します。
func (p *Policy) MaskExec(output string) string {
	if p.outputMasker == nil {
		return output
	}
	return p.outputMasker.MaskExec(output)
}

// MaskInspect masks sensitive data in container inspection output.
// Returns the original string if masking is disabled or masker is not initialized.
//
// MaskInspectはコンテナ検査出力内の機密データをマスクします。
// マスキングが無効またはマスカーが初期化されていない場合は元の文字列を返します。
func (p *Policy) MaskInspect(output string) string {
	if p.outputMasker == nil {
		return output
	}
	return p.outputMasker.MaskInspect(output)
}

// GetOutputMaskingStatus returns the current output masking configuration status.
// GetOutputMaskingStatusは現在の出力マスキング設定状態を返します。
func (p *Policy) GetOutputMaskingStatus() map[string]any {
	if p.outputMasker == nil {
		return map[string]any{
			"enabled":       false,
			"pattern_count": 0,
		}
	}
	return map[string]any{
		"enabled":       p.outputMasker.IsEnabled(),
		"pattern_count": p.outputMasker.PatternCount(),
		"apply_to": map[string]bool{
			"logs":    p.outputMasker.ShouldMaskLogs(),
			"exec":    p.outputMasker.ShouldMaskExec(),
			"inspect": p.outputMasker.ShouldMaskInspect(),
		},
	}
}

// CanExecDangerously checks if a command can be executed in dangerous mode.
// This allows commands like tail, grep, cat that are not in the whitelist,
// but enforces blocked_paths restrictions on file paths in the command.
//
// Security checks performed:
//  1. Is exec_dangerously globally enabled?
//  2. Is exec globally enabled?
//  3. Is the container accessible?
//  4. Is the base command in exec_dangerously.commands list?
//  5. Are there any pipes or redirects? (forbidden)
//  6. Does the command contain path traversal (..)? (forbidden)
//  7. Are all file paths in the command allowed (not in blocked_paths)?
//
// CanExecDangerouslyは危険モードでコマンドを実行できるかチェックします。
// ホワイトリストにないtail、grep、catなどのコマンドを許可しますが、
// コマンド内のファイルパスにblocked_pathsの制限を適用します。
//
// 実行されるセキュリティチェック:
//  1. exec_dangerouslyがグローバルに有効か？
//  2. execがグローバルに有効か？
//  3. コンテナにアクセス可能か？
//  4. ベースコマンドがexec_dangerously.commandsリストにあるか？
//  5. パイプやリダイレクトがあるか？（禁止）
//  6. コマンドにパストラバーサル(..)が含まれるか？（禁止）
//  7. コマンド内の全ファイルパスが許可されているか（blocked_pathsにないか）？
func (p *Policy) CanExecDangerously(containerName string, command string) (bool, error) {
	// Check if dangerous mode is globally enabled
	// 危険モードがグローバルに有効かチェック
	if !p.config.ExecDangerously.Enabled {
		return false, fmt.Errorf("dangerous mode is not enabled in security policy")
	}

	// Check if exec is globally enabled
	// execがグローバルに有効かチェック
	if !p.config.Permissions.Exec {
		return false, fmt.Errorf("exec is disabled in security policy")
	}

	// Check if container is accessible
	// コンテナにアクセス可能かチェック
	if !p.CanAccessContainer(containerName) {
		return false, fmt.Errorf("container not in allowed list: %s", containerName)
	}

	// In strict mode, dangerous exec is never allowed
	// strictモードでは、危険なexecは許可されない
	if p.config.Mode == "strict" {
		return false, fmt.Errorf("dangerous exec is not allowed in strict mode")
	}

	// Check for shell meta-characters: pipes, redirects, command chaining,
	// command substitution ($(), backticks), and newlines (forbidden in dangerous mode)
	// シェルメタ文字をチェック：パイプ、リダイレクト、コマンドチェーン、
	// コマンド置換（$()、バッククォート）、改行（危険モードでは禁止）
	if strings.ContainsAny(command, "|><;&`\n") || strings.Contains(command, "$(") {
		return false, fmt.Errorf("shell meta-characters (pipes, redirects, command chaining, command substitution, newlines) are not allowed in dangerous mode")
	}

	// Check for path traversal (forbidden)
	// パストラバーサルをチェック（禁止）
	if strings.Contains(command, "..") {
		return false, fmt.Errorf("path traversal (..) is not allowed in dangerous mode")
	}

	// Extract base command name
	// ベースコマンド名を抽出
	baseCommand := extractBaseCommand(command)
	if baseCommand == "" {
		return false, fmt.Errorf("empty command")
	}

	// Check if base command is allowed for this container
	// ベースコマンドがこのコンテナで許可されているかチェック
	if !p.isDangerousCommandAllowed(containerName, baseCommand) {
		return false, fmt.Errorf("command '%s' is not in exec_dangerously list for container '%s'", baseCommand, containerName)
	}

	// Extract file paths from command and check against blocked paths
	// コマンドからファイルパスを抽出し、ブロックパスに対してチェック
	paths := extractPathsFromCommand(command)
	for _, path := range paths {
		if blocked := p.IsPathBlocked(containerName, path); blocked != nil {
			return false, fmt.Errorf("path is blocked: %s (reason: %s)", path, blocked.Reason)
		}
	}

	return true, nil
}

// isDangerousCommandAvailable checks if a command could be executed in dangerous mode.
// This is used to provide helpful hints when a command is not whitelisted.
// It extracts the base command and checks if it's in the exec_dangerously list.
//
// isDangerousCommandAvailableはコマンドが危険モードで実行可能かチェックします。
// コマンドがホワイトリストにない場合に役立つヒントを提供するために使用されます。
// ベースコマンドを抽出し、exec_dangerouslyリストにあるかチェックします。
func (p *Policy) isDangerousCommandAvailable(containerName string, command string) bool {
	// Extract base command (first word)
	// ベースコマンド（最初の単語）を抽出
	baseCommand := strings.Fields(command)
	if len(baseCommand) == 0 {
		return false
	}
	return p.isDangerousCommandAllowed(containerName, baseCommand[0])
}

// isDangerousCommandAllowed checks if a base command is in the exec_dangerously list.
// Checks container-specific list first, then falls back to global list (*).
//
// isDangerousCommandAllowedはベースコマンドがexec_dangerouslyリストにあるかチェックします。
// まずコンテナ固有のリストをチェックし、次にグローバルリスト（*）にフォールバックします。
func (p *Policy) isDangerousCommandAllowed(containerName string, baseCommand string) bool {
	// Check container-specific list first
	// まずコンテナ固有のリストをチェック
	if commands, ok := p.config.ExecDangerously.Commands[containerName]; ok {
		for _, cmd := range commands {
			if cmd == baseCommand {
				return true
			}
		}
	}

	// Check global list (*)
	// グローバルリスト（*）をチェック
	if commands, ok := p.config.ExecDangerously.Commands["*"]; ok {
		for _, cmd := range commands {
			if cmd == baseCommand {
				return true
			}
		}
	}

	return false
}

// parseCommandArgs parses a command string into individual arguments,
// handling quoted strings and backslash escapes properly.
// This provides more robust parsing than simple whitespace splitting.
//
// Supported syntax:
//   - Unquoted arguments are split by whitespace
//   - Single-quoted strings: 'arg with spaces' (content is literal)
//   - Double-quoted strings: "arg with spaces" (content is literal)
//   - Backslash escapes: arg\ with\ spaces (escapes the next character)
//
// parseCommandArgsはコマンド文字列を個々の引数に解析し、
// 引用符で囲まれた文字列とバックスラッシュエスケープを適切に処理します。
// これは単純な空白分割よりも堅牢な解析を提供します。
//
// サポートされる構文：
//   - 引用符なしの引数は空白で分割される
//   - シングルクォート文字列: 'arg with spaces'（内容はリテラル）
//   - ダブルクォート文字列: "arg with spaces"（内容はリテラル）
//   - バックスラッシュエスケープ: arg\ with\ spaces（次の文字をエスケープ）
func parseCommandArgs(command string) []string {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}

	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	// Track if we're building an argument (needed for empty quoted strings like "")
	// 引数を構築中かどうかを追跡（""のような空の引用符文字列に必要）
	hasContent := false

	for i := 0; i < len(command); i++ {
		ch := command[i]

		// Handle escape sequences
		// エスケープシーケンスを処理
		if escaped {
			current.WriteByte(ch)
			escaped = false
			hasContent = true
			continue
		}

		// Handle backslash escape (outside quotes or inside double quotes)
		// バックスラッシュエスケープを処理（引用符外またはダブルクォート内）
		if ch == '\\' && !inSingleQuote {
			// In double quotes, only escape certain characters (", \, $, `)
			// ダブルクォート内では特定の文字のみエスケープ（", \, $, `）
			if inDoubleQuote {
				if i+1 < len(command) {
					next := command[i+1]
					if next == '"' || next == '\\' || next == '$' || next == '`' {
						escaped = true
						continue
					}
				}
				// If not a special escape, treat backslash as literal
				// 特殊エスケープでない場合、バックスラッシュをリテラルとして扱う
				current.WriteByte(ch)
				hasContent = true
				continue
			}
			escaped = true
			continue
		}

		// Handle single quotes (not inside double quotes)
		// シングルクォートを処理（ダブルクォート内ではない場合）
		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			hasContent = true // Even empty quotes count as content
			continue
		}

		// Handle double quotes (not inside single quotes)
		// ダブルクォートを処理（シングルクォート内ではない場合）
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			hasContent = true // Even empty quotes count as content
			continue
		}

		// Handle whitespace (argument separator when not in quotes)
		// 空白を処理（引用符内でない場合は引数の区切り）
		if (ch == ' ' || ch == '\t') && !inSingleQuote && !inDoubleQuote {
			if hasContent {
				args = append(args, current.String())
				current.Reset()
				hasContent = false
			}
			continue
		}

		// Regular character
		// 通常の文字
		current.WriteByte(ch)
		hasContent = true
	}

	// Add the last argument if any
	// 最後の引数があれば追加
	if hasContent {
		args = append(args, current.String())
	}

	return args
}

// extractBaseCommand extracts the base command name from a command string.
// For example, "tail -f /var/log/app.log" returns "tail".
//
// extractBaseCommandはコマンド文字列からベースコマンド名を抽出します。
// 例えば、"tail -f /var/log/app.log"は"tail"を返します。
func extractBaseCommand(command string) string {
	// Use the robust parser to handle quoted commands
	// 引用符付きコマンドを処理するために堅牢なパーサーを使用
	parts := parseCommandArgs(command)
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

// extractPathsFromCommand extracts file paths from a command string.
// Only arguments that look like file paths are extracted:
//   - Arguments starting with "/" (absolute paths)
//   - Arguments that don't start with "-" (options) and contain "/" (relative paths)
//   - Arguments starting with "." (hidden files like .env, .gitignore)
//
// This function uses parseCommandArgs to properly handle quoted strings and escapes,
// preventing bypass attempts like `cat '/app/.env'` or `cat /app/.e\ nv`.
//
// extractPathsFromCommandはコマンド文字列からファイルパスを抽出します。
// ファイルパスのように見える引数のみが抽出されます：
//   - "/" で始まる引数（絶対パス）
//   - "-" で始まらず（オプション）、"/" を含む引数（相対パス）
//   - "." で始まる引数（.envや.gitignoreなどの隠しファイル）
//
// この関数はparseCommandArgsを使用して引用符で囲まれた文字列とエスケープを
// 適切に処理し、`cat '/app/.env'`や`cat /app/.e\ nv`のようなバイパス試行を防ぎます。
func extractPathsFromCommand(command string) []string {
	// Use the robust parser to handle quoted paths and escapes
	// 引用符付きパスとエスケープを処理するために堅牢なパーサーを使用
	parts := parseCommandArgs(command)
	if len(parts) <= 1 {
		return nil
	}

	var paths []string
	// Skip first part (command name), iterate over arguments
	// 最初の部分（コマンド名）をスキップし、引数を反復
	for i := 1; i < len(parts); i++ {
		arg := parts[i]

		// Skip options (arguments starting with "-")
		// オプション（"-"で始まる引数）をスキップ
		if strings.HasPrefix(arg, "-") {
			continue
		}

		// Check if it looks like a file path
		// ファイルパスのように見えるかチェック
		// - Absolute path: starts with "/"
		// - Relative path with directory: contains "/"
		// - Hidden file: starts with "." (e.g., .env, .gitignore)
		// - 絶対パス: "/" で始まる
		// - ディレクトリを含む相対パス: "/" を含む
		// - 隠しファイル: "." で始まる（例: .env, .gitignore）
		if strings.HasPrefix(arg, "/") || strings.Contains(arg, "/") || strings.HasPrefix(arg, ".") {
			paths = append(paths, arg)
		}
	}

	return paths
}

// GetDangerousCommandsForContainer returns the dangerous commands allowed for a container.
// Includes both container-specific commands and global commands (*).
//
// GetDangerousCommandsForContainerはコンテナで許可される危険コマンドを返します。
// コンテナ固有のコマンドとグローバルコマンド（*）の両方を含みます。
func (p *Policy) GetDangerousCommandsForContainer(containerName string) []string {
	var commands []string

	// Add container-specific commands
	// コンテナ固有のコマンドを追加
	if cmds, ok := p.config.ExecDangerously.Commands[containerName]; ok {
		commands = append(commands, cmds...)
	}

	// Add global commands
	// グローバルコマンドを追加
	if cmds, ok := p.config.ExecDangerously.Commands["*"]; ok {
		commands = append(commands, cmds...)
	}

	return commands
}

// GetAllDangerousCommands returns a map of all containers and their dangerous commands.
// Returns a map of container name to list of dangerous commands.
// Includes the special "*" entry for global commands.
//
// GetAllDangerousCommandsはすべてのコンテナとそれぞれの危険コマンドのマップを返します。
// コンテナ名から危険コマンドリストへのマップを返します。
// グローバルコマンドの特別な"*"エントリを含みます。
func (p *Policy) GetAllDangerousCommands() map[string][]string {
	result := make(map[string][]string)

	// Copy all container-specific commands
	// 全てのコンテナ固有コマンドをコピー
	for container, commands := range p.config.ExecDangerously.Commands {
		if container != "*" {
			result[container] = commands
		}
	}

	// Add global commands under "*" key
	// "*"キーの下にグローバルコマンドを追加
	if defaultCmds, ok := p.config.ExecDangerously.Commands["*"]; ok {
		result["*"] = defaultCmds
	}

	return result
}

// IsDangerousModeEnabled returns whether dangerous mode is globally enabled.
// IsDangerousModeEnabledは危険モードがグローバルに有効かを返します。
func (p *Policy) IsDangerousModeEnabled() bool {
	return p.config.ExecDangerously.Enabled
}

// SetDangerousModeEnabled enables or disables dangerous mode at runtime.
// This allows CLI flags to override the config file setting.
//
// SetDangerousModeEnabledは実行時に危険モードを有効/無効にします。
// これによりCLIフラグが設定ファイルの設定を上書きできます。
func (p *Policy) SetDangerousModeEnabled(enabled bool) {
	p.config.ExecDangerously.Enabled = enabled
}

// SetDangerousCommands sets the dangerous commands for specific containers at runtime.
// Pass "*" as container name to set global commands.
// This allows CLI flags to override the config file setting.
//
// SetDangerousCommandsは実行時に特定のコンテナの危険コマンドを設定します。
// コンテナ名に"*"を渡すとグローバルコマンドを設定できます。
// これによりCLIフラグが設定ファイルの設定を上書きできます。
func (p *Policy) SetDangerousCommands(containerName string, commands []string) {
	if p.config.ExecDangerously.Commands == nil {
		p.config.ExecDangerously.Commands = make(map[string][]string)
	}
	p.config.ExecDangerously.Commands[containerName] = commands
}

// MaskHostPaths masks host OS paths in the given string.
// This is used to hide the host OS username and directory structure from AI assistants.
// Only applies when host_path_masking is enabled in the security configuration.
//
// Masking rules:
//   - /Users/<username>/... → [HOST_PATH]/...
//   - /home/<username>/... → [HOST_PATH]/...
//   - C:\Users\<username>\... → [HOST_PATH]\...
//
// MaskHostPathsは指定された文字列内のホストOSパスをマスクします。
// これはAIアシスタントからホストOSのユーザー名やディレクトリ構造を隠すために使用されます。
// セキュリティ設定でhost_path_maskingが有効な場合のみ適用されます。
//
// マスキングルール:
//   - /Users/<username>/... → [HOST_PATH]/...
//   - /home/<username>/... → [HOST_PATH]/...
//   - C:\Users\<username>\... → [HOST_PATH]\...
func (p *Policy) MaskHostPaths(input string) string {
	if !p.config.HostPathMasking.Enabled {
		return input
	}

	replacement := p.config.HostPathMasking.Replacement
	if replacement == "" {
		replacement = "[HOST_PATH]"
	}

	return maskHostPathsInString(input, replacement)
}

// IsHostPathMaskingEnabled returns whether host path masking is enabled.
// IsHostPathMaskingEnabledはホストパスマスキングが有効かを返します。
func (p *Policy) IsHostPathMaskingEnabled() bool {
	return p.config.HostPathMasking.Enabled
}

// maskHostPathsInString performs the actual masking of host paths in a string.
// It uses regular expressions to find and replace home directory paths.
//
// maskHostPathsInStringは文字列内のホストパスの実際のマスキングを行います。
// 正規表現を使用してホームディレクトリパスを検索し置換します。
func maskHostPathsInString(input string, replacement string) string {
	// Define patterns for different OS home directories
	// 異なるOSのホームディレクトリのパターンを定義
	patterns := []string{
		// macOS: /Users/username/...
		"/Users/",
		// Linux: /home/username/...
		"/home/",
		// Windows (forward slash variant): /c/Users/username/...
		"/c/Users/",
		"/C/Users/",
	}

	result := input

	// First handle Windows-style paths (C:\Users\... or C:/Users/...)
	// This must be done BEFORE Unix patterns, otherwise /Users/ would match
	// the middle of C:/Users/...
	// 最初にWindows形式のパスを処理（C:\Users\... または C:/Users/...）
	// Unix パターンより先に処理しないと、/Users/ が C:/Users/... の途中にマッチしてしまう
	result = maskWindowsPaths(result, replacement)

	// Then handle Unix-style paths
	// その後、Unix形式のパスを処理
	for _, prefix := range patterns {
		result = maskPathsWithPrefix(result, prefix, replacement)
	}

	return result
}

// pathTerminators defines characters that terminate a path segment.
// These characters mark the end of a username or path in structured text
// such as JSON, YAML, or command-line output.
//
// pathTerminatorsはパスセグメントを終了する文字を定義します。
// JSONやYAML、コマンドライン出力などの構造化テキスト内で
// ユーザー名やパスの終端を示す文字です。
const pathTerminators = "\"' ,]}\n\t"

// maskPathWithSeparators finds occurrences of prefix in input, extracts the username
// after the prefix (terminated by any character in separators or terminators),
// and replaces prefix+username with replacement.
//
// separators: characters that mark the end of the username and the start of the
// remaining path (e.g., "/" for Unix, "\\/" for Windows).
//
// maskPathWithSeparatorsはinput内のprefixの出現を検索し、prefix後のユーザー名を
// 抽出（separatorsまたはterminatorsのいずれかの文字で終端）し、
// prefix+usernameをreplacementに置換します。
//
// separators: ユーザー名の終端かつ残りのパスの開始を示す文字
// （例: Unixでは"/"、Windowsでは"\\/"）。
func maskPathWithSeparators(input, prefix, replacement, separators, terminators string) string {
	result := input
	startIdx := 0

	for {
		idx := strings.Index(result[startIdx:], prefix)
		if idx == -1 {
			break
		}
		idx += startIdx

		usernameStart := idx + len(prefix)
		usernameEnd := usernameStart

		for i := usernameStart; i < len(result); i++ {
			ch := result[i]
			if strings.ContainsRune(separators, rune(ch)) {
				usernameEnd = i
				break
			}
			if strings.ContainsRune(terminators, rune(ch)) {
				usernameEnd = i
				break
			}
			usernameEnd = i + 1
		}

		if usernameEnd > usernameStart {
			result = result[:idx] + replacement + result[usernameEnd:]
			startIdx = idx + len(replacement)
		} else {
			startIdx = idx + 1
		}
	}

	return result
}

// maskPathsWithPrefix replaces paths that start with a given prefix.
// For example, /Users/john/workspace → [HOST_PATH]/workspace
//
// maskPathsWithPrefixは指定されたプレフィックスで始まるパスを置換します。
// 例: /Users/john/workspace → [HOST_PATH]/workspace
func maskPathsWithPrefix(input string, prefix string, replacement string) string {
	return maskPathWithSeparators(input, prefix, replacement, "/", pathTerminators)
}

// maskWindowsPaths handles Windows-style paths with backslashes or forward slashes.
// C:\Users\john\... → [HOST_PATH]\...
// C:/Users/john/... → [HOST_PATH]/...
//
// maskWindowsPathsはバックスラッシュまたはフォワードスラッシュを使用したWindows形式のパスを処理します。
// C:\Users\john\... → [HOST_PATH]\...
// C:/Users/john/... → [HOST_PATH]/...
func maskWindowsPaths(input string, replacement string) string {
	// Pattern: C:\Users\ or C:/Users/ or similar
	// パターン: C:\Users\ や C:/Users/ など
	patterns := []string{
		`C:\Users\`,
		`c:\Users\`,
		`D:\Users\`,
		`d:\Users\`,
		`C:/Users/`,
		`c:/Users/`,
		`D:/Users/`,
		`d:/Users/`,
	}

	result := input
	for _, prefix := range patterns {
		result = maskPathWithSeparators(result, prefix, replacement, "\\/", pathTerminators)
	}
	return result
}
