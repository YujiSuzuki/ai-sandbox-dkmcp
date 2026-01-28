// Package cli implements the command-line interface for DockMCP.
// It provides commands for starting the MCP server, listing containers,
// viewing logs, executing commands, and client operations for remote access.
//
// cliパッケージはDockMCPのコマンドラインインターフェースを実装します。
// MCPサーバーの起動、コンテナ一覧表示、ログ閲覧、コマンド実行、
// およびリモートアクセス用のクライアント操作のコマンドを提供します。
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is the server version, set at build time via ldflags.
// This variable is populated during the build process using:
// go build -ldflags "-X github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/cli.version=1.0.0"
//
// versionはサーバーのバージョンで、ビルド時にldflagsで設定されます。
// この変数はビルドプロセス中に以下のコマンドで設定されます：
// go build -ldflags "-X github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/cli.version=1.0.0"
var version = "dev"

var (
	// cfgFile holds the path to the configuration file.
	// If empty, the default path (./dkmcp.yaml) will be used.
	//
	// cfgFileは設定ファイルへのパスを保持します。
	// 空の場合、デフォルトパス（./dkmcp.yaml）が使用されます。
	cfgFile string

	// rootCmd is the base command for the DockMCP CLI.
	// All other commands are added as subcommands to this root command.
	//
	// rootCmdはDockMCP CLIの基本コマンドです。
	// 他のすべてのコマンドはこのルートコマンドのサブコマンドとして追加されます。
	rootCmd = &cobra.Command{
		Use:   "dkmcp",
		Short: "DockMCP - Secure Docker container access for AI assistants",
		Long: `DockMCP provides secure access to Docker containers for AI coding assistants
through MCP (Model Context Protocol). It allows AI tools like Claude Code
and Gemini Code Assist to interact with containers while maintaining
security through whitelisting and permission controls.`,
	}
)

// Execute runs the root command and all its subcommands.
// This is the main entry point for the CLI application.
// It handles command parsing, execution, and error reporting.
// If an error occurs, it prints the error to stderr and exits with code 1.
//
// Executeはルートコマンドとそのすべてのサブコマンドを実行します。
// これはCLIアプリケーションのメインエントリーポイントです。
// コマンドの解析、実行、エラー報告を処理します。
// エラーが発生した場合、stderrにエラーを出力し、終了コード1で終了します。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// init registers the global flags for the root command.
// This function is automatically called by Go when the package is imported.
//
// initはルートコマンドのグローバルフラグを登録します。
// この関数はパッケージがインポートされたときにGoによって自動的に呼び出されます。
func init() {
	// Register --config flag that can be used with any subcommand.
	// PersistentFlags are inherited by all subcommands.
	//
	// 任意のサブコマンドで使用できる--configフラグを登録します。
	// PersistentFlagsはすべてのサブコマンドに継承されます。
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./dkmcp.yaml)")
}
