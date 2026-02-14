// client_host_exec.go implements the 'client host-exec' subcommand for host command execution via HTTP.
// It uses HTTPBackend to communicate with the DockMCP server for executing whitelisted host CLI commands.
//
// client_host_exec.goはHTTP経由でホストコマンドを実行する'client host-exec'サブコマンドを実装します。
// HTTPBackendを使用してDockMCPサーバーと通信し、ホワイトリストに登録されたホストCLIコマンドを実行します。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// clientHostExecCmd represents the 'client host-exec' subcommand.
// It executes a whitelisted CLI command on the host OS via the DockMCP server.
//
// clientHostExecCmdは'client host-exec'サブコマンドを表します。
// DockMCPサーバー経由でホストOS上のホワイトリストに登録されたCLIコマンドを実行します。
var clientHostExecCmd = &cobra.Command{
	Use:   `host-exec COMMAND`,
	Short: "Execute a whitelisted command on the host OS via DockMCP server",
	Long: `Execute a whitelisted CLI command on the host OS through the DockMCP server.
Commands must be configured in the host_commands whitelist.
Use --dangerously for commands in the dangerously list.

Examples:
  dkmcp client host-exec "df -h"
  dkmcp client host-exec "lsof -i :8080"
  dkmcp client host-exec --dangerously "kill 12345"`,
	Args: cobra.ExactArgs(1),
	RunE: runClientHostExec,
}

func init() {
	clientCmd.AddCommand(clientHostExecCmd)
	clientHostExecCmd.Flags().Bool("dangerously", false,
		"Enable dangerous mode to execute commands from the dangerously list")
}

// runClientHostExec executes a host command via DockMCP server.
// runClientHostExecはDockMCPサーバー経由でホストコマンドを実行します。
func runClientHostExec(cmd *cobra.Command, args []string) error {
	command := args[0]
	dangerously, _ := cmd.Flags().GetBool("dangerously")

	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	ctx := context.Background()
	result, err := backend.ExecHostCommand(ctx, command, dangerously)
	if err != nil {
		return fmt.Errorf("failed to execute host command: %w", err)
	}

	if result != "" {
		fmt.Print(result)
	}

	return nil
}
