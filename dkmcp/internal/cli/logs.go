// logs.go implements the 'logs' command for retrieving container logs.
// It uses the DirectBackend to fetch logs from Docker containers.
//
// logs.goはコンテナログを取得する'logs'コマンドを実装します。
// DirectBackendを使用してDockerコンテナからログを取得します。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// logsTail specifies the number of lines to retrieve from the end of the log.
	// Default is "100" lines. Can be set via the --tail or -n flag.
	//
	// logsTailはログの末尾から取得する行数を指定します。
	// デフォルトは"100"行です。--tailまたは-nフラグで設定できます。
	logsTail string
)

// logsCmd represents the 'logs' command for retrieving container logs.
// If no container is specified and current container is set, uses the current container.
//
// logsCmdはコンテナログを取得する'logs'コマンドを表します。
// コンテナが指定されていない場合、カレントコンテナが設定されていればそれを使用します。
var logsCmd = &cobra.Command{
	Use:   "logs [container]",
	Short: "Get logs from a container",
	Long: `Retrieve and display logs from a specified container.

If no container is specified and a current container is set (via 'dkmcp use'),
it will use the current container.

Examples:
  dkmcp logs securenote-api      # Logs from specific container
  dkmcp logs -n 50 securenote-api # Last 50 lines
  dkmcp use securenote-api       # Set current container
  dkmcp logs                     # Logs from current container`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogs,
}

// init registers the logs command and its flags with the root command.
// This function is automatically called when the package is imported.
//
// initはlogsコマンドとそのフラグをルートコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	// Register the --tail flag with -n shorthand.
	// The default value of "100" provides a reasonable amount of recent logs.
	//
	// --tailフラグを-nショートハンドで登録します。
	// デフォルト値"100"は最近のログの妥当な量を提供します。
	logsCmd.Flags().StringVarP(&logsTail, "tail", "n", "100", "Number of lines to show from the end")
	rootCmd.AddCommand(logsCmd)
}

// runLogs is the execution function for the logs command.
// It retrieves logs from the specified container and prints them to stdout.
//
// runLogsはlogsコマンドの実行関数です。
// 指定されたコンテナからログを取得し、stdoutに出力します。
func runLogs(cmd *cobra.Command, args []string) error {
	// Determine container name (from argument or current)
	// コンテナ名を決定（引数またはカレントから）
	var containerArg string
	if len(args) > 0 {
		containerArg = args[0]
	}

	container, err := GetContainerOrCurrent(containerArg)
	if err != nil {
		return err
	}

	// Create a DirectBackend for Docker access.
	// Docker接続用のDirectBackendを作成します。
	backend, err := NewDirectBackend()
	if err != nil {
		return err
	}
	defer backend.Close()

	// Retrieve logs from the container.
	// コンテナからログを取得します。
	ctx := context.Background()
	logs, err := backend.GetLogs(ctx, container, logsTail, "")
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	// Print the logs to stdout.
	// ログをstdoutに出力します。
	fmt.Println(logs)
	return nil
}
