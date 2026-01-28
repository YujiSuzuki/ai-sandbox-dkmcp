// blocked_paths.go provides file path blocking functionality for DockMCP.
// It prevents AI assistants from reading sensitive files like secrets,
// API keys, and configuration files.
//
// blocked_paths.goはDockMCPのファイルパスブロック機能を提供します。
// AIアシスタントがシークレット、APIキー、設定ファイルなどの
// 機密ファイルを読むことを防ぎます。
//
// Blocked paths can come from multiple sources:
// ブロックパスは複数のソースから取得できます：
//   - Manual configuration in dkmcp.yaml (dkmcp.yamlでの手動設定)
//   - Global patterns (グローバルパターン)
//   - Auto-import from DevContainer configs (DevContainer設定からの自動インポート)
//   - Claude Code settings files (Claude Code設定ファイル)
package security

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// BlockedPath represents a blocked file path with metadata about why it's blocked.
// This information is used for debugging and to provide meaningful error messages.
//
// BlockedPathはブロックされたファイルパスとブロック理由のメタデータを表します。
// この情報はデバッグと意味のあるエラーメッセージの提供に使用されます。
type BlockedPath struct {
	// Container is the container name this path applies to, or "*" for all
	// Containerはこのパスが適用されるコンテナ名、または全てに適用する場合は"*"
	Container string `json:"container"`

	// Pattern is the file path pattern to block (supports globs)
	// Patternはブロックするファイルパスパターン（globをサポート）
	Pattern string `json:"pattern"`

	// Reason explains why this path is blocked (e.g., "manual_block", "global_pattern")
	// Reasonはこのパスがブロックされている理由を説明します（例: "manual_block", "global_pattern"）
	Reason string `json:"reason"`

	// Source is the file where this block was defined (e.g., "dkmcp.yaml")
	// Sourceはこのブロックが定義されたファイルです（例: "dkmcp.yaml"）
	Source string `json:"source,omitempty"`

	// SourceLine is the line number in the source file (if available)
	// SourceLineはソースファイル内の行番号です（利用可能な場合）
	SourceLine int `json:"source_line,omitempty"`

	// OriginalPath is the original path before normalization (for debugging)
	// OriginalPathは正規化前の元のパスです（デバッグ用）
	OriginalPath string `json:"original_path,omitempty"`
}

// BlockedPathsManager manages blocked file paths for all containers.
// It loads paths from configuration and provides path checking functionality.
//
// BlockedPathsManagerは全コンテナのブロックファイルパスを管理します。
// 設定からパスをロードし、パスチェック機能を提供します。
type BlockedPathsManager struct {
	// config holds the blocked paths configuration
	// configはブロックパス設定を保持します
	config *config.BlockedPathsConfig

	// blockedPaths is the list of all blocked paths from all sources
	// blockedPathsは全ソースからの全ブロックパスのリストです
	blockedPaths []BlockedPath

	// containers is the list of known container names for pattern matching
	// containersはパターンマッチング用の既知のコンテナ名リストです
	containers []string
}

// NewBlockedPathsManager creates a new blocked paths manager.
// The containers parameter is used to match container names in path patterns.
//
// NewBlockedPathsManagerは新しいブロックパスマネージャを作成します。
// containersパラメータはパスパターン内のコンテナ名のマッチングに使用されます。
func NewBlockedPathsManager(cfg *config.BlockedPathsConfig, containers []string) *BlockedPathsManager {
	return &BlockedPathsManager{
		config:       cfg,
		blockedPaths: []BlockedPath{},
		containers:   containers,
	}
}

// LoadBlockedPaths loads all blocked paths from configuration and auto-import sources.
// This should be called once during initialization.
//
// LoadBlockedPathsは設定と自動インポートソースから全てのブロックパスをロードします。
// これは初期化時に一度呼び出す必要があります。
func (m *BlockedPathsManager) LoadBlockedPaths() error {
	// Load manual blocked paths from configuration
	// 設定から手動ブロックパスをロード
	for container, patterns := range m.config.Manual {
		for _, pattern := range patterns {
			m.blockedPaths = append(m.blockedPaths, BlockedPath{
				Container: container,
				Pattern:   pattern,
				Reason:    "manual_block",
				Source:    "dkmcp.yaml",
			})
		}
	}

	// Load global patterns (apply to all containers)
	// グローバルパターンをロード（全コンテナに適用）
	for _, pattern := range m.config.AutoImport.GlobalPatterns {
		m.blockedPaths = append(m.blockedPaths, BlockedPath{
			Container: "*",
			Pattern:   pattern,
			Reason:    "global_pattern",
			Source:    "dkmcp.yaml",
		})
	}

	// Auto-import from DevContainer configs if enabled
	// 有効な場合はDevContainer設定から自動インポート
	if m.config.AutoImport.Enabled {
		if err := m.autoImportBlockedPaths(); err != nil {
			return fmt.Errorf("auto-import failed: %w", err)
		}
	}

	return nil
}

// autoImportBlockedPaths scans DevContainer and Claude Code config files for blocked paths.
// It detects patterns like /dev/null mounts and tmpfs mounts that hide files from AI.
//
// autoImportBlockedPathsはDevContainerとClaude Code設定ファイルからブロックパスをスキャンします。
// AIからファイルを隠す/dev/nullマウントやtmpfsマウントなどのパターンを検出します。
func (m *BlockedPathsManager) autoImportBlockedPaths() error {
	workspaceRoot := m.config.AutoImport.WorkspaceRoot
	if workspaceRoot == "" {
		workspaceRoot = "."
	}

	// Scan DevContainer config files (docker-compose.yml, devcontainer.json)
	// DevContainer設定ファイルをスキャン（docker-compose.yml, devcontainer.json）
	for _, scanFile := range m.config.AutoImport.ScanFiles {
		fullPath := filepath.Join(workspaceRoot, scanFile)

		// Skip if file doesn't exist
		// ファイルが存在しない場合はスキップ
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}

		// Parse Docker Compose files
		// Docker Composeファイルを解析
		if strings.HasSuffix(scanFile, "docker-compose.yml") || strings.HasSuffix(scanFile, "docker-compose.yaml") {
			if err := m.parseDockerCompose(fullPath); err != nil {
				return fmt.Errorf("failed to parse %s: %w", fullPath, err)
			}
		} else if strings.HasSuffix(scanFile, "devcontainer.json") {
			// Parse DevContainer JSON files
			// DevContainer JSONファイルを解析
			if err := m.parseDevcontainerJSON(fullPath); err != nil {
				return fmt.Errorf("failed to parse %s: %w", fullPath, err)
			}
		}
	}

	// Scan Claude Code settings files if enabled
	// 有効な場合はClaude Code設定ファイルをスキャン
	if m.config.AutoImport.ClaudeCodeSettings.Enabled {
		if err := m.scanClaudeCodeSettings(workspaceRoot); err != nil {
			return fmt.Errorf("failed to scan Claude Code settings: %w", err)
		}
	}

	// Scan Gemini settings files if enabled
	// 有効な場合はGemini設定ファイルをスキャン
	if m.config.AutoImport.GeminiSettings.Enabled {
		if err := m.scanGeminiSettings(workspaceRoot); err != nil {
			return fmt.Errorf("failed to scan Gemini settings: %w", err)
		}
	}

	return nil
}

// scanClaudeCodeSettings scans for Claude Code settings files with depth limit.
// It recursively searches directories up to the configured max depth.
//
// scanClaudeCodeSettingsは深度制限付きでClaude Code設定ファイルをスキャンします。
// 設定された最大深度まで再帰的にディレクトリを検索します。
func (m *BlockedPathsManager) scanClaudeCodeSettings(workspaceRoot string) error {
	maxDepth := m.config.AutoImport.ClaudeCodeSettings.MaxDepth
	settingsFiles := m.config.AutoImport.ClaudeCodeSettings.SettingsFiles

	// Log scan parameters for debugging
	// デバッグ用にスキャンパラメータをログ出力
	absWorkspaceRoot, _ := filepath.Abs(workspaceRoot)
	slog.Debug("Scanning Claude Code settings",
		"workspace_root", absWorkspaceRoot,
		"max_depth", maxDepth,
		"settings_files", settingsFiles)

	// Scan workspace root (depth 0)
	// ワークスペースルートをスキャン（深度0）
	for _, settingsFile := range settingsFiles {
		fullPath := filepath.Join(workspaceRoot, settingsFile)
		absPath, _ := filepath.Abs(fullPath)
		if _, err := os.Stat(fullPath); err == nil {
			slog.Debug("Found Claude Code settings file", "path", absPath, "depth", 0)
			if err := m.parseClaudeCodeSettings(fullPath); err != nil {
				return fmt.Errorf("failed to parse %s: %w", fullPath, err)
			}
		} else {
			slog.Debug("Claude Code settings file not found", "path", absPath, "depth", 0)
		}
	}

	// Scan subdirectories if max_depth > 0
	// max_depth > 0の場合はサブディレクトリをスキャン
	if maxDepth > 0 {
		if err := m.scanSettingsAtDepth(workspaceRoot, settingsFiles, 1, maxDepth, "Claude Code", m.parseClaudeCodeSettings); err != nil {
			return err
		}
	}

	return nil
}

// settingsParseFunc is a callback for parsing a settings file at a given path.
// settingsParseFuncは指定されたパスの設定ファイルを解析するためのコールバックです。
type settingsParseFunc func(filePath string) error

// scanSettingsAtDepth recursively scans subdirectories for settings files,
// delegating parsing to the provided callback function.
// settingsType is used for debug log messages (e.g., "Claude Code", "Gemini").
//
// scanSettingsAtDepthはサブディレクトリを再帰的にスキャンして設定ファイルを探し、
// 解析を指定されたコールバック関数に委譲します。
// settingsTypeはデバッグログメッセージに使用されます（例: "Claude Code", "Gemini"）。
func (m *BlockedPathsManager) scanSettingsAtDepth(
	dir string, settingsFiles []string,
	currentDepth, maxDepth int,
	settingsType string,
	parseFunc settingsParseFunc,
) error {
	// Stop if we've exceeded max depth
	// 最大深度を超えた場合は停止
	if currentDepth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Debug("Cannot read directory", "dir", dir, "error", err)
		return nil // Skip unreadable directories / 読み取れないディレクトリはスキップ
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories and common non-project directories
		// 隠しディレクトリと一般的な非プロジェクトディレクトリをスキップ
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
			continue
		}

		subDir := filepath.Join(dir, name)
		slog.Debug("Scanning subdirectory for settings", "type", settingsType, "dir", subDir, "depth", currentDepth)

		// Check for settings files in this subdirectory
		// このサブディレクトリの設定ファイルをチェック
		for _, settingsFile := range settingsFiles {
			fullPath := filepath.Join(subDir, settingsFile)
			absPath, _ := filepath.Abs(fullPath)
			if _, err := os.Stat(fullPath); err == nil {
				slog.Debug("Found settings file", "type", settingsType, "path", absPath, "depth", currentDepth)
				if err := parseFunc(fullPath); err != nil {
					return fmt.Errorf("failed to parse %s: %w", fullPath, err)
				}
			}
		}

		// Recurse deeper if not at max depth
		// 最大深度でなければより深く再帰
		if currentDepth < maxDepth {
			if err := m.scanSettingsAtDepth(subDir, settingsFiles, currentDepth+1, maxDepth, settingsType, parseFunc); err != nil {
				return err
			}
		}
	}

	return nil
}

// DockerComposeConfig represents a partial docker-compose.yml structure.
// Only the fields needed for blocked path detection are included.
//
// DockerComposeConfigはdocker-compose.ymlの部分的な構造を表します。
// ブロックパス検出に必要なフィールドのみが含まれています。
type DockerComposeConfig struct {
	// Services maps service names to their configurations
	// Servicesはサービス名からその設定へのマップです
	Services map[string]DockerComposeService `yaml:"services"`
}

// DockerComposeService represents a service in docker-compose.yml.
// We only need volumes and tmpfs for blocked path detection.
//
// DockerComposeServiceはdocker-compose.ymlのサービスを表します。
// ブロックパス検出にはvolumesとtmpfsのみが必要です。
type DockerComposeService struct {
	// Volumes contains volume mount specifications
	// Volumesはボリュームマウント仕様を含みます
	Volumes []string `yaml:"volumes"`

	// Tmpfs contains tmpfs mount paths
	// Tmpfsはtmpfsマウントパスを含みます
	Tmpfs []string `yaml:"tmpfs"`
}

// parseDockerCompose parses a docker-compose.yml file for blocked paths.
// It detects /dev/null volume mounts and tmpfs mounts that hide files.
//
// parseDockerComposeはdocker-compose.ymlファイルからブロックパスを解析します。
// ファイルを隠す/dev/nullボリュームマウントとtmpfsマウントを検出します。
func (m *BlockedPathsManager) parseDockerCompose(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var compose DockerComposeConfig
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return err
	}

	for serviceName, service := range compose.Services {
		// Check volumes for /dev/null mounts (files hidden from AI)
		// /dev/nullマウントをチェック（AIから隠されたファイル）
		for _, volume := range service.Volumes {
			if strings.HasPrefix(volume, "/dev/null:") {
				// Extract the target path from the mount specification
				// マウント仕様からターゲットパスを抽出
				parts := strings.Split(volume, ":")
				if len(parts) >= 2 {
					targetPath := parts[1]
					blocked := m.extractBlockedPath(targetPath, filePath, "volume_mount_to_dev_null")
					if blocked != nil {
						m.blockedPaths = append(m.blockedPaths, *blocked)
					}
				}
			}
		}

		// Check tmpfs mounts (directories hidden from AI)
		// tmpfsマウントをチェック（AIから隠されたディレクトリ）
		for _, tmpfs := range service.Tmpfs {
			// tmpfs can be just a path or "path:options"
			// tmpfsはパスのみまたは"path:options"形式
			path := strings.Split(tmpfs, ":")[0]
			blocked := m.extractBlockedPath(path, filePath, "tmpfs_mount")
			if blocked != nil {
				m.blockedPaths = append(m.blockedPaths, *blocked)
			}
		}

		_ = serviceName // May use later for logging / ログ用に後で使用する可能性あり
	}

	return nil
}

// ClaudeCodeSettings represents the Claude Code settings.json structure.
// This is used to import blocked paths from Claude Code configuration.
//
// ClaudeCodeSettingsはClaude Codeのsettings.json構造を表します。
// これはClaude Code設定からブロックパスをインポートするために使用されます。
type ClaudeCodeSettings struct {
	// Permissions contains allow/deny lists for operations
	// Permissionsは操作の許可/拒否リストを含みます
	Permissions ClaudeCodePermissions `json:"permissions"`
}

// ClaudeCodePermissions represents the permissions section in Claude Code settings.
// The deny list contains patterns that should be blocked.
//
// ClaudeCodePermissionsはClaude Code設定のpermissionsセクションを表します。
// denyリストにはブロックすべきパターンが含まれます。
type ClaudeCodePermissions struct {
	// Deny contains patterns that should be denied (e.g., "Read(.env)")
	// Denyには拒否すべきパターンが含まれます（例: "Read(.env)"）
	Deny []string `json:"deny"`

	// Allow contains patterns that are explicitly allowed
	// Allowには明示的に許可されたパターンが含まれます
	Allow []string `json:"allow"`
}

// parseClaudeCodeSettings parses a Claude Code settings.json file for blocked paths.
// It extracts Read() patterns from the deny list.
//
// parseClaudeCodeSettingsはClaude Codeのsettings.jsonファイルからブロックパスを解析します。
// denyリストからRead()パターンを抽出します。
func (m *BlockedPathsManager) parseClaudeCodeSettings(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var settings ClaudeCodeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	// Regex to extract file patterns from Read() deny rules
	// Read()拒否ルールからファイルパターンを抽出する正規表現
	readPattern := regexp.MustCompile(`^Read\(([^)]+)\)$`)

	importedCount := 0
	for _, deny := range settings.Permissions.Deny {
		matches := readPattern.FindStringSubmatch(deny)
		if len(matches) < 2 {
			continue // Not a Read() pattern, skip / Read()パターンでない、スキップ
		}

		pattern := matches[1]

		// Convert Claude Code pattern to blocked path
		// Claude CodeパターンをBlockedPathに変換
		blocked := m.convertClaudeCodePattern(pattern, filePath)
		if blocked != nil {
			m.blockedPaths = append(m.blockedPaths, *blocked)
			importedCount++
		}
	}

	if importedCount > 0 {
		slog.Debug("Imported blocked patterns from Claude Code settings",
			"file", filePath,
			"patterns_count", importedCount)
	}

	return nil
}

// convertClaudeCodePattern converts a Claude Code Read() pattern to a BlockedPath.
// It attempts to identify the container from the path and normalizes the pattern.
//
// convertClaudeCodePatternはClaude CodeのRead()パターンをBlockedPathに変換します。
// パスからコンテナを識別し、パターンを正規化しようとします。
func (m *BlockedPathsManager) convertClaudeCodePattern(pattern string, source string) *BlockedPath {
	// Remove leading ./ if present
	// 先頭の./があれば削除
	pattern = strings.TrimPrefix(pattern, "./")

	// Try to find container name in the pattern path
	// パターンパス内のコンテナ名を探す
	var containerName string
	var filePattern string

	pathParts := strings.Split(pattern, "/")
	for i, part := range pathParts {
		// Skip glob patterns
		// globパターンはスキップ
		if part == "**" || strings.Contains(part, "*") {
			continue
		}
		// Check if this part matches a known container name
		// この部分が既知のコンテナ名と一致するかチェック
		for _, container := range m.containers {
			if part == container || matchesPattern(part, container) {
				containerName = part
				filePattern = strings.Join(pathParts[i+1:], "/")
				break
			}
		}
		if containerName != "" {
			break
		}
	}

	// If no container found, treat as global pattern
	// コンテナが見つからない場合はグローバルパターンとして扱う
	if containerName == "" {
		containerName = "*"
		filePattern = pattern
	}

	// Handle empty filePattern
	// 空のfilePatternを処理
	if filePattern == "" {
		filePattern = pattern
	}

	return &BlockedPath{
		Container:    containerName,
		Pattern:      filePattern,
		Reason:       "claude_code_settings_deny",
		Source:       source,
		OriginalPath: pattern,
	}
}

// parseDevcontainerJSON parses devcontainer.json for blocked paths.
// It detects mount configurations that hide files from AI.
//
// parseDevcontainerJSONはdevcontainer.jsonからブロックパスを解析します。
// AIからファイルを隠すマウント設定を検出します。
func (m *BlockedPathsManager) parseDevcontainerJSON(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Use regex-based parsing for mount configurations
	// Looking for patterns like:
	//   "source=/dev/null,target=/workspace/..."
	//   "type=tmpfs,target=/workspace/..."
	//
	// マウント設定の正規表現ベース解析を使用
	// 以下のようなパターンを探します：
	//   "source=/dev/null,target=/workspace/..."
	//   "type=tmpfs,target=/workspace/..."
	content := string(data)

	// Pattern for /dev/null bind mounts
	// /dev/nullバインドマウントのパターン
	devNullPattern := regexp.MustCompile(`source=/dev/null[^"]*target=([^,"]+)`)
	matches := devNullPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			targetPath := match[1]
			blocked := m.extractBlockedPath(targetPath, filePath, "devcontainer_bind_mount")
			if blocked != nil {
				m.blockedPaths = append(m.blockedPaths, *blocked)
			}
		}
	}

	// Pattern for tmpfs mounts
	// tmpfsマウントのパターン
	tmpfsPattern := regexp.MustCompile(`type=(?:tmpfs|volume)[^"]*target=([^,"]+)`)
	matches = tmpfsPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			targetPath := match[1]
			blocked := m.extractBlockedPath(targetPath, filePath, "devcontainer_tmpfs_mount")
			if blocked != nil {
				m.blockedPaths = append(m.blockedPaths, *blocked)
			}
		}
	}

	return nil
}

// extractBlockedPath extracts container and path from a DevContainer mount path.
// It attempts to identify which container the path belongs to.
//
// extractBlockedPathはDevContainerマウントパスからコンテナとパスを抽出します。
// パスがどのコンテナに属するかを識別しようとします。
func (m *BlockedPathsManager) extractBlockedPath(path string, source string, reason string) *BlockedPath {
	// Path format: /workspace/some-project/container-name/path/to/file
	// パス形式: /workspace/some-project/container-name/path/to/file
	pathParts := strings.Split(path, "/")

	// Try to find a container name in the path
	// パス内のコンテナ名を探す
	var containerName string
	var remainingPath string

	for i, part := range pathParts {
		for _, container := range m.containers {
			if part == container || matchesPattern(part, container) {
				containerName = part
				remainingPath = "/" + strings.Join(pathParts[i+1:], "/")
				break
			}
		}
		if containerName != "" {
			break
		}
	}

	// If no container found, extract filename pattern for global blocking
	// コンテナが見つからない場合はグローバルブロック用のファイル名パターンを抽出
	if containerName == "" {
		pattern := filepath.Base(path)
		if pattern == "" || pattern == "." {
			return nil
		}
		return &BlockedPath{
			Container:    "*",
			Pattern:      pattern,
			Reason:       reason,
			Source:       source,
			OriginalPath: path,
		}
	}

	// Extract the pattern from the remaining path
	// 残りのパスからパターンを抽出
	pattern := remainingPath
	if pattern == "" {
		pattern = "/*" // Block everything in the container root / コンテナルートの全てをブロック
	}

	return &BlockedPath{
		Container:    containerName,
		Pattern:      pattern,
		Reason:       reason,
		Source:       source,
		OriginalPath: path,
	}
}

// matchesPattern checks if a string matches a wildcard pattern.
// Uses filepath.Match for glob pattern matching.
//
// matchesPatternは文字列がワイルドカードパターンに一致するかチェックします。
// globパターンマッチングにfilepath.Matchを使用します。
func matchesPattern(str string, pattern string) bool {
	if !strings.Contains(pattern, "*") {
		return str == pattern
	}
	matched, _ := filepath.Match(pattern, str)
	return matched
}

// IsPathBlocked checks if a file path is blocked for a specific container.
// Returns BlockedPath info if blocked, nil if allowed.
//
// IsPathBlockedはファイルパスが特定のコンテナに対してブロックされているかチェックします。
// ブロックされている場合はBlockedPath情報を返し、許可されている場合はnilを返します。
func (m *BlockedPathsManager) IsPathBlocked(containerName string, path string) *BlockedPath {
	for _, blocked := range m.blockedPaths {
		// Check container match (skip if doesn't match and not global)
		// コンテナマッチをチェック（マッチせずグローバルでもない場合はスキップ）
		if blocked.Container != "*" && blocked.Container != containerName {
			continue
		}

		// Check if path matches the blocked pattern
		// パスがブロックパターンに一致するかチェック
		if m.matchPath(path, blocked.Pattern) {
			return &blocked
		}
	}

	return nil
}

// matchPath checks if a file path matches a blocked pattern.
// Supports exact matches, filename matches, directory patterns, and glob patterns.
//
// matchPathはファイルパスがブロックパターンに一致するかチェックします。
// 完全一致、ファイル名一致、ディレクトリパターン、globパターンをサポートします。
func (m *BlockedPathsManager) matchPath(path string, pattern string) bool {
	// Normalize both paths
	// 両方のパスを正規化
	path = filepath.Clean(path)
	pattern = filepath.Clean(pattern)

	// Direct exact match
	// 直接の完全一致
	if path == pattern {
		return true
	}

	// Filename match (e.g., ".env" matches "/app/.env")
	// ファイル名一致（例: ".env"は"/app/.env"にマッチ）
	if !strings.Contains(pattern, "/") {
		basename := filepath.Base(path)
		if basename == pattern {
			return true
		}
		// Wildcard match for filename (e.g., "*.key" matches "secret.key")
		// ファイル名のワイルドカード一致（例: "*.key"は"secret.key"にマッチ）
		if strings.Contains(pattern, "*") {
			matched, _ := filepath.Match(pattern, basename)
			if matched {
				return true
			}
		}
	}

	// Directory pattern (e.g., "secrets/*" matches "/app/secrets/key.pem")
	// ディレクトリパターン（例: "secrets/*"は"/app/secrets/key.pem"にマッチ）
	if strings.HasSuffix(pattern, "/*") {
		dirPattern := strings.TrimSuffix(pattern, "/*")
		// Check if dirPattern appears at path boundary (start or after /)
		// dirPatternがパス境界（先頭または/の後）に出現するかチェック
		dirWithSlash := dirPattern + "/"
		if strings.HasPrefix(path, dirWithSlash) || strings.Contains(path, "/"+dirWithSlash) {
			return true
		}
	}

	// General filepath pattern match
	// 一般的なfilepathパターンマッチ
	matched, _ := filepath.Match(pattern, path)
	return matched
}

// GetBlockedPaths returns all blocked paths from all sources.
// GetBlockedPathsは全ソースからの全ブロックパスを返します。
func (m *BlockedPathsManager) GetBlockedPaths() []BlockedPath {
	return m.blockedPaths
}

// GetBlockedPathsForContainer returns blocked paths applicable to a specific container.
// Includes both container-specific paths and global patterns (Container = "*").
//
// GetBlockedPathsForContainerは特定のコンテナに適用されるブロックパスを返します。
// コンテナ固有のパスとグローバルパターン（Container = "*"）の両方を含みます。
func (m *BlockedPathsManager) GetBlockedPathsForContainer(containerName string) []BlockedPath {
	var result []BlockedPath
	for _, blocked := range m.blockedPaths {
		// Include if global or container-specific match
		// グローバルまたはコンテナ固有の一致の場合に含める
		if blocked.Container == "*" || blocked.Container == containerName {
			result = append(result, blocked)
		}
	}
	return result
}

// SetContainers updates the known container list.
// This should be called when the container list changes.
//
// SetContainersは既知のコンテナリストを更新します。
// コンテナリストが変更された場合にこれを呼び出す必要があります。
func (m *BlockedPathsManager) SetContainers(containers []string) {
	m.containers = containers
}

// AddBlockedPath adds a blocked path manually.
// This can be used to add paths programmatically.
//
// AddBlockedPathはブロックパスを手動で追加します。
// これはプログラムでパスを追加するために使用できます.
func (m *BlockedPathsManager) AddBlockedPath(blocked BlockedPath) {
	m.blockedPaths = append(m.blockedPaths, blocked)
}

// scanGeminiSettings scans for Gemini Code Assist exclusion files with depth limit.
// It recursively searches directories up to the configured max depth.
// Gemini uses .aiexclude and .geminiignore files with gitignore-style syntax.
//
// scanGeminiSettingsは深度制限付きでGemini Code Assist除外ファイルをスキャンします。
// 設定された最大深度まで再帰的にディレクトリを検索します。
// Geminiはgitignore形式の.aiexcludeと.geminiignoreファイルを使用します。
func (m *BlockedPathsManager) scanGeminiSettings(workspaceRoot string) error {
	maxDepth := m.config.AutoImport.GeminiSettings.MaxDepth
	settingsFiles := m.config.AutoImport.GeminiSettings.SettingsFiles

	// Log scan parameters for debugging
	// デバッグ用にスキャンパラメータをログ出力
	absWorkspaceRoot, _ := filepath.Abs(workspaceRoot)
	slog.Debug("Scanning Gemini settings",
		"workspace_root", absWorkspaceRoot,
		"max_depth", maxDepth,
		"settings_files", settingsFiles)

	// Scan workspace root (depth 0)
	// ワークスペースルートをスキャン（深度0）
	for _, settingsFile := range settingsFiles {
		fullPath := filepath.Join(workspaceRoot, settingsFile)
		absPath, _ := filepath.Abs(fullPath)
		if _, err := os.Stat(fullPath); err == nil {
			slog.Debug("Found Gemini settings file", "path", absPath, "depth", 0)
			if err := m.parseGeminiExcludeFile(fullPath); err != nil {
				return fmt.Errorf("failed to parse %s: %w", fullPath, err)
			}
		} else {
			slog.Debug("Gemini settings file not found", "path", absPath, "depth", 0)
		}
	}

	// Scan subdirectories if max_depth > 0
	// max_depth > 0の場合はサブディレクトリをスキャン
	if maxDepth > 0 {
		if err := m.scanSettingsAtDepth(workspaceRoot, settingsFiles, 1, maxDepth, "Gemini", m.parseGeminiExcludeFile); err != nil {
			return err
		}
	}

	return nil
}


// parseGeminiExcludeFile parses a Gemini .aiexclude or .geminiignore file.
// These files use gitignore-style syntax.
//
// parseGeminiExcludeFileはGeminiの.aiexcludeまたは.geminiignoreファイルを解析します。
// これらのファイルはgitignore形式の構文を使用します。
func (m *BlockedPathsManager) parseGeminiExcludeFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	importedCount := 0

	for _, line := range lines {
		// Trim whitespace
		// ホワイトスペースを削除
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		// 空行とコメントをスキップ
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip negation patterns (lines starting with !)
		// We don't support negation in DockMCP's blocked paths
		// 否定パターン（!で始まる行）をスキップ
		// DockMCPのブロックパスでは否定をサポートしていません
		if strings.HasPrefix(line, "!") {
			slog.Debug("Skipping negation pattern in Gemini exclude file",
				"pattern", line, "file", filePath)
			continue
		}

		// Convert gitignore pattern to blocked path
		// gitignoreパターンをブロックパスに変換
		blocked := m.convertGitignorePattern(line, filePath)
		if blocked != nil {
			m.blockedPaths = append(m.blockedPaths, *blocked)
			importedCount++
		}
	}

	if importedCount > 0 {
		slog.Debug("Imported blocked patterns from Gemini exclude file",
			"file", filePath,
			"patterns_count", importedCount)
	}

	return nil
}

// convertGitignorePattern converts a gitignore-style pattern to a BlockedPath.
// Gitignore patterns are applied globally (to all containers) since they don't
// specify container names.
//
// convertGitignorePatternはgitignore形式のパターンをBlockedPathに変換します。
// gitignoreパターンはコンテナ名を指定しないため、グローバル（全コンテナ）に適用されます。
func (m *BlockedPathsManager) convertGitignorePattern(pattern string, source string) *BlockedPath {
	// Handle directory patterns (ending with /)
	// ディレクトリパターン（/で終わる）を処理
	if strings.HasSuffix(pattern, "/") {
		pattern = pattern + "*"
	}

	// Handle patterns starting with / (relative to root)
	// /で始まるパターン（ルートからの相対パス）を処理
	pattern = strings.TrimPrefix(pattern, "/")

	return &BlockedPath{
		Container:    "*", // Gemini patterns are global / Geminiパターンはグローバル
		Pattern:      pattern,
		Reason:       "gemini_exclude_file",
		Source:       source,
		OriginalPath: pattern,
	}
}
