// current.go provides utilities for managing the "current container" state.
// This feature allows users to set a default container for CLI commands,
// reducing the need to specify the container name for every command.
//
// current.goは「カレントコンテナ」状態を管理するユーティリティを提供します。
// この機能により、ユーザーはCLIコマンドのデフォルトコンテナを設定でき、
// 毎回コンテナ名を指定する必要がなくなります。
//
// Design philosophy:
// - AI (MCP/client): Uses explicit parameters, no convenience features needed
// - Human (direct commands): Uses convenience features for better UX
//
// 設計思想:
// - AI (MCP/client): 明示的なパラメータを使用、利便性機能は不要
// - 人 (直接コマンド): より良いUXのために利便性機能を使用
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// CurrentContainerStateFile is the default filename for storing current container state.
// CurrentContainerStateFileはカレントコンテナ状態を保存するデフォルトのファイル名です。
const CurrentContainerStateFile = "current"

// GetCurrentContainerDir returns the directory for storing DockMCP state files.
// Returns ~/.dkmcp by default.
//
// GetCurrentContainerDirはDockMCP状態ファイルを保存するディレクトリを返します。
// デフォルトは~/.dkmcpです。
func GetCurrentContainerDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".dkmcp"), nil
}

// GetCurrentContainerPath returns the full path to the current container state file.
// Returns ~/.dkmcp/current by default.
//
// GetCurrentContainerPathはカレントコンテナ状態ファイルのフルパスを返します。
// デフォルトは~/.dkmcp/currentです。
func GetCurrentContainerPath() (string, error) {
	dir, err := GetCurrentContainerDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, CurrentContainerStateFile), nil
}

// ReadCurrentContainer reads the current container from the state file.
// Returns an empty string if no current container is set or file doesn't exist.
//
// ReadCurrentContainerは状態ファイルからカレントコンテナを読み取ります。
// カレントコンテナが設定されていないかファイルが存在しない場合は空文字列を返します。
func ReadCurrentContainer() (string, error) {
	path, err := GetCurrentContainerPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No current container set
		}
		return "", fmt.Errorf("failed to read current container: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// WriteCurrentContainer writes the current container to the state file.
// Creates the state directory if it doesn't exist.
//
// WriteCurrentContainerはカレントコンテナを状態ファイルに書き込みます。
// 状態ディレクトリが存在しない場合は作成します。
func WriteCurrentContainer(container string) error {
	dir, err := GetCurrentContainerDir()
	if err != nil {
		return err
	}

	// Create state directory if it doesn't exist
	// 状態ディレクトリが存在しない場合は作成
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	path, err := GetCurrentContainerPath()
	if err != nil {
		return err
	}

	// Write container name to file
	// コンテナ名をファイルに書き込み
	if err := os.WriteFile(path, []byte(container+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write current container: %w", err)
	}

	return nil
}

// ClearCurrentContainer removes the current container state file.
// No error is returned if the file doesn't exist.
//
// ClearCurrentContainerはカレントコンテナ状態ファイルを削除します。
// ファイルが存在しない場合はエラーを返しません。
func ClearCurrentContainer() error {
	path, err := GetCurrentContainerPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already cleared
		}
		return fmt.Errorf("failed to clear current container: %w", err)
	}

	return nil
}

// IsCurrentContainerEnabled checks if the current container feature is enabled in config.
// Loads the configuration and returns the enabled state.
//
// IsCurrentContainerEnabledは設定でカレントコンテナ機能が有効かどうかをチェックします。
// 設定を読み込み、有効状態を返します。
func IsCurrentContainerEnabled() (bool, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg.CLI.CurrentContainer.Enabled, nil
}

// GetContainerOrCurrent returns the specified container or falls back to the current container.
// If container is not empty, it's returned as-is.
// If container is empty and current container feature is enabled, the current container is returned.
// Returns an error if container is empty and no current container is set or feature is disabled.
//
// GetContainerOrCurrentは指定されたコンテナを返すか、カレントコンテナにフォールバックします。
// containerが空でない場合はそのまま返します。
// containerが空でカレントコンテナ機能が有効な場合はカレントコンテナを返します。
// containerが空でカレントコンテナが設定されていないか機能が無効な場合はエラーを返します。
func GetContainerOrCurrent(container string) (string, error) {
	// If container is specified, use it directly
	// コンテナが指定されている場合はそのまま使用
	if container != "" {
		return container, nil
	}

	// Check if current container feature is enabled
	// カレントコンテナ機能が有効かチェック
	enabled, err := IsCurrentContainerEnabled()
	if err != nil {
		return "", err
	}

	if !enabled {
		return "", fmt.Errorf("container is required (current_container feature is disabled in config)")
	}

	// Read current container
	// カレントコンテナを読み取り
	current, err := ReadCurrentContainer()
	if err != nil {
		return "", err
	}

	if current == "" {
		return "", fmt.Errorf("no current container set. Use 'dkmcp use <container>' to set one")
	}

	return current, nil
}
