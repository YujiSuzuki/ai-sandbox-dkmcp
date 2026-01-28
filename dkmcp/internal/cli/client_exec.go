// client_exec.go implements the 'client exec' subcommand for executing commands via HTTP.
// It uses HTTPBackend to communicate with the DockMCP server instead of direct Docker access.
// Commands must be whitelisted on the server side to be executed.
//
// client_exec.goはHTTP経由でコマンドを実行する'client exec'サブコマンドを実装します。
// 直接Docker接続の代わりにHTTPBackendを使用してDockMCPサーバーと通信します。
// コマンドは実行するためにサーバー側でホワイトリストに登録されている必要があります。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// clientExecCmd represents the 'client exec' subcommand.
// It executes a whitelisted command in a container via the DockMCP HTTP server.
// Requires exactly two arguments: container name and command to execute.
//
// clientExecCmdは'client exec'サブコマンドを表します。
// DockMCP HTTPサーバー経由でホワイトリストに登録されたコマンドをコンテナ内で実行します。
// コンテナ名と実行するコマンドの2つの引数が必要です。
var clientExecCmd = &cobra.Command{
	Use:   "exec CONTAINER COMMAND",
	Short: "Execute a command in a container via DockMCP server",
	Long:  `Execute a whitelisted command in a Docker container through the DockMCP server.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runClientExec,
}

// init registers the exec subcommand with the client command.
// This function is automatically called when the package is imported.
//
// initはexecサブコマンドをclientコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	clientCmd.AddCommand(clientExecCmd)

	// Add --dangerously flag for dangerous mode execution
	// 危険モード実行用の--dangerouslyフラグを追加
	clientExecCmd.Flags().Bool("dangerously", false, "Enable dangerous mode to execute commands from exec_dangerously list (file paths are still checked against blocked_paths)")
}

// runClientExec is the execution function for the client exec subcommand.
// It creates an HTTPBackend, executes the command, and displays the result.
//
// runClientExecはclient execサブコマンドの実行関数です。
// HTTPBackendを作成し、コマンドを実行して、結果を表示します。
func runClientExec(cmd *cobra.Command, args []string) error {
	// Get container name and command from arguments.
	// Note: Unlike the direct exec command, this only accepts a single command argument.
	// For commands with spaces, they should be quoted: dkmcp client exec myapp "npm test"
	//
	// 引数からコンテナ名とコマンドを取得します。
	// 注意：直接のexecコマンドとは異なり、これは単一のコマンド引数のみを受け付けます。
	// スペースを含むコマンドはクォートする必要があります：dkmcp client exec myapp "npm test"
	containerName := args[0]
	command := args[1]

	// Get the dangerously flag value.
	// dangerouslyフラグの値を取得します。
	dangerously, _ := cmd.Flags().GetBool("dangerously")

	// Create an HTTPBackend for remote DockMCP server access.
	// リモートDockMCPサーバーアクセス用のHTTPBackendを作成します。
	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	// Execute the command in the container via MCP.
	// The server will validate that the command is whitelisted.
	// If dangerously is true, exec_dangerously commands are allowed.
	//
	// MCP経由でコンテナ内でコマンドを実行します。
	// サーバーはコマンドがホワイトリストに登録されているかを検証します。
	// dangerouslyがtrueの場合、exec_dangerouslyコマンドが許可されます。
	ctx := context.Background()
	result, err := backend.Exec(ctx, containerName, command, dangerously)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Print the command output.
	// Using Print (not Println) to avoid double newline if output ends with newline.
	//
	// コマンド出力を表示します。
	// 出力が改行で終わる場合の二重改行を避けるためPrint（Printlnではなく）を使用します。
	if result.Output != "" {
		fmt.Print(result.Output)
	}

	// Return error if command exited with non-zero exit code.
	// This ensures the CLI exits with non-zero status for failed commands.
	//
	// コマンドが非ゼロの終了コードで終了した場合はエラーを返します。
	// これにより、失敗したコマンドに対してCLIが非ゼロステータスで終了します。
	if result.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d", result.ExitCode)
	}

	return nil
}
