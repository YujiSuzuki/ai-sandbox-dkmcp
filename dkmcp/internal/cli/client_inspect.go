// client_inspect.go implements the 'client inspect' subcommand for retrieving container details via HTTP.
// It uses HTTPBackend to communicate with the DockMCP server instead of direct Docker access.
//
// client_inspect.goはHTTP経由でコンテナ詳細を取得する'client inspect'サブコマンドを実装します。
// 直接Docker接続の代わりにHTTPBackendを使用してDockMCPサーバーと通信します。
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

// extractJSONFromMarkdown extracts JSON content from a Markdown code block.
// The MCP server returns responses wrapped in Markdown format like:
// Container 'name' details:
//
// ```json
// {...}
// ```
//
// extractJSONFromMarkdownはMarkdownコードブロックからJSONコンテンツを抽出します。
// MCPサーバーはMarkdown形式でラップされたレスポンスを返します。
func extractJSONFromMarkdown(content string) string {
	// Look for ```json ... ``` block
	// ```json ... ``` ブロックを探す
	startMarker := "```json\n"
	endMarker := "\n```"

	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		// No JSON code block found, return original content
		// JSONコードブロックが見つからない場合、元のコンテンツを返す
		return content
	}

	// Move past the start marker
	// 開始マーカーを過ぎた位置に移動
	jsonStart := startIdx + len(startMarker)

	// Find the end marker after the start
	// 開始後の終了マーカーを見つける
	endIdx := strings.Index(content[jsonStart:], endMarker)
	if endIdx == -1 {
		// No closing marker found, return from start to end
		// 終了マーカーが見つからない場合、開始から終わりまで返す
		return content[jsonStart:]
	}

	return content[jsonStart : jsonStart+endIdx]
}

// clientInspectJSONFlag controls whether to output JSON instead of summary.
// clientInspectJSONFlagはサマリーの代わりにJSONを出力するかどうかを制御します。
var clientInspectJSONFlag bool

// clientInspectCmd represents the 'client inspect' subcommand.
// It retrieves and displays detailed container information via the DockMCP HTTP server.
// Requires exactly one argument: the container name.
//
// clientInspectCmdは'client inspect'サブコマンドを表します。
// DockMCP HTTPサーバー経由でコンテナの詳細情報を取得して表示します。
// コンテナ名という1つの引数が必要です。
var clientInspectCmd = &cobra.Command{
	Use:   "inspect CONTAINER",
	Short: "Get detailed information about a container via DockMCP server",
	Long: `Retrieve detailed information about a Docker container including configuration,
network settings, and mount information through the DockMCP server.

By default, shows a human-readable summary. Use --json for full JSON output.

Examples:
  dkmcp client inspect securenote-api         # Show summary (default)
  dkmcp client inspect securenote-api --json  # Show full JSON
  dkmcp client inspect --url http://host.docker.internal:8080 securenote-api`,
	Args: cobra.ExactArgs(1),
	RunE: runClientInspect,
}

// init registers the inspect subcommand with the client command.
// This function is automatically called when the package is imported.
//
// initはinspectサブコマンドをclientコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	// Add inspect as a subcommand of client.
	// inspectをclientのサブコマンドとして追加します。
	clientCmd.AddCommand(clientInspectCmd)

	// Add --json flag for full JSON output.
	// フルJSON出力用の--jsonフラグを追加します。
	clientInspectCmd.Flags().BoolVar(&clientInspectJSONFlag, "json", false, "Output full JSON instead of summary")
}

// runClientInspect is the execution function for the client inspect subcommand.
// It creates an HTTPBackend, retrieves container details, and displays them.
//
// runClientInspectはclient inspectサブコマンドの実行関数です。
// HTTPBackendを作成し、コンテナ詳細を取得して表示します。
func runClientInspect(cmd *cobra.Command, args []string) error {
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

	// Retrieve detailed information from the container via MCP.
	// MCP経由でコンテナから詳細情報を取得します。
	ctx := context.Background()
	infoJSON, err := backend.InspectContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// Handle empty response.
	// 空のレスポンスを処理します。
	if infoJSON == "" {
		fmt.Println("No container information available.")
		return nil
	}

	// Extract JSON from Markdown code block (MCP server wraps response in Markdown)
	// MarkdownコードブロックからJSONを抽出（MCPサーバーはレスポンスをMarkdownでラップする）
	jsonContent := extractJSONFromMarkdown(infoJSON)

	// Output based on --json flag
	// --jsonフラグに基づいて出力
	if clientInspectJSONFlag {
		// JSON output mode - print extracted JSON
		// JSON出力モード - 抽出したJSONを出力
		fmt.Println(jsonContent)
	} else {
		// Summary output mode (default) - parse JSON and show summary
		// サマリー出力モード（デフォルト） - JSONをパースしてサマリーを表示
		var info types.ContainerJSON
		if err := json.Unmarshal([]byte(jsonContent), &info); err != nil {
			// If parsing fails, fall back to JSON output
			// パースに失敗した場合、JSON出力にフォールバック
			fmt.Printf("Container '%s' details (JSON parse failed, showing raw):\n\n", containerName)
			fmt.Println(jsonContent)
			return nil
		}
		printInspectSummary(containerName, &info)
	}

	return nil
}
