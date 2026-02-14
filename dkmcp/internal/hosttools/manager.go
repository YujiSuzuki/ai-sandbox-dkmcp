package hosttools

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// Manager coordinates tool discovery and execution for host tools.
// Supports two modes:
//   - Legacy mode: tools loaded from Directories (when ApprovedDir is empty)
//   - Secure mode: tools loaded from approved directory + optional common directory
//
// In secure mode, --dev flag enables development mode where staging directories
// are also included with highest priority (staging > approved > common).
//
// Managerはホストツールの検出と実行を調整します。
// 2つのモードをサポートします:
//   - レガシーモード: Directoriesからツールを読み込み（ApprovedDirが空の場合）
//   - セキュアモード: 承認済みディレクトリ + オプションの共通ディレクトリからツールを読み込み
//
// セキュアモードでは、--devフラグで開発モードを有効にでき、
// ステージングディレクトリも最優先で読み込まれます（staging > approved > common）。
type Manager struct {
	config        *config.HostToolsConfig
	workspaceRoot string
	devMode       bool
}

// NewManager creates a new host tools manager.
// NewManagerは新しいホストツールマネージャーを作成します。
func NewManager(cfg *config.HostToolsConfig, workspaceRoot string) *Manager {
	return &Manager{
		config:        cfg,
		workspaceRoot: workspaceRoot,
	}
}

// IsEnabled returns whether host tools are enabled.
// IsEnabledはホストツールが有効かどうかを返します。
func (m *Manager) IsEnabled() bool {
	return m.config != nil && m.config.Enabled
}

// IsSecureMode returns whether secure mode is active.
// IsSecureModeはセキュアモードが有効かどうかを返します。
func (m *Manager) IsSecureMode() bool {
	return m.config != nil && m.config.IsSecureMode()
}

// SetDevMode enables development mode.
// In dev mode, staging directories are included with highest priority,
// allowing tools under development to be tested without approval.
//
// SetDevModeは開発モードを有効にします。
// 開発モードでは、ステージングディレクトリが最優先で読み込まれ、
// 承認なしで開発中のツールをテストできます。
func (m *Manager) SetDevMode(enabled bool) {
	m.devMode = enabled
}

// IsDevMode returns whether development mode is active.
// IsDevModeは開発モードが有効かどうかを返します。
func (m *Manager) IsDevMode() bool {
	return m.devMode
}

// toolDirs returns the directories to load tools from based on the current mode.
// toolDirsは現在のモードに基づいてツールを読み込むディレクトリを返します。
func (m *Manager) toolDirs() ([]string, error) {
	if !m.config.IsSecureMode() {
		// Legacy mode: use configured directories
		// レガシーモード: 設定されたディレクトリを使用
		var dirs []string
		for _, dir := range m.config.Directories {
			dirs = append(dirs, m.resolveDir(dir))
		}
		return dirs, nil
	}

	// Secure mode: use approved directory + optional common
	// Dev mode adds staging dirs with highest priority
	//
	// セキュアモード: 承認済みディレクトリ + オプションの共通ディレクトリ
	// 開発モードではステージングディレクトリを最優先で追加
	var dirs []string

	// Dev mode: staging directories first (highest priority)
	// 開発モード: ステージングディレクトリを最優先
	if m.devMode {
		stagingDirs := m.config.StagingDirs
		if len(stagingDirs) == 0 {
			stagingDirs = m.config.Directories
		}
		for _, dir := range stagingDirs {
			dirs = append(dirs, m.resolveDir(dir))
		}
	}

	projectDir, err := ProjectApprovedDir(m.config.ApprovedDir, m.workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving approved directory: %w", err)
	}
	dirs = append(dirs, projectDir)

	if m.config.Common {
		commonDir, err := CommonApprovedDir(m.config.ApprovedDir)
		if err != nil {
			slog.Debug("Cannot resolve common directory", "error", err)
		} else {
			dirs = append(dirs, commonDir)
		}
	}

	return dirs, nil
}

// ListTools returns metadata for all discovered tools across all configured directories.
// ListToolsはすべての設定されたディレクトリ内の発見されたツールのメタデータを返します。
func (m *Manager) ListTools() ([]ToolInfo, error) {
	if !m.IsEnabled() {
		return nil, fmt.Errorf("host tools are disabled")
	}

	dirs, err := m.toolDirs()
	if err != nil {
		return nil, err
	}

	var allTools []ToolInfo
	seen := make(map[string]bool) // Deduplicate by name (project takes priority over common)
	for _, dir := range dirs {
		tools, err := ListTools(dir, m.config.AllowedExtensions)
		if err != nil {
			// Skip directories that don't exist or can't be read
			// 存在しないまたは読み取れないディレクトリはスキップ
			continue
		}
		for _, tool := range tools {
			if !seen[tool.Name] {
				seen[tool.Name] = true
				allTools = append(allTools, tool)
			}
		}
	}
	return allTools, nil
}

// GetToolInfo returns detailed info for a specific tool by name.
// It searches all configured directories.
//
// GetToolInfoは名前で指定されたツールの詳細情報を返します。
// すべての設定されたディレクトリを検索します。
func (m *Manager) GetToolInfo(name string) (ToolInfo, error) {
	if !m.IsEnabled() {
		return ToolInfo{}, fmt.Errorf("host tools are disabled")
	}

	dirs, err := m.toolDirs()
	if err != nil {
		return ToolInfo{}, err
	}

	for _, dir := range dirs {
		info, err := GetToolInfo(dir, name, m.config.AllowedExtensions)
		if err == nil {
			return info, nil
		}
	}
	return ToolInfo{}, fmt.Errorf("tool not found: %s", name)
}

// RunTool executes a tool by name with the given arguments.
// It searches all configured directories for the tool.
//
// RunToolは名前で指定されたツールを引数付きで実行します。
// すべての設定されたディレクトリでツールを検索します。
func (m *Manager) RunTool(name string, args []string) (*Result, error) {
	if !m.IsEnabled() {
		return nil, fmt.Errorf("host tools are disabled")
	}

	timeout := time.Duration(m.config.Timeout) * time.Second

	dirs, err := m.toolDirs()
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		// Check if tool exists in this directory via GetToolInfo
		// このディレクトリにツールが存在するかGetToolInfoで確認
		_, err := GetToolInfo(dir, name, m.config.AllowedExtensions)
		if err != nil {
			continue
		}
		return RunTool(dir, name, args, timeout, m.workspaceRoot)
	}
	return nil, fmt.Errorf("tool not found: %s", name)
}

// resolveDir resolves a directory path relative to workspaceRoot.
// resolveDirはworkspaceRootからの相対ディレクトリパスを解決します。
func (m *Manager) resolveDir(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(m.workspaceRoot, dir)
}
