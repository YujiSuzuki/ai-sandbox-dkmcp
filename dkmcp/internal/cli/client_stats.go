// client_stats.go implements the 'client stats' subcommand for retrieving container statistics via HTTP.
// It uses HTTPBackend to communicate with the DockMCP server instead of direct Docker access.
//
// client_stats.goはHTTP経由でコンテナ統計を取得する'client stats'サブコマンドを実装します。
// 直接Docker接続の代わりにHTTPBackendを使用してDockMCPサーバーと通信します。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// clientStatsCmd represents the 'client stats' subcommand.
// It retrieves and displays container resource statistics via the DockMCP HTTP server.
// Requires exactly one argument: the container name.
//
// clientStatsCmdは'client stats'サブコマンドを表します。
// DockMCP HTTPサーバー経由でコンテナのリソース統計を取得して表示します。
// コンテナ名という1つの引数が必要です。
var clientStatsCmd = &cobra.Command{
	Use:   "stats CONTAINER",
	Short: "Get resource statistics from a container via DockMCP server",
	Long: `Retrieve resource statistics (CPU, memory, network, disk I/O) from a Docker container
through the DockMCP server.

Examples:
  dkmcp client stats securenote-api
  dkmcp client stats --url http://host.docker.internal:8080 securenote-api`,
	Args: cobra.ExactArgs(1),
	RunE: runClientStats,
}

// init registers the stats subcommand with the client command.
// This function is automatically called when the package is imported.
//
// initはstatsサブコマンドをclientコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	// Add stats as a subcommand of client.
	// statsをclientのサブコマンドとして追加します。
	clientCmd.AddCommand(clientStatsCmd)
}

// runClientStats is the execution function for the client stats subcommand.
// It creates an HTTPBackend, retrieves stats, and displays them.
//
// runClientStatsはclient statsサブコマンドの実行関数です。
// HTTPBackendを作成し、統計を取得して表示します。
func runClientStats(cmd *cobra.Command, args []string) error {
	// Get the container name from command arguments.
	// コマンド引数からコンテナ名を取得します。
	containerName := args[0]

	// Create an HTTPBackend for remote DockMCP server access.
	// リモートDockMCPサーバーアクセス用のHTTPBackendを作成します。
	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	// Retrieve stats from the container via MCP.
	// MCP経由でコンテナから統計を取得します。
	ctx := context.Background()
	stats, err := backend.GetStats(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	// Handle empty response.
	// 空のレスポンスを処理します。
	if stats == "" {
		fmt.Println("No stats available.")
		return nil
	}

	// Print the statistics header.
	// 統計ヘッダーを出力します。
	fmt.Printf("Resource statistics for container '%s':\n\n", containerName)

	// Print the stats (already JSON formatted from server).
	// 統計を出力（サーバーからすでにJSON形式）。
	fmt.Println(stats)
	return nil
}
