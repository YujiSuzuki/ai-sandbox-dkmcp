// client_list.go implements the 'client list' subcommand for listing containers via HTTP.
// It uses HTTPBackend to communicate with the DockMCP server instead of direct Docker access.
//
// client_list.goはHTTP経由でコンテナを一覧表示する'client list'サブコマンドを実装します。
// 直接Docker接続の代わりにHTTPBackendを使用してDockMCPサーバーと通信します。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// clientListCmd represents the 'client list' subcommand.
// It retrieves and displays accessible containers via the DockMCP HTTP server.
// This is useful when running in environments without Docker socket access.
//
// clientListCmdは'client list'サブコマンドを表します。
// DockMCP HTTPサーバー経由でアクセス可能なコンテナを取得して表示します。
// Dockerソケットへのアクセスがない環境で実行する場合に便利です。
var clientListCmd = &cobra.Command{
	Use:   "list",
	Short: "List accessible containers via DockMCP server",
	Long:  `List all Docker containers that are accessible through the DockMCP server.`,
	RunE:  runClientList,
}

// init registers the list subcommand with the client command.
// This function is automatically called when the package is imported.
//
// initはlistサブコマンドをclientコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	clientCmd.AddCommand(clientListCmd)
}

// runClientList is the execution function for the client list subcommand.
// It creates an HTTPBackend, retrieves the container list, and displays it in a table.
//
// runClientListはclient listサブコマンドの実行関数です。
// HTTPBackendを作成し、コンテナリストを取得して、テーブル形式で表示します。
func runClientList(cmd *cobra.Command, args []string) error {
	// Create an HTTPBackend for remote DockMCP server access.
	// serverURL is inherited from the parent client command's PersistentFlags.
	//
	// リモートDockMCPサーバーアクセス用のHTTPBackendを作成します。
	// serverURLは親のclientコマンドのPersistentFlagsから継承されます。
	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	// Retrieve the list of accessible containers via MCP.
	// MCP経由でアクセス可能なコンテナのリストを取得します。
	ctx := context.Background()
	containers, err := backend.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Display the containers in a formatted table.
	// Uses the same printContainerTable function as the direct list command.
	//
	// コンテナをフォーマットされたテーブルで表示します。
	// 直接のlistコマンドと同じprintContainerTable関数を使用します。
	return printContainerTable(containers)
}
