// client_logs.go implements the 'client logs' subcommand for retrieving container logs via HTTP.
// It uses HTTPBackend to communicate with the DockMCP server instead of direct Docker access.
//
// client_logs.goはHTTP経由でコンテナログを取得する'client logs'サブコマンドを実装します。
// 直接Docker接続の代わりにHTTPBackendを使用してDockMCPサーバーと通信します。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// clientLogsTail specifies the number of lines to retrieve from the end of the log.
	// Default is 100 lines. Can be set via the --tail flag.
	//
	// clientLogsTailはログの末尾から取得する行数を指定します。
	// デフォルトは100行です。--tailフラグで設定できます。
	clientLogsTail int

	// clientLogsSince specifies a timestamp to show logs since.
	// Format: ISO 8601 timestamp (e.g., "2024-01-01T00:00:00Z") or relative (e.g., "42m").
	//
	// clientLogsSinceはログを表示する開始タイムスタンプを指定します。
	// フォーマット：ISO 8601タイムスタンプ（例："2024-01-01T00:00:00Z"）または相対時間（例："42m"）。
	clientLogsSince string

	// clientLogsFollow enables follow mode for continuous log output.
	// Note: Follow mode has limited support via HTTP due to request-response nature.
	//
	// clientLogsFollowは継続的なログ出力のフォローモードを有効にします。
	// 注意：HTTPのリクエスト-レスポンスの性質のため、フォローモードのサポートは限定的です。
	clientLogsFollow bool
)

// clientLogsCmd represents the 'client logs' subcommand.
// It retrieves and displays container logs via the DockMCP HTTP server.
// Requires exactly one argument: the container name.
//
// clientLogsCmdは'client logs'サブコマンドを表します。
// DockMCP HTTPサーバー経由でコンテナログを取得して表示します。
// コンテナ名という1つの引数が必要です。
var clientLogsCmd = &cobra.Command{
	Use:   "logs CONTAINER",
	Short: "Get logs from a container via DockMCP server",
	Long:  `Retrieve logs from a Docker container through the DockMCP server.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runClientLogs,
}

// init registers the logs subcommand and its flags with the client command.
// This function is automatically called when the package is imported.
//
// initはlogsサブコマンドとそのフラグをclientコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	// Add logs as a subcommand of client.
	// logsをclientのサブコマンドとして追加します。
	clientCmd.AddCommand(clientLogsCmd)

	// Register flags for log retrieval options.
	// ログ取得オプションのフラグを登録します。
	clientLogsCmd.Flags().IntVar(&clientLogsTail, "tail", 100, "Number of lines to show from the end of logs")
	clientLogsCmd.Flags().StringVar(&clientLogsSince, "since", "", "Show logs since timestamp (e.g., 2024-01-01T00:00:00Z)")
	clientLogsCmd.Flags().BoolVar(&clientLogsFollow, "follow", false, "Follow log output (note: not fully supported via HTTP)")
}

// runClientLogs is the execution function for the client logs subcommand.
// It creates an HTTPBackend, retrieves logs, and displays them.
//
// runClientLogsはclient logsサブコマンドの実行関数です。
// HTTPBackendを作成し、ログを取得して表示します。
func runClientLogs(cmd *cobra.Command, args []string) error {
	// Get the container name from command arguments.
	// コマンド引数からコンテナ名を取得します。
	containerName := args[0]

	// Warn user about follow mode limitations via HTTP.
	// HTTP経由でのフォローモードの制限についてユーザーに警告します。
	if clientLogsFollow {
		fmt.Println("Note: Follow mode is limited via HTTP client. Showing available logs only.")
	}

	// Create an HTTPBackend for remote DockMCP server access.
	// リモートDockMCPサーバーアクセス用のHTTPBackendを作成します。
	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	// Retrieve logs from the container via MCP.
	// Convert tail count to string for the backend API.
	//
	// MCP経由でコンテナからログを取得します。
	// バックエンドAPI用にtailカウントを文字列に変換します。
	ctx := context.Background()
	logs, err := backend.GetLogs(ctx, containerName, fmt.Sprintf("%d", clientLogsTail), clientLogsSince)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	// Handle empty log output.
	// 空のログ出力を処理します。
	if logs == "" {
		fmt.Println("No logs available.")
		return nil
	}

	// Print the logs to stdout.
	// Using Print (not Println) to avoid double newline if logs end with newline.
	//
	// ログをstdoutに出力します。
	// ログが改行で終わる場合の二重改行を避けるためPrint（Printlnではなく）を使用します。
	fmt.Print(logs)
	return nil
}
