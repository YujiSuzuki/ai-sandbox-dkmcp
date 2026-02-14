// client_lifecycle.go implements the 'client restart/stop/start' subcommands
// for container lifecycle management via Docker API.
//
// client_lifecycle.goはDocker API経由のコンテナライフサイクル管理用の
// 'client restart/stop/start'サブコマンドを実装します。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// clientRestartCmd represents the 'client restart' subcommand.
// clientRestartCmdは'client restart'サブコマンドを表します。
var clientRestartCmd = &cobra.Command{
	Use:   "restart CONTAINER",
	Short: "Restart a container via DockMCP server",
	Long: `Restart a container using Docker API directly (no CLI execution).
Requires lifecycle permission to be enabled in dkmcp.yaml.

Examples:
  dkmcp client restart securenote-api
  dkmcp client restart securenote-api --timeout 30`,
	Args: cobra.ExactArgs(1),
	RunE: runClientRestart,
}

// clientStopCmd represents the 'client stop' subcommand.
// clientStopCmdは'client stop'サブコマンドを表します。
var clientStopCmd = &cobra.Command{
	Use:   "stop CONTAINER",
	Short: "Stop a container via DockMCP server",
	Long: `Stop a running container using Docker API directly (no CLI execution).
Requires lifecycle permission to be enabled in dkmcp.yaml.

Examples:
  dkmcp client stop securenote-api
  dkmcp client stop securenote-api --timeout 30`,
	Args: cobra.ExactArgs(1),
	RunE: runClientStop,
}

// clientStartCmd represents the 'client start' subcommand.
// clientStartCmdは'client start'サブコマンドを表します。
var clientStartCmd = &cobra.Command{
	Use:   "start CONTAINER",
	Short: "Start a container via DockMCP server",
	Long: `Start a stopped container using Docker API directly (no CLI execution).
Requires lifecycle permission to be enabled in dkmcp.yaml.

Examples:
  dkmcp client start securenote-api`,
	Args: cobra.ExactArgs(1),
	RunE: runClientStart,
}

func init() {
	clientCmd.AddCommand(clientRestartCmd)
	clientCmd.AddCommand(clientStopCmd)
	clientCmd.AddCommand(clientStartCmd)

	clientRestartCmd.Flags().Int("timeout", 0, "Timeout in seconds to wait before killing (0 = Docker default)")
	clientStopCmd.Flags().Int("timeout", 0, "Timeout in seconds to wait before killing (0 = Docker default)")
}

// runClientRestart restarts a container via DockMCP server.
// runClientRestartはDockMCPサーバー経由でコンテナを再起動します。
func runClientRestart(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	var timeout *int
	if cmd.Flags().Changed("timeout") {
		v, _ := cmd.Flags().GetInt("timeout")
		timeout = &v
	}

	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	ctx := context.Background()
	result, err := backend.RestartContainer(ctx, containerName, timeout)
	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	if result != "" {
		fmt.Print(result)
	}
	return nil
}

// runClientStop stops a container via DockMCP server.
// runClientStopはDockMCPサーバー経由でコンテナを停止します。
func runClientStop(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	var timeout *int
	if cmd.Flags().Changed("timeout") {
		v, _ := cmd.Flags().GetInt("timeout")
		timeout = &v
	}

	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	ctx := context.Background()
	result, err := backend.StopContainer(ctx, containerName, timeout)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if result != "" {
		fmt.Print(result)
	}
	return nil
}

// runClientStart starts a container via DockMCP server.
// runClientStartはDockMCPサーバー経由でコンテナを起動します。
func runClientStart(cmd *cobra.Command, args []string) error {
	containerName := args[0]

	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	ctx := context.Background()
	result, err := backend.StartContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	if result != "" {
		fmt.Print(result)
	}
	return nil
}
