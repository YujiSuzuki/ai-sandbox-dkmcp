// use.go implements the 'use' command for managing the current container.
// This command allows users to set, display, or clear the current container
// for use with other CLI commands like logs, stats, and exec.
//
// use.goはカレントコンテナを管理する'use'コマンドを実装します。
// このコマンドにより、ユーザーはlogs、stats、execなどの他のCLIコマンドで使用する
// カレントコンテナを設定、表示、またはクリアできます。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// useClear clears the current container when set.
	// useClearが設定されている場合、カレントコンテナをクリアします。
	useClear bool
)

// useCmd represents the 'use' command for managing current container.
// - `dkmcp use <container>`: Set current container
// - `dkmcp use`: Show current container
// - `dkmcp use --clear`: Clear current container
//
// useCmdはカレントコンテナを管理する'use'コマンドを表します。
// - `dkmcp use <container>`: カレントコンテナを設定
// - `dkmcp use`: カレントコンテナを表示
// - `dkmcp use --clear`: カレントコンテナをクリア
var useCmd = &cobra.Command{
	Use:   "use [container]",
	Short: "Set, show, or clear the current container",
	Long: `Manage the current container for CLI commands.

This is a human convenience feature. When a current container is set,
commands like 'logs', 'stats', and 'exec' can be used without specifying
the container name.

Examples:
  dkmcp use securenote-api    # Set current container
  dkmcp use                   # Show current container
  dkmcp use --clear           # Clear current container

Note: This feature can be disabled in the config file:
  cli:
    current_container:
      enabled: false`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUse,
}

// init registers the use command and its flags with the root command.
// This function is automatically called when the package is imported.
//
// initはuseコマンドとそのフラグをルートコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	useCmd.Flags().BoolVar(&useClear, "clear", false, "Clear the current container")
	rootCmd.AddCommand(useCmd)
}

// runUse is the execution function for the use command.
// It handles setting, showing, and clearing the current container.
//
// runUseはuseコマンドの実行関数です。
// カレントコンテナの設定、表示、クリアを処理します。
func runUse(cmd *cobra.Command, args []string) error {
	// Check if current container feature is enabled
	// カレントコンテナ機能が有効かチェック
	enabled, err := IsCurrentContainerEnabled()
	if err != nil {
		return err
	}

	if !enabled {
		return fmt.Errorf("current_container feature is disabled in config")
	}

	// Handle --clear flag
	// --clearフラグを処理
	if useClear {
		if err := ClearCurrentContainer(); err != nil {
			return err
		}
		fmt.Println("Current container cleared.")
		return nil
	}

	// If container argument is provided, set it as current
	// コンテナ引数が提供されている場合、カレントとして設定
	if len(args) == 1 {
		container := args[0]

		// Verify container exists and is accessible
		// コンテナが存在しアクセス可能かを確認
		if err := verifyContainerExists(container); err != nil {
			return err
		}

		if err := WriteCurrentContainer(container); err != nil {
			return err
		}
		fmt.Printf("Current container set to: %s\n", container)
		return nil
	}

	// No arguments: show current container
	// 引数なし: カレントコンテナを表示
	current, err := ReadCurrentContainer()
	if err != nil {
		return err
	}

	if current == "" {
		fmt.Println("No current container set.")
		fmt.Println("Use 'dkmcp use <container>' to set one.")
	} else {
		fmt.Printf("Current container: %s\n", current)
	}

	return nil
}

// verifyContainerExists checks if the container exists and is accessible.
// It uses the DirectBackend to list containers and verify the name.
//
// verifyContainerExistsはコンテナが存在しアクセス可能かをチェックします。
// DirectBackendを使用してコンテナをリストし、名前を確認します。
func verifyContainerExists(containerName string) error {
	backend, err := NewDirectBackend()
	if err != nil {
		return err
	}
	defer backend.Close()

	ctx := context.Background()
	containers, err := backend.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		if c.Name == containerName {
			return nil
		}
	}

	return fmt.Errorf("container '%s' not found or not accessible", containerName)
}
