// Package config provides configuration management for DockMCP.
// It handles loading, parsing, and validating configuration from YAML files.
//
// configパッケージはDockMCPの設定管理を提供します。
// YAMLファイルからの設定の読み込み、解析、検証を処理します。
//
// Configuration is loaded from the following locations (in order of precedence):
// 設定は以下の場所から読み込まれます（優先順位順）：
//   1. Explicitly specified config file path (明示的に指定された設定ファイルパス)
//   2. ./dkmcp.yaml or ./dkmcp.yml (カレントディレクトリ)
//   3. ./configs/dkmcp.yaml (configsディレクトリ)
//   4. ~/.dkmcp/dkmcp.yaml (ホームディレクトリ)
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration.
// It contains all settings needed to run the DockMCP server.
//
// Configはアプリケーション全体の設定を表します。
// DockMCPサーバーを実行するために必要なすべての設定を含みます。
type Config struct {
	// Server contains HTTP server settings (port, host)
	// Serverはサーバー設定を含みます（ポート、ホスト）
	Server ServerConfig `yaml:"server"`

	// Security contains access control and permission settings
	// Securityはアクセス制御と権限の設定を含みます
	Security SecurityConfig `yaml:"security"`

	// Logging contains log output settings
	// Loggingはログ出力の設定を含みます
	Logging LoggingConfig `yaml:"logging"`

	// Audit contains audit logging settings for security monitoring
	// Auditはセキュリティ監視のための監査ログ設定を含みます
	Audit AuditConfig `yaml:"audit"`

	// CLI contains CLI-specific settings for human convenience features
	// CLIはユーザーの利便性のためのCLI固有の設定を含みます
	CLI CLIConfig `yaml:"cli"`
}

// ServerConfig holds server-related configuration.
// These settings control how the MCP server listens for connections.
//
// ServerConfigはサーバー関連の設定を保持します。
// これらの設定はMCPサーバーが接続を待ち受ける方法を制御します。
type ServerConfig struct {
	// Port is the TCP port to listen on (default: 8080)
	// Portは待ち受けるTCPポートです（デフォルト: 8080）
	Port int `yaml:"port"`

	// Host is the network interface to bind to (default: "0.0.0.0" = all interfaces)
	// Hostはバインドするネットワークインターフェースです（デフォルト: "0.0.0.0" = 全インターフェース）
	Host string `yaml:"host"`
}

// SecurityConfig holds security-related configuration.
// This is the core of DockMCP's access control system.
//
// SecurityConfigはセキュリティ関連の設定を保持します。
// これはDockMCPのアクセス制御システムの中核です。
type SecurityConfig struct {
	// Mode determines the overall security strictness.
	// Valid values: "strict", "moderate", "permissive"
	//   - strict: Only explicitly allowed containers and commands
	//   - moderate: Balanced security with sensible defaults
	//   - permissive: Less restrictive, more access allowed
	//
	// Modeは全体的なセキュリティの厳格さを決定します。
	// 有効な値: "strict", "moderate", "permissive"
	//   - strict: 明示的に許可されたコンテナとコマンドのみ
	//   - moderate: 適切なデフォルトでバランスの取れたセキュリティ
	//   - permissive: 制限が少なく、より多くのアクセスを許可
	Mode string `yaml:"mode"`

	// AllowedContainers is a list of container name patterns that can be accessed.
	// Supports glob patterns (e.g., "myapp-*", "prod-api-?").
	// Empty list means all containers are accessible (in moderate/permissive mode).
	//
	// AllowedContainersはアクセス可能なコンテナ名パターンのリストです。
	// globパターンをサポートします（例: "myapp-*", "prod-api-?"）。
	// 空のリストはすべてのコンテナにアクセス可能を意味します（moderate/permissiveモード）。
	AllowedContainers []string `yaml:"allowed_containers"`

	// ExecWhitelist defines which commands can be executed in each container.
	// Key: container name, Value: list of allowed commands.
	// Example: {"api": ["npm test", "npm run lint"]}
	//
	// ExecWhitelistは各コンテナで実行可能なコマンドを定義します。
	// キー: コンテナ名、値: 許可されたコマンドのリスト。
	// 例: {"api": ["npm test", "npm run lint"]}
	ExecWhitelist map[string][]string `yaml:"exec_whitelist"`

	// Permissions defines which operations are globally allowed.
	// Permissionsはグローバルに許可される操作を定義します。
	Permissions SecurityPermissions `yaml:"permissions"`

	// BlockedPaths configures which file paths are blocked from access.
	// BlockedPathsはアクセスをブロックするファイルパスを設定します。
	BlockedPaths BlockedPathsConfig `yaml:"blocked_paths"`

	// OutputMasking configures masking of sensitive data in command output.
	// This applies to logs, exec results, and container inspection.
	// OutputMaskingはコマンド出力内の機密データのマスキングを設定します。
	// これはログ、exec結果、コンテナ検査に適用されます。
	OutputMasking OutputMaskingConfig `yaml:"output_masking"`

	// ExecDangerously configures dangerous mode for exec_command.
	// When enabled, allows execution of commands like tail, grep, cat with
	// file path validation against blocked_paths.
	// ExecDangerouslyはexec_commandの危険モードを設定します。
	// 有効にすると、tail、grep、catなどのコマンドを
	// blocked_pathsに対するファイルパス検証付きで実行できます。
	ExecDangerously ExecDangerouslyConfig `yaml:"exec_dangerously"`

	// HostPathMasking configures masking of host OS paths in MCP tool output.
	// This hides the host OS username and directory structure from AI assistants.
	// HostPathMaskingはMCPツール出力でのホストOSパスのマスキングを設定します。
	// これによりAIアシスタントからホストOSのユーザー名やディレクトリ構造を隠します。
	HostPathMasking HostPathMaskingConfig `yaml:"host_path_masking"`
}

// BlockedPathsConfig holds configuration for blocked file paths.
// This prevents AI from reading sensitive files like secrets and credentials.
//
// BlockedPathsConfigはブロックされるファイルパスの設定を保持します。
// これによりAIがシークレットや認証情報などの機密ファイルを読むことを防ぎます。
type BlockedPathsConfig struct {
	// Manual is a map of container name to list of blocked paths.
	// Paths can be exact matches or glob patterns.
	// Example: {"api": ["/app/secrets/*", "/app/.env"]}
	//
	// Manualはコンテナ名からブロックパスリストへのマップです。
	// パスは完全一致またはglobパターンが使用できます。
	// 例: {"api": ["/app/secrets/*", "/app/.env"]}
	Manual map[string][]string `yaml:"manual"`

	// AutoImport configures automatic detection of blocked paths from DevContainer configs.
	// AutoImportはDevContainer設定からのブロックパス自動検出を設定します。
	AutoImport AutoImportConfig `yaml:"auto_import"`
}

// OutputMaskingConfig configures masking of sensitive data in output.
// Sensitive information like passwords, API keys, and tokens are replaced
// with a masked string before being returned to the AI assistant.
//
// OutputMaskingConfigは出力内の機密データのマスキングを設定します。
// パスワード、APIキー、トークンなどの機密情報は、
// AIアシスタントに返される前にマスク文字列に置き換えられます。
type OutputMaskingConfig struct {
	// Enabled activates output masking globally.
	// EnabledはOutputMasking機能をグローバルに有効化します。
	Enabled bool `yaml:"enabled"`

	// Replacement is the string used to replace sensitive data.
	// Default: "[MASKED]"
	// Replacementは機密データを置き換える文字列です。
	// デフォルト: "[MASKED]"
	Replacement string `yaml:"replacement"`

	// Patterns is a list of regex patterns to match sensitive data.
	// Each pattern will be replaced with the Replacement string.
	// Patternsは機密データにマッチする正規表現パターンのリストです。
	// 各パターンはReplacement文字列に置き換えられます。
	Patterns []string `yaml:"patterns"`

	// ApplyTo specifies which outputs to apply masking to.
	// ApplyToはマスキングを適用する出力を指定します。
	ApplyTo OutputMaskingTargets `yaml:"apply_to"`
}

// OutputMaskingTargets specifies which tool outputs should be masked.
// OutputMaskingTargetsはマスキングを適用するツール出力を指定します。
type OutputMaskingTargets struct {
	// Logs applies masking to get_logs and search_logs output.
	// Logsはget_logsとsearch_logsの出力にマスキングを適用します。
	Logs bool `yaml:"logs"`

	// Exec applies masking to exec_command output.
	// Execはexec_commandの出力にマスキングを適用します。
	Exec bool `yaml:"exec"`

	// Inspect applies masking to inspect_container output (env vars).
	// Inspectはinspect_containerの出力（環境変数）にマスキングを適用します。
	Inspect bool `yaml:"inspect"`
}

// ExecDangerouslyConfig configures the dangerous mode for exec_command.
// This allows execution of file inspection commands (tail, grep, cat, etc.)
// that are not in the whitelist, while still enforcing blocked_paths restrictions.
//
// ExecDangerouslyConfigはexec_commandの危険モードを設定します。
// ホワイトリストにないファイル検査コマンド（tail、grep、cat等）の実行を許可しますが、
// blocked_pathsの制限は引き続き適用されます。
type ExecDangerouslyConfig struct {
	// Enabled activates dangerous mode feature.
	// When false, the dangerously parameter in exec_command is ignored.
	// Enabledは危険モード機能を有効化します。
	// falseの場合、exec_commandのdangerouslyパラメータは無視されます。
	Enabled bool `yaml:"enabled"`

	// Commands defines which base commands are allowed in dangerous mode per container.
	// Key: container name (use "*" for default/all containers)
	// Value: list of allowed command names (without arguments)
	// Only the base command name is checked; file paths are validated against blocked_paths.
	//
	// Example:
	//   commands:
	//     "securenote-api":
	//       - "tail"
	//       - "cat"
	//     "*":
	//       - "tail"
	//
	// Commandsは危険モードで許可されるベースコマンドをコンテナごとに定義します。
	// キー: コンテナ名（"*"でデフォルト/全コンテナ）
	// 値: 許可されるコマンド名のリスト（引数なし）
	// ベースコマンド名のみチェックされ、ファイルパスはblocked_pathsに対して検証されます。
	Commands map[string][]string `yaml:"commands"`
}

// HostPathMaskingConfig configures masking of host OS paths in MCP tool output.
// This prevents AI assistants from seeing the host OS username and directory structure.
// Only applies to MCP tool output; CLI commands show full paths for human users.
//
// HostPathMaskingConfigはMCPツール出力でのホストOSパスのマスキングを設定します。
// これによりAIアシスタントがホストOSのユーザー名やディレクトリ構造を見ることを防ぎます。
// MCPツール出力にのみ適用され、CLIコマンドは人間のユーザー向けにフルパスを表示します。
type HostPathMaskingConfig struct {
	// Enabled activates host path masking in MCP tool output.
	// Default: true (recommended for security)
	// EnabledはMCPツール出力でのホストパスマスキングを有効化します。
	// デフォルト: true（セキュリティのため推奨）
	Enabled bool `yaml:"enabled"`

	// Replacement is the string used to replace the home directory portion.
	// Default: "[HOST_PATH]"
	// Example: "/Users/john/workspace/project" → "[HOST_PATH]/workspace/project"
	//
	// Replacementはホームディレクトリ部分を置き換える文字列です。
	// デフォルト: "[HOST_PATH]"
	// 例: "/Users/john/workspace/project" → "[HOST_PATH]/workspace/project"
	Replacement string `yaml:"replacement"`
}

// AutoImportConfig holds settings for auto-importing blocked paths from DevContainer configs.
// This feature automatically detects which files are hidden from AI in DevContainer
// and applies the same restrictions in DockMCP.
//
// AutoImportConfigはDevContainer設定からブロックパスを自動インポートする設定を保持します。
// この機能はDevContainerでAIから隠されているファイルを自動検出し、
// 同じ制限をDockMCPに適用します。
type AutoImportConfig struct {
	// Enabled activates auto-import feature.
	// EnabledはAutoImport機能を有効化します。
	Enabled bool `yaml:"enabled"`

	// WorkspaceRoot is the root directory to scan for configuration files.
	// Default: current directory (".")
	//
	// WorkspaceRootは設定ファイルをスキャンするルートディレクトリです。
	// デフォルト: カレントディレクトリ (".")
	WorkspaceRoot string `yaml:"workspace_root"`

	// ScanFiles is a list of files to scan for blocked path configurations.
	// These are typically Docker Compose files that define volume mounts.
	//
	// ScanFilesはブロックパス設定をスキャンするファイルのリストです。
	// 通常、ボリュームマウントを定義するDocker Composeファイルです。
	ScanFiles []string `yaml:"scan_files"`

	// GlobalPatterns are file patterns that are blocked in all containers.
	// These are applied globally regardless of container-specific settings.
	// Examples: ".env", "*.key", "*.pem", "secrets/*"
	//
	// GlobalPatternsはすべてのコンテナでブロックされるファイルパターンです。
	// コンテナ固有の設定に関係なくグローバルに適用されます。
	// 例: ".env", "*.key", "*.pem", "secrets/*"
	GlobalPatterns []string `yaml:"global_patterns"`

	// ClaudeCodeSettings configures import from Claude Code configuration files.
	// ClaudeCodeSettingsはClaude Code設定ファイルからのインポートを設定します。
	ClaudeCodeSettings ClaudeCodeSettingsConfig `yaml:"claude_code_settings"`

	// GeminiSettings configures import from Gemini Code Assist configuration files.
	// GeminiSettingsはGemini Code Assist設定ファイルからのインポートを設定します。
	GeminiSettings GeminiSettingsConfig `yaml:"gemini_settings"`
}

// ClaudeCodeSettingsConfig holds settings for importing blocked paths from Claude Code settings.
// Claude Code can have its own list of files to ignore, which can be imported here.
//
// ClaudeCodeSettingsConfigはClaude Code設定からブロックパスをインポートする設定を保持します。
// Claude Codeは独自の無視ファイルリストを持つことができ、ここでインポートできます。
type ClaudeCodeSettingsConfig struct {
	// Enabled activates import from Claude Code settings.
	// EnabledはClaude Code設定からのインポートを有効化します。
	Enabled bool `yaml:"enabled"`

	// MaxDepth controls how deep to scan for settings files.
	//   0 = workspace_root only
	//   1 = one level deep
	//   2 = two levels deep
	//
	// MaxDepthは設定ファイルをスキャンする深さを制御します。
	//   0 = workspace_root のみ
	//   1 = 1階層下まで
	//   2 = 2階層下まで
	MaxDepth int `yaml:"max_depth"`

	// SettingsFiles lists the Claude Code settings files to scan.
	// Paths are relative to workspace root or subdirectories.
	// Default: [".claude/settings.json", ".claude/settings.local.json"]
	//
	// SettingsFilesはスキャンするClaude Code設定ファイルをリストします。
	// パスはワークスペースルートまたはサブディレクトリからの相対パスです。
	// デフォルト: [".claude/settings.json", ".claude/settings.local.json"]
	SettingsFiles []string `yaml:"settings_files"`
}

// GeminiSettingsConfig holds settings for importing blocked paths from Gemini Code Assist.
// Gemini uses .aiexclude and .geminiignore files with gitignore-style syntax.
//
// GeminiSettingsConfigはGemini Code Assistからブロックパスをインポートする設定を保持します。
// Geminiはgitignore形式の.aiexcludeと.geminiignoreファイルを使用します。
type GeminiSettingsConfig struct {
	// Enabled activates import from Gemini settings files.
	// EnabledはGemini設定ファイルからのインポートを有効化します。
	Enabled bool `yaml:"enabled"`

	// MaxDepth controls how deep to scan for settings files.
	//   0 = workspace_root only
	//   1 = one level deep
	//   2 = two levels deep
	//
	// MaxDepthは設定ファイルをスキャンする深さを制御します。
	//   0 = workspace_root のみ
	//   1 = 1階層下まで
	//   2 = 2階層下まで
	MaxDepth int `yaml:"max_depth"`

	// SettingsFiles lists the Gemini exclusion files to scan.
	// These use gitignore-style syntax.
	// Default: [".aiexclude", ".geminiignore"]
	//
	// SettingsFilesはスキャンするGemini除外ファイルをリストします。
	// これらはgitignore形式の構文を使用します。
	// デフォルト: [".aiexclude", ".geminiignore"]
	SettingsFiles []string `yaml:"settings_files"`
}

// SecurityPermissions defines what operations are allowed globally.
// These are high-level toggles for entire categories of operations.
//
// SecurityPermissionsはグローバルに許可される操作を定義します。
// これらは操作カテゴリ全体に対する高レベルのトグルです。
type SecurityPermissions struct {
	// Logs allows reading container logs via get_logs tool.
	// Logsはget_logsツールによるコンテナログの読み取りを許可します。
	Logs bool `yaml:"logs"`

	// Inspect allows getting container details via inspect_container tool.
	// Inspectはinspect_containerツールによるコンテナ詳細の取得を許可します。
	Inspect bool `yaml:"inspect"`

	// Stats allows getting container resource statistics via get_stats tool.
	// Statsはget_statsツールによるコンテナリソース統計の取得を許可します。
	Stats bool `yaml:"stats"`

	// Exec allows executing commands in containers via exec_command tool.
	// Even when enabled, only whitelisted commands are allowed.
	//
	// Execはexec_commandツールによるコンテナ内でのコマンド実行を許可します。
	// 有効な場合でも、ホワイトリストに登録されたコマンドのみが許可されます。
	Exec bool `yaml:"exec"`
}

// LoggingConfig holds logging configuration.
// Controls how DockMCP outputs logs.
//
// Note: Log output destination is configured via command-line flags:
//   --log-file /path/to/file.log
//   --log-also-stdout
//
// LoggingConfigはロギング設定を保持します。
// DockMCPがログを出力する方法を制御します。
//
// 注意: ログ出力先はコマンドラインフラグで設定します:
//   --log-file /path/to/file.log
//   --log-also-stdout
type LoggingConfig struct {
	// Level sets the minimum log level to output.
	// Valid values: "debug", "info", "warn", "error"
	//
	// Levelは出力する最小ログレベルを設定します。
	// 有効な値: "debug", "info", "warn", "error"
	Level string `yaml:"level"`
}

// AuditConfig holds audit logging configuration.
// Audit logs provide security monitoring by recording all tool executions,
// access denials, and security-relevant events.
//
// AuditConfigは監査ログ設定を保持します。
// 監査ログは、すべてのツール実行、アクセス拒否、セキュリティ関連イベントを
// 記録することでセキュリティ監視を提供します。
type AuditConfig struct {
	// Enabled activates audit logging.
	// Enabledは監査ログを有効化します。
	Enabled bool `yaml:"enabled"`

	// File is the path to write audit logs.
	// If empty, audit logs are written to stdout with regular logs.
	//
	// Fileは監査ログを書き込むパスです。
	// 空の場合、監査ログは通常のログと共にstdoutに出力されます。
	File string `yaml:"file"`

	// Events specifies which events to log.
	// Eventsはログ記録するイベントを指定します。
	Events AuditEvents `yaml:"events"`
}

// AuditEvents specifies which event types to include in audit logs.
// AuditEventsは監査ログに含めるイベントタイプを指定します。
type AuditEvents struct {
	// ToolCalls logs all MCP tool invocations (exec_command, get_logs, etc.)
	// ToolCallsはすべてのMCPツール呼び出しをログ記録します（exec_command、get_logs等）
	ToolCalls bool `yaml:"tool_calls"`

	// AccessDenied logs all permission/access denials (blocked paths, disallowed commands)
	// AccessDeniedはすべての権限/アクセス拒否をログ記録します（ブロックパス、不許可コマンド）
	AccessDenied bool `yaml:"access_denied"`

	// ClientConnections logs client connect/disconnect events
	// ClientConnectionsはクライアント接続/切断イベントをログ記録します
	ClientConnections bool `yaml:"client_connections"`

	// SecurityPolicy logs when security policy is queried
	// SecurityPolicyはセキュリティポリシーが照会された時をログ記録します
	SecurityPolicy bool `yaml:"security_policy"`
}

// CLIConfig holds CLI-specific configuration for human convenience features.
// These features are designed for human users on the host OS, not for AI assistants.
// AI assistants should use MCP tools with explicit parameters instead.
//
// CLIConfigはユーザーの利便性のためのCLI固有の設定を保持します。
// これらの機能はホストOS上のユーザー向けであり、AIアシスタント向けではありません。
// AIアシスタントは代わりに明示的なパラメータを持つMCPツールを使用すべきです。
type CLIConfig struct {
	// CurrentContainer configures the "current container" feature for CLI commands.
	// CurrentContainerはCLIコマンドの「カレントコンテナ」機能を設定します。
	CurrentContainer CurrentContainerConfig `yaml:"current_container"`
}

// CurrentContainerConfig configures the current container feature.
// This allows users to set a default container for CLI commands like logs, stats, exec.
//
// CurrentContainerConfigはカレントコンテナ機能を設定します。
// これによりユーザーはlogs、stats、execなどのCLIコマンドのデフォルトコンテナを設定できます。
//
// Design philosophy:
// - AI (MCP/client): Uses explicit parameters, no convenience features needed
// - Human (direct commands): Uses convenience features for better UX
//
// 設計思想:
// - AI (MCP/client): 明示的なパラメータを使用、利便性機能は不要
// - 人 (直接コマンド): より良いUXのために利便性機能を使用
type CurrentContainerConfig struct {
	// Enabled controls whether the current container feature is active.
	// Default: true (recommended for sandbox environments)
	//
	// When enabled:
	// - `dkmcp use <container>` sets the current container
	// - `dkmcp logs` uses the current container if no argument is provided
	// - `dkmcp exec <command>` uses the current container
	//
	// When disabled:
	// - `dkmcp use` commands return an error
	// - `dkmcp logs` requires explicit container argument
	// - `dkmcp exec` requires explicit container argument
	//
	// Set to false in environments where AI might directly use CLI commands
	// (e.g., non-sandboxed environments) to prevent unintended behavior.
	//
	// Enabledはカレントコンテナ機能が有効かどうかを制御します。
	// デフォルト: true（サンドボックス環境で推奨）
	//
	// 有効な場合:
	// - `dkmcp use <container>` でカレントコンテナを設定
	// - `dkmcp logs` は引数がない場合カレントコンテナを使用
	// - `dkmcp exec <command>` はカレントコンテナを使用
	//
	// 無効な場合:
	// - `dkmcp use` コマンドはエラーを返す
	// - `dkmcp logs` は明示的なコンテナ引数が必要
	// - `dkmcp exec` は明示的なコンテナ引数が必要
	//
	// AIがCLIコマンドを直接使用する可能性がある環境（例：サンドボックスなし）では
	// 予期しない動作を防ぐためにfalseに設定してください。
	Enabled bool `yaml:"enabled"`
}

// NewDefaultConfig returns a Config with sensible default values.
// These defaults provide a balance between security and usability.
//
// NewDefaultConfigは適切なデフォルト値を持つConfigを返します。
// これらのデフォルトはセキュリティと使いやすさのバランスを提供します。
func NewDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Security: SecurityConfig{
			Mode: "moderate",
			Permissions: SecurityPermissions{
				Logs:    true,
				Inspect: true,
				Stats:   true,
				Exec:    true,
			},
			BlockedPaths: BlockedPathsConfig{
				Manual: make(map[string][]string),
				AutoImport: AutoImportConfig{
					Enabled:       false,
					WorkspaceRoot: ".",
					ScanFiles: []string{
						".devcontainer/docker-compose.yml",
						".devcontainer/devcontainer.json",
						"cli_sandbox/docker-compose.yml",
					},
					GlobalPatterns: []string{
						".env",
						"*.key",
						"*.pem",
						"secrets/*",
					},
					ClaudeCodeSettings: ClaudeCodeSettingsConfig{
						Enabled: true,
						SettingsFiles: []string{
							".claude/settings.json",
							".claude/settings.local.json",
						},
					},
					GeminiSettings: GeminiSettingsConfig{
						Enabled: true,
						SettingsFiles: []string{
							".aiexclude",
							".geminiignore",
						},
					},
				},
			},
			OutputMasking: OutputMaskingConfig{
				Enabled:     true,
				Replacement: "[MASKED]",
				Patterns: []string{
					// Password patterns / パスワードパターン
					`(?i)(password|passwd|pwd)\s*[=:]\s*["']?[^\s"'\n]+["']?`,
					// API key patterns / APIキーパターン
					`(?i)(api[_-]?key|apikey|secret[_-]?key)\s*[=:]\s*["']?[^\s"'\n]+["']?`,
					// Generic secret patterns / 一般的なシークレットパターン
					`(?i)(secret|token|credential)\s*[=:]\s*["']?[^\s"'\n]+["']?`,
					// Bearer tokens / Bearerトークン
					`(?i)bearer\s+[a-zA-Z0-9._-]+`,
					// OpenAI API keys / OpenAI APIキー
					`sk-[a-zA-Z0-9]{20,}`,
					// AWS keys / AWSキー
					`(?i)(aws[_-]?access[_-]?key[_-]?id|aws[_-]?secret[_-]?access[_-]?key)\s*[=:]\s*["']?[A-Z0-9/+=]+["']?`,
					// Database connection strings with passwords / パスワード付きDB接続文字列
					`(?i)(postgres|mysql|mongodb|redis)://[^:]+:[^@]+@`,
				},
				ApplyTo: OutputMaskingTargets{
					Logs:    true,
					Exec:    true,
					Inspect: true,
				},
			},
			// ExecDangerously is disabled by default for security
			// ExecDangerouslyはセキュリティのためデフォルトで無効
			ExecDangerously: ExecDangerouslyConfig{
				Enabled:  false,
				Commands: make(map[string][]string),
			},
			// HostPathMasking is enabled by default for security
			// HostPathMaskingはセキュリティのためデフォルトで有効
			HostPathMasking: HostPathMaskingConfig{
				Enabled:     true,
				Replacement: "[HOST_PATH]",
			},
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		Audit: AuditConfig{
			Enabled: false,
			File:    "",
			Events: AuditEvents{
				ToolCalls:         true,
				AccessDenied:      true,
				ClientConnections: true,
				SecurityPolicy:    false,
			},
		},
		CLI: CLIConfig{
			CurrentContainer: CurrentContainerConfig{
				Enabled: true, // Default: enabled (recommended for sandbox environments)
			},
		},
	}
}

// Load loads configuration from a file.
// If configPath is empty, it searches for configuration in common locations.
//
// Loadはファイルから設定を読み込みます。
// configPathが空の場合、一般的な場所で設定を検索します。
//
// Search order when configPath is empty:
// configPathが空の場合の検索順序：
//   1. ./dkmcp.yaml or ./dkmcp.yml
//   2. ./configs/dkmcp.yaml or ./configs/dkmcp.yml
//   3. ~/.dkmcp/dkmcp.yaml or ~/.dkmcp/dkmcp.yml
//
// Returns default configuration if no config file is found.
// 設定ファイルが見つからない場合はデフォルト設定を返します。
func Load(configPath string) (*Config, error) {
	// Start with default configuration
	// デフォルト設定から開始
	cfg := NewDefaultConfig()
	var fileToRead string

	if configPath != "" {
		// Use explicitly specified config file
		// 明示的に指定された設定ファイルを使用
		fileToRead = configPath
	} else {
		// Search for config in common locations
		// 一般的な場所で設定を検索
		searchPaths := []string{".", "./configs"}
		if home, err := os.UserHomeDir(); err == nil {
			searchPaths = append(searchPaths, filepath.Join(home, ".dkmcp"))
		}

		// Try each search path with both .yaml and .yml extensions
		// 各検索パスで.yamlと.yml両方の拡張子を試行
		for _, p := range searchPaths {
			for _, ext := range []string{"yaml", "yml"} {
				f := filepath.Join(p, "dkmcp."+ext)
				if _, err := os.Stat(f); err == nil {
					fileToRead = f
					break
				}
			}
			if fileToRead != "" {
				break
			}
		}
	}

	// Load configuration from file if found
	// ファイルが見つかった場合は設定を読み込み
	if fileToRead != "" {
		data, err := os.ReadFile(fileToRead)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", fileToRead, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", fileToRead, err)
		}
	}

	// Validate configuration before returning
	// 返す前に設定を検証
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks that the configuration is valid.
// Returns an error describing the first validation failure found.
//
// Validateは設定が有効かどうかをチェックします。
// 最初に見つかった検証エラーを説明するエラーを返します。
func (c *Config) Validate() error {
	// Validate server port range (1-65535)
	// サーバーポート範囲を検証（1-65535）
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Server.Port)
	}

	// Validate security mode
	// セキュリティモードを検証
	validModes := map[string]bool{
		"strict":     true,
		"moderate":   true,
		"permissive": true,
	}
	if !validModes[c.Security.Mode] {
		return fmt.Errorf("invalid security mode: %s (must be strict, moderate, or permissive)", c.Security.Mode)
	}

	// Validate logging level
	// ログレベルを検証
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	return nil
}

// GetAddress returns the complete server address in "host:port" format.
// This is used when starting the HTTP server.
//
// GetAddressは"host:port"形式の完全なサーバーアドレスを返します。
// これはHTTPサーバーを起動する際に使用されます。
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
