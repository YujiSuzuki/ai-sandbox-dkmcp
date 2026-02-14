// tools.go implements the 'tools' command group for managing host tools.
// Provides subcommands for syncing tools from staging to approved directory.
//
// tools.goはホストツール管理用の'tools'コマンドグループを実装します。
// ステージングから承認済みディレクトリへのツール同期のサブコマンドを提供します。
package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/hosttools"
)

// toolsCmd is the parent command for tool management subcommands.
// toolsCmdはツール管理サブコマンドの親コマンドです。
var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage host tools",
	Long:  `Manage host tools: sync staging tools to approved directory, list tools, etc.`,
}

// toolsSyncCmd syncs host tools from staging directories to the approved directory.
// toolsSyncCmdはステージングディレクトリから承認済みディレクトリにホストツールを同期します。
var toolsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync host tools from staging to approved directory",
	Long: `Compare tools in staging directories (workspace) with approved tools,
and interactively approve new or updated tools for execution.

This command requires host_tools.approved_dir to be configured in dkmcp.yaml.
Tools in staging directories are proposals; only approved tools are executed.`,
	RunE: runToolsSync,
}

// toolsListCmd lists the approved directory path and project ID.
// toolsListCmdは承認済みディレクトリパスとプロジェクトIDを表示します。
var toolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show approved tools and directories",
	Long:  `Show the approved tools directory, project ID, and list all approved tools.`,
	RunE:  runToolsList,
}

// flagToolsWorkspace overrides workspace_root for tools commands.
var flagToolsWorkspace string

func init() {
	rootCmd.AddCommand(toolsCmd)
	toolsCmd.AddCommand(toolsSyncCmd)
	toolsCmd.AddCommand(toolsListCmd)

	toolsCmd.PersistentFlags().StringVar(&flagToolsWorkspace, "workspace", "", "Workspace root directory (overrides config)")
}

// runToolsSync performs an interactive sync of host tools.
// runToolsSyncはホストツールのインタラクティブな同期を実行します。
func runToolsSync(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.HostAccess.HostTools.Enabled {
		return fmt.Errorf("host_tools is not enabled in configuration")
	}

	if !cfg.HostAccess.HostTools.IsSecureMode() {
		return fmt.Errorf("host_tools.approved_dir is not configured; sync requires secure mode")
	}

	workspaceRoot := cfg.HostAccess.WorkspaceRoot
	if flagToolsWorkspace != "" {
		workspaceRoot = flagToolsWorkspace
	}
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	absPath, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return fmt.Errorf("resolving workspace path: %w", err)
	}
	workspaceRoot = absPath

	// Set up minimal logging
	handler := NewColoredHandler(os.Stdout, slog.LevelInfo)
	slog.SetDefault(slog.New(handler))

	syncMgr := hosttools.NewSyncManager(&cfg.HostAccess.HostTools, workspaceRoot)
	synced, err := syncMgr.RunInteractiveSync()
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	if synced > 0 {
		fmt.Printf("\n✅ %d tool(s) synced successfully.\n", synced)
	}

	return nil
}

// runToolsList shows approved tools information.
// runToolsListは承認済みツール情報を表示します。
func runToolsList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.HostAccess.HostTools.Enabled {
		return fmt.Errorf("host_tools is not enabled in configuration")
	}

	workspaceRoot := cfg.HostAccess.WorkspaceRoot
	if flagToolsWorkspace != "" {
		workspaceRoot = flagToolsWorkspace
	}
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	absPath, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return fmt.Errorf("resolving workspace path: %w", err)
	}
	workspaceRoot = absPath

	if cfg.HostAccess.HostTools.IsSecureMode() {
		projectID := hosttools.ProjectID(workspaceRoot)
		projectDir, _ := hosttools.ProjectApprovedDir(cfg.HostAccess.HostTools.ApprovedDir, workspaceRoot)
		commonDir, _ := hosttools.CommonApprovedDir(cfg.HostAccess.HostTools.ApprovedDir)

		fmt.Printf("Mode:         secure\n")
		fmt.Printf("Project ID:   %s\n", projectID)
		fmt.Printf("Workspace:    %s\n", workspaceRoot)
		fmt.Printf("Approved dir: %s\n", projectDir)
		if cfg.HostAccess.HostTools.Common {
			fmt.Printf("Common dir:   %s\n", commonDir)
		}
	} else {
		fmt.Printf("Mode:         legacy\n")
		fmt.Printf("Workspace:    %s\n", workspaceRoot)
		fmt.Printf("Directories:  %v\n", cfg.HostAccess.HostTools.Directories)
	}

	fmt.Println()

	// List tools
	mgr := hosttools.NewManager(&cfg.HostAccess.HostTools, workspaceRoot)
	tools, err := mgr.ListTools()
	if err != nil {
		fmt.Printf("No tools found: %v\n", err)
		return nil
	}

	if len(tools) == 0 {
		fmt.Println("No tools found.")
		return nil
	}

	fmt.Printf("Tools (%d):\n", len(tools))
	for _, tool := range tools {
		desc := tool.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Printf("  %-30s %s\n", tool.Name, desc)
	}

	return nil
}
