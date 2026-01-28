// client_commands.go implements the 'client commands' subcommand for listing allowed commands.
// It queries the DockMCP server to show which commands are whitelisted for each container.
//
// client_commands.goは許可されたコマンドを一覧表示する'client commands'サブコマンドを実装します。
// DockMCPサーバーに問い合わせて、各コンテナでホワイトリストに登録されているコマンドを表示します。
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/client"
)

// clientCommandsCmd represents the 'client commands' subcommand.
// It displays the whitelisted commands that can be executed in containers.
// Optionally accepts a container name to show commands for a specific container.
//
// clientCommandsCmdは'client commands'サブコマンドを表します。
// コンテナで実行できるホワイトリストに登録されたコマンドを表示します。
// オプションでコンテナ名を受け取り、特定のコンテナのコマンドを表示できます。
var clientCommandsCmd = &cobra.Command{
	Use:   "commands [container]",
	Short: "List allowed commands for a container",
	Long: `List the whitelisted commands that can be executed in a container.

If no container is specified, shows allowed commands for all containers.
Commands with '*' wildcard match any suffix (e.g., 'echo *' matches 'echo hello').`,
	RunE: runClientCommands,
}

// init registers the commands subcommand with the client command.
// This function is automatically called when the package is imported.
//
// initはcommandsサブコマンドをclientコマンドに登録します。
// この関数はパッケージがインポートされたときに自動的に呼び出されます。
func init() {
	clientCmd.AddCommand(clientCommandsCmd)
}

// runClientCommands is the execution function for the commands subcommand.
// It connects to the DockMCP server and retrieves the allowed commands list.
//
// runClientCommandsはcommandsサブコマンドの実行関数です。
// DockMCPサーバーに接続し、許可されたコマンドリストを取得します。
func runClientCommands(cmd *cobra.Command, args []string) error {
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

	// Prepare arguments for the get_allowed_commands tool.
	// Include container name if specified as argument.
	//
	// get_allowed_commandsツールの引数を準備します。
	// 引数として指定された場合はコンテナ名を含めます。
	toolArgs := map[string]interface{}{}
	if len(args) > 0 {
		toolArgs["container"] = args[0]
	}

	// Call get_allowed_commands tool via MCP.
	// MCP経由でget_allowed_commandsツールを呼び出します。
	resp, err := c.CallTool("get_allowed_commands", toolArgs)
	if err != nil {
		return fmt.Errorf("failed to get allowed commands: %w", err)
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
			Container            string   `json:"container"`              // Container name / コンテナ名
			AllowedCommands      []string `json:"allowed_commands"`       // List of allowed commands / 許可されたコマンドのリスト
			DangerousCommands    []string `json:"dangerous_commands"`     // List of dangerous commands / 危険コマンドのリスト
			DangerousModeEnabled bool     `json:"dangerous_mode_enabled"` // Whether dangerous mode is enabled / 危険モードが有効かどうか
			Note                 string   `json:"note"`                   // Additional note / 追加の注記
		}

		// Parse the JSON response.
		// JSONレスポンスを解析します。
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Display the container header.
		// コンテナヘッダーを表示します。
		fmt.Printf("Allowed commands for container '%s':\n\n", result.Container)

		// Display the commands or indicate none are allowed.
		// コマンドを表示するか、許可されていないことを示します。
		if len(result.AllowedCommands) == 0 {
			fmt.Println("  (no commands allowed)")
		} else {
			for _, cmd := range result.AllowedCommands {
				fmt.Printf("  - %s\n", cmd)
			}
		}

		// Display dangerous commands if enabled.
		// 危険モードが有効な場合、危険コマンドを表示します。
		if result.DangerousModeEnabled {
			fmt.Printf("\nDangerous commands (requires dangerously=true):\n")
			if len(result.DangerousCommands) == 0 {
				fmt.Println("  (no dangerous commands configured)")
			} else {
				for _, cmd := range result.DangerousCommands {
					fmt.Printf("  - %s\n", cmd)
				}
			}
		}

		// Display the note about wildcard matching.
		// ワイルドカードマッチングに関する注記を表示します。
		fmt.Printf("\nNote: %s\n", result.Note)
	} else {
		// All containers response format.
		// 全コンテナのレスポンス形式。
		var result struct {
			Containers           map[string][]string `json:"containers"`             // Map of container to commands / コンテナからコマンドへのマップ
			DangerousContainers  map[string][]string `json:"dangerous_containers"`   // Map of container to dangerous commands / コンテナから危険コマンドへのマップ
			DangerousModeEnabled bool                `json:"dangerous_mode_enabled"` // Whether dangerous mode is enabled / 危険モードが有効かどうか
			Note                 string              `json:"note"`                   // Additional note / 追加の注記
		}

		// Parse the JSON response.
		// JSONレスポンスを解析します。
		if err := json.Unmarshal([]byte(resp.Content[0].Text), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Handle case where no whitelists are configured.
		// ホワイトリストが設定されていない場合を処理します。
		if len(result.Containers) == 0 && len(result.DangerousContainers) == 0 {
			fmt.Println("No command whitelists configured.")
			return nil
		}

		// Display in table format using tabwriter.
		// tabwriterを使用してテーブル形式で表示します。
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CONTAINER\tALLOWED COMMANDS")
		fmt.Fprintln(w, "---------\t----------------")

		// Iterate through each container and its commands.
		// 各コンテナとそのコマンドを反復処理します。
		for container, commands := range result.Containers {
			// Special display for wildcard container.
			// ワイルドカードコンテナの特別な表示。
			displayName := container
			if container == "*" {
				displayName = "* (all containers)"
			}

			// Handle containers with no commands.
			// コマンドがないコンテナを処理します。
			if len(commands) == 0 {
				fmt.Fprintf(w, "%s\t(none)\n", displayName)
			} else {
				// Print first command with container name, subsequent commands indented.
				// 最初のコマンドはコンテナ名とともに、後続のコマンドはインデントして出力します。
				for i, cmd := range commands {
					if i == 0 {
						fmt.Fprintf(w, "%s\t%s\n", displayName, cmd)
					} else {
						fmt.Fprintf(w, "\t%s\n", cmd)
					}
				}
			}
		}

		// Flush the tabwriter to ensure all output is written.
		// すべての出力が書き込まれるようにtabwriterをフラッシュします。
		w.Flush()

		// Display dangerous commands if enabled.
		// 危険モードが有効な場合、危険コマンドを表示します。
		if result.DangerousModeEnabled && len(result.DangerousContainers) > 0 {
			fmt.Printf("\n")
			w2 := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w2, "CONTAINER\tDANGEROUS COMMANDS (requires dangerously=true)")
			fmt.Fprintln(w2, "---------\t----------------------------------------------")

			for container, commands := range result.DangerousContainers {
				displayName := container
				if container == "*" {
					displayName = "* (all containers)"
				}

				if len(commands) == 0 {
					fmt.Fprintf(w2, "%s\t(none)\n", displayName)
				} else {
					for i, cmd := range commands {
						if i == 0 {
							fmt.Fprintf(w2, "%s\t%s\n", displayName, cmd)
						} else {
							fmt.Fprintf(w2, "\t%s\n", cmd)
						}
					}
				}
			}
			w2.Flush()
		}

		// Display the note about wildcard matching.
		// ワイルドカードマッチングに関する注記を表示します。
		fmt.Printf("\nNote: %s\n", result.Note)
	}

	return nil
}
