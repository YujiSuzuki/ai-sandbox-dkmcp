// client_host_tools.go implements the 'client host-tools' subcommands for host tool operations via HTTP.
// It uses HTTPBackend to communicate with the DockMCP server for listing, inspecting, and running host tools.
//
// client_host_tools.goはHTTP経由でホストツール操作を行う'client host-tools'サブコマンドを実装します。
// HTTPBackendを使用してDockMCPサーバーと通信し、ホストツールの一覧表示、詳細取得、実行を行います。
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// clientHostToolsCmd is the parent command for host tool subcommands.
// clientHostToolsCmdはホストツールサブコマンドの親コマンドです。
var clientHostToolsCmd = &cobra.Command{
	Use:   "host-tools",
	Short: "Host tool commands via DockMCP server",
	Long:  `Commands for discovering and running host tools through the DockMCP server.`,
}

// clientHostToolsListCmd lists available host tools.
// clientHostToolsListCmdは利用可能なホストツールを一覧表示します。
var clientHostToolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available host tools",
	Long:  `List all host tools available through the DockMCP server.`,
	RunE:  runClientHostToolsList,
}

// clientHostToolsInfoCmd gets detailed info about a host tool.
// clientHostToolsInfoCmdはホストツールの詳細情報を取得します。
var clientHostToolsInfoCmd = &cobra.Command{
	Use:   "info NAME",
	Short: "Get detailed info about a host tool",
	Long:  `Get detailed information about a specific host tool including usage and examples.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runClientHostToolsInfo,
}

// clientHostToolsRunCmd executes a host tool.
// clientHostToolsRunCmdはホストツールを実行します。
var clientHostToolsRunCmd = &cobra.Command{
	Use:   "run NAME [ARGS...]",
	Short: "Execute a host tool",
	Long:  `Execute a host tool with optional arguments through the DockMCP server.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runClientHostToolsRun,
}

func init() {
	clientCmd.AddCommand(clientHostToolsCmd)
	clientHostToolsCmd.AddCommand(clientHostToolsListCmd)
	clientHostToolsCmd.AddCommand(clientHostToolsInfoCmd)
	clientHostToolsCmd.AddCommand(clientHostToolsRunCmd)
}

// runClientHostToolsList lists available host tools via DockMCP server.
// runClientHostToolsListはDockMCPサーバー経由で利用可能なホストツールを一覧表示します。
func runClientHostToolsList(cmd *cobra.Command, args []string) error {
	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	ctx := context.Background()
	result, err := backend.ListHostTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list host tools: %w", err)
	}

	if result == "" {
		fmt.Println("No host tools available.")
		return nil
	}

	fmt.Println(result)
	return nil
}

// runClientHostToolsInfo gets detailed info about a host tool via DockMCP server.
// runClientHostToolsInfoはDockMCPサーバー経由でホストツールの詳細情報を取得します。
func runClientHostToolsInfo(cmd *cobra.Command, args []string) error {
	name := args[0]

	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	ctx := context.Background()
	result, err := backend.GetHostToolInfo(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get host tool info: %w", err)
	}

	if result == "" {
		fmt.Printf("No info available for tool: %s\n", name)
		return nil
	}

	fmt.Println(result)
	return nil
}

// runClientHostToolsRun executes a host tool via DockMCP server.
// runClientHostToolsRunはDockMCPサーバー経由でホストツールを実行します。
func runClientHostToolsRun(cmd *cobra.Command, args []string) error {
	name := args[0]
	var toolArgs []string
	if len(args) > 1 {
		toolArgs = args[1:]
	}

	backend, err := NewHTTPBackend(serverURL)
	if err != nil {
		return err
	}
	defer backend.Close()

	ctx := context.Background()
	result, err := backend.RunHostTool(ctx, name, toolArgs)
	if err != nil {
		return fmt.Errorf("failed to run host tool: %w", err)
	}

	if result != "" {
		fmt.Print(result)
	}

	return nil
}
