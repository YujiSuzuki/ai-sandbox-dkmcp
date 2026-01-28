// client_blocked.go implements the 'client blocked-paths' subcommand for listing blocked file paths.
// It queries the DockMCP server to show which file paths are protected from AI access.
// This is part of the security feature that hides sensitive files like secrets and .env.
//
// client_blocked.goはブロックされたファイルパスを一覧表示する'client blocked-paths'サブコマンドを実装します。
// DockMCPサーバーに問い合わせて、AIアクセスから保護されているファイルパスを表示します。
// これはsecretsや.envのような機密ファイルを隠すセキュリティ機能の一部です。
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/client"
)

// clientBlockedCmd represents the 'client blocked-paths' subcommand.
// It displays the file paths that are blocked from access in containers.
// Optionally accepts a container name to show blocked paths for a specific container.
//
// clientBlockedCmdは'client blocked-paths'サブコマンドを表します。
// コンテナ内でアクセスがブロックされているファイルパスを表示します。
// オプションでコンテナ名を受け取り、特定のコンテナのブロックパスを表示できます。
var clientBlockedCmd = &cobra.Command{
	Use:   "blocked-paths [container]",
	Short: "List blocked file paths for a container",
	Long: `List the file paths that are blocked from access in a container.

If no container is specified, shows blocked paths for all containers.
Blocked paths protect sensitive files (secrets, .env, etc.) from AI access.`,
	RunE: runClientBlocked,
}

// init registers the blocked-paths subcommand with the client command.
// This function is automatically called when the package is imported.
//
// initはblocked-pathsサブコマンドをclientコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	clientCmd.AddCommand(clientBlockedCmd)
}

// runClientBlocked is the execution function for the blocked-paths subcommand.
// It connects to the DockMCP server and retrieves the blocked paths list.
//
// runClientBlockedはblocked-pathsサブコマンドの実行関数です。
// DockMCPサーバーに接続し、ブロックパスリストを取得します。
func runClientBlocked(cmd *cobra.Command, args []string) error {
	// Create client for DockMCP server communication.
	// DockMCPサーバー通信用のクライアントを作成します。
	c := client.NewClient(serverURL)
	if clientSuffix != "" {
		c.SetClientSuffix(clientSuffix)
	}
	defer c.Close()

	// Perform health check to verify server is running.
	// サーバーが実行中であることを確認するためにヘルスチェックを実行します。
	if err := c.HealthCheck(); err != nil {
		return fmt.Errorf("server health check failed: %w", err)
	}

	// Establish SSE connection for MCP communication.
	// MCP通信用のSSE接続を確立します。
	if err := c.Connect(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Prepare arguments for the get_blocked_paths tool.
	// Include container name if specified as argument.
	//
	// get_blocked_pathsツールの引数を準備します。
	// 引数として指定された場合はコンテナ名を含めます。
	toolArgs := map[string]interface{}{}
	if len(args) > 0 {
		toolArgs["container"] = args[0]
	}

	// Call get_blocked_paths tool via MCP.
	// MCP経由でget_blocked_pathsツールを呼び出します。
	resp, err := c.CallTool("get_blocked_paths", toolArgs)
	if err != nil {
		return fmt.Errorf("failed to get blocked paths: %w", err)
	}

	// Handle empty response.
	// 空のレスポンスを処理します。
	if len(resp.Content) == 0 {
		fmt.Println("No response from server.")
		return nil
	}

	// Handle response based on whether a specific container was requested.
	// 特定のコンテナが要求されたかどうかに基づいてレスポンスを処理します。
	if len(args) > 0 {
		// Single container response format.
		// 単一コンテナのレスポンス形式。
		var result struct {
			Container    string        `json:"container"`     // Container name / コンテナ名
			BlockedPaths []blockedPath `json:"blocked_paths"` // List of blocked paths / ブロックパスのリスト
		}

		// Parse the JSON response.
		// JSONレスポンスを解析します。
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Display the container header and blocked paths.
		// コンテナヘッダーとブロックパスを表示します。
		fmt.Printf("Blocked paths for container '%s':\n\n", result.Container)
		printBlockedPaths(result.BlockedPaths)
	} else {
		// All containers response format.
		// 全コンテナのレスポンス形式。
		var result struct {
			AllBlockedPaths []blockedPath `json:"all_blocked_paths"` // List of all blocked paths / すべてのブロックパスのリスト
		}

		// Parse the JSON response.
		// JSONレスポンスを解析します。
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Handle case where no blocked paths are configured.
		// ブロックパスが設定されていない場合を処理します。
		if len(result.AllBlockedPaths) == 0 {
			fmt.Println("No blocked paths configured.")
			fmt.Println("\nTip: Enable auto_import in dkmcp.yaml to automatically import blocked paths from DevContainer configs.")
			return nil
		}

		// Display all blocked paths.
		// すべてのブロックパスを表示します。
		fmt.Println("All blocked paths:")
		printBlockedPaths(result.AllBlockedPaths)
	}

	return nil
}

// blockedPath represents a blocked file path entry with metadata.
// It contains information about why the path is blocked and its source.
//
// blockedPathはメタデータを持つブロックされたファイルパスエントリを表します。
// パスがブロックされている理由とそのソースに関する情報を含みます。
type blockedPath struct {
	Container    string `json:"container"`               // Container this applies to / これが適用されるコンテナ
	Pattern      string `json:"pattern"`                 // Path pattern (may include globs) / パスパターン（グロブを含む場合あり）
	Reason       string `json:"reason"`                  // Human-readable reason for blocking / ブロックの人が読める理由
	Source       string `json:"source,omitempty"`        // Where this rule came from / このルールの出所
	OriginalPath string `json:"original_path,omitempty"` // Original path if transformed / 変換された場合の元のパス
}

// printBlockedPaths displays blocked paths in a formatted table.
// It uses tabwriter for aligned column output.
//
// printBlockedPathsはブロックパスをフォーマットされたテーブルで表示します。
// 列を揃えた出力のためにtabwriterを使用します。
func printBlockedPaths(paths []blockedPath) {
	// Handle case of no blocked paths.
	// ブロックパスがない場合を処理します。
	if len(paths) == 0 {
		fmt.Println("  (no blocked paths)")
		return
	}

	// Display in table format using tabwriter.
	// tabwriterを使用してテーブル形式で表示します。
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print table header.
	// テーブルヘッダーを出力します。
	fmt.Fprintln(w, "CONTAINER\tPATTERN\tREASON\tSOURCE")
	fmt.Fprintln(w, "---------\t-------\t------\t------")

	// Print each blocked path entry.
	// 各ブロックパスエントリを出力します。
	for _, p := range paths {
		// Special display for wildcard container.
		// ワイルドカードコンテナの特別な表示。
		container := p.Container
		if container == "*" {
			container = "* (all)"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			container, p.Pattern, p.Reason, p.Source)
	}

	// Flush the tabwriter to ensure all output is written.
	// すべての出力が書き込まれるようにtabwriterをフラッシュします。
	w.Flush()
}
